package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

// FailedLoginTracker tracks per-account failed login attempts for brute-force protection.
// The default in-memory implementation is suitable for single-instance deployments;
// multi-instance deployments should use a Redis-backed implementation.
type FailedLoginTracker interface {
	// Check returns ErrAccountLocked if the account is currently locked.
	Check(ctx context.Context, loginID string) error
	// RecordFailed increments the failed-attempt counter for loginID.
	RecordFailed(ctx context.Context, loginID string)
	// Clear resets the counter for loginID (called on successful login).
	Clear(ctx context.Context, loginID string)
}

// loginLockEntry records the state of a locked account.
type loginLockEntry struct {
	failures  int
	lockedAt  time.Time
	lockUntil time.Time
}

const (
	defaultMaxFailures    = 5
	defaultLockDuration   = 15 * time.Minute
	defaultTrackerCleanup = 10 * time.Minute
)

// inMemoryLoginTracker is a simple in-memory failed-login tracker.
// It should be replaced with a Redis-backed implementation in multi-instance deployments.
type inMemoryLoginTracker struct {
	mu       sync.Mutex
	entries  map[string]*loginLockEntry
	stopOnce sync.Once
	stopCh   chan struct{}
}

func newInMemoryLoginTracker() *inMemoryLoginTracker {
	t := &inMemoryLoginTracker{
		entries: make(map[string]*loginLockEntry),
		stopCh:  make(chan struct{}),
	}
	go t.sweep()
	return t
}

func (t *inMemoryLoginTracker) Check(_ context.Context, loginID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[loginID]
	if !ok {
		return nil
	}
	now := time.Now()
		if now.Before(entry.lockUntil) {
			return ErrAccountLocked
		}
		// Lock expired (lockUntil was set and is now in the past) — clean up.
		// Don't delete entries that haven't been locked yet (lockUntil is zero).
		if !entry.lockUntil.IsZero() && now.After(entry.lockUntil) {
			delete(t.entries, loginID)
		}
		return nil
}

func (t *inMemoryLoginTracker) RecordFailed(_ context.Context, loginID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	entry := t.entries[loginID]
	if entry == nil {
		entry = &loginLockEntry{}
		t.entries[loginID] = entry
	} else if now.Before(entry.lockUntil) {
		// Already locked — don't reset the timer
		return
	}
	entry.failures++
	if entry.failures >= defaultMaxFailures {
		entry.lockedAt = now
		entry.lockUntil = now.Add(defaultLockDuration)
	}
}

func (t *inMemoryLoginTracker) Clear(_ context.Context, loginID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, loginID)
}

func (t *inMemoryLoginTracker) sweep() {
	ticker := time.NewTicker(defaultTrackerCleanup)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for id, entry := range t.entries {
				if now.After(entry.lockUntil) {
					delete(t.entries, id)
				}
			}
			t.mu.Unlock()
		}
	}
}

func (t *inMemoryLoginTracker) stop() {
	t.stopOnce.Do(func() { close(t.stopCh) })
}

type emailSender interface {
	SendPasswordReset(ctx context.Context, to string, code string) error
}

type Service struct {
	auth         *platformauth.Service
	repo         Repository
	cfg          config.Config
	appConfig    appconfig.Source
	emailSender  emailSender
	loginTracker FailedLoginTracker
}

func NewService(auth *platformauth.Service, repo Repository, cfg config.Config) *Service {
	return NewServiceWithSource(auth, repo, cfg, appconfig.NewStaticSource(cfg))
}

func NewServiceWithSource(auth *platformauth.Service, repo Repository, cfg config.Config, source appconfig.Source) *Service {
	if repo == nil {
		repo = NewInMemoryRepository()
	}
	if source == nil {
		source = appconfig.NewStaticSource(cfg)
	}
		return &Service{auth: auth, repo: repo, cfg: cfg, appConfig: source, emailSender: &noopEmailSender{}, loginTracker: newInMemoryLoginTracker()}
	}

func (s *Service) WithEmailSender(sender emailSender) *Service {
	s.emailSender = sender
	return s
}

// WithLoginTracker replaces the default in-memory login tracker.
// Use this to inject a Redis-backed tracker for multi-instance deployments.
// For tests, pass a tracker that uses configurable thresholds and a fake clock.
func (s *Service) WithLoginTracker(tracker FailedLoginTracker) *Service {
	if tracker != nil {
		if lt, ok := s.loginTracker.(*inMemoryLoginTracker); ok {
			lt.stop() // stop the old tracker's sweeper goroutine
		}
		s.loginTracker = tracker
	}
	return s
}

type noopEmailSender struct{}

func (*noopEmailSender) SendPasswordReset(_ context.Context, _ string, _ string) error {
	return ErrEmailServiceUnavailable
}

func (s *Service) CheckStatus(ctx context.Context) AuthStatus {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return AuthStatus{IsValid: false}
	}
	expiresAt, ok := platformauth.AccessExpiryFromContext(ctx)
	if !ok {
		return AuthStatus{IsValid: true}
	}
	return AuthStatus{IsValid: true, ExpiresAt: expiresAt.UTC().Format(time.RFC3339)}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	account, err := s.repo.Register(ctx, RegisterParams{
		LoginID:          req.LoginID,
		Nickname:         req.Nickname,
		Password:         req.Password,
		Defaults:         s.profileDefaults(ctx),
		RegisterIP:       meta.IPAddress,
		RegisterLocation: meta.IPLocation,
	})
		if err != nil {
			observability.L().Error("service error",
				"method", "Register",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Register: repo: %w", err)
		}
		result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
		if err != nil {
			observability.L().Error("service error",
				"method", "Register",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Register: issue session: %w", err)
		}
	resourceguard.RecordUser(ctx, account.UserID)
	return result, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()

	// Check brute-force lockout before even hashing the password.
	if err := s.loginTracker.Check(ctx, req.LoginID); err != nil {
		observability.L().Warn("login blocked by lockout",
			"method", "Login",
			"login_id", req.LoginID,
		)
		return platformauth.AuthResult{}, err
	}

	account, err := s.repo.Authenticate(ctx, req.LoginID, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			s.loginTracker.RecordFailed(ctx, req.LoginID)
		}
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: authenticate: %w", err)
	}

	// Successful login — reset the failed-attempt counter.
	s.loginTracker.Clear(ctx, req.LoginID)

	if err := s.repo.UpdateLoginMeta(ctx, account.UserID, meta.IPAddress, meta.IPLocation); err != nil {
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: update meta: %w", err)
	}
	result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
	if err != nil {
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: issue session: %w", err)
	}
	return result, nil
}

func (s *Service) Guest(ctx context.Context, req GuestRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	account, err := s.repo.CreateGuest(ctx, GuestParams{Nickname: req.Nickname, Defaults: s.profileDefaults(ctx), RegisterIP: meta.IPAddress, RegisterLocation: meta.IPLocation})
		if err != nil {
			observability.L().Error("service error",
				"method", "Guest",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Guest: repo: %w", err)
		}
		result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
		if err != nil {
			observability.L().Error("service error",
				"method", "Guest",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Guest: issue session: %w", err)
		}
	resourceguard.RecordUser(ctx, account.UserID)
	return result, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshTokenRequest, refreshCookie string) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	refreshToken := req.RefreshToken
	if req.ClientType == ClientTypeWeb {
		refreshToken = refreshCookie
	}
		result, err := s.auth.RefreshSession(ctx, refreshToken)
		if err != nil {
			observability.L().Error("service error",
				"method", "Refresh",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Refresh: %w", err)
		}
	return result, nil
}

func (s *Service) ListSessions(ctx context.Context) SessionList {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return SessionList{Items: []SessionItem{}}
	}
	sessions := s.auth.ListSessions(ctx, principal.UserID, principal.SessionID)
	items := make([]SessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, SessionItem{
			SessionID:    session.SessionID,
			DeviceType:   session.DeviceType,
			DeviceName:   session.DeviceName,
			IPAddress:    maskIPAddress(session.IPAddress),
			IPLocation:   session.IPLocation,
			CreatedAt:    session.CreatedAt.UTC().Format(time.RFC3339),
			LastActiveAt: session.LastActiveAt.UTC().Format(time.RFC3339),
			IsCurrent:    session.SessionID == principal.SessionID,
		})
	}
	return SessionList{Items: items}
}

func (s *Service) RevokeSession(ctx context.Context, req RevokeSessionRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
	if req.SessionID == principal.SessionID {
		return platformauth.Forbidden("不允许撤销当前会话")
	}
		if err := s.auth.RevokeSession(ctx, principal.UserID, principal.SessionID, req.SessionID); err != nil {
			observability.L().Error("service error",
				"method", "RevokeSession",
				"error", err,
			)
			return fmt.Errorf("auth.Service.RevokeSession: %w", err)
		}
	realtime.DefaultHub().CloseSession(req.SessionID, sessionRevokedEvent())
	return nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, req RequestPasswordResetRequest) error {
		code, err := s.repo.CreatePasswordReset(ctx, req.LoginID, time.Now().UTC().Add(10*time.Minute))
		if errors.Is(err, ErrPasswordResetThrottled) {
			return nil
		}
		if err != nil {
			observability.L().Error("service error",
				"method", "RequestPasswordReset",
				"error", err,
			)
			return fmt.Errorf("auth.Service.RequestPasswordReset: repo: %w", err)
		}
	if code == "" {
		return nil
	}
	if sendErr := s.emailSender.SendPasswordReset(ctx, req.LoginID, code); sendErr != nil {
		observability.L().Warn("email send failed for password reset",
			"method", "RequestPasswordReset",
			"login_id", req.LoginID,
			"error", sendErr,
		)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
		account, err := s.repo.ResetPassword(ctx, req.LoginID, req.VerificationCode, req.NewPassword)
		if err != nil {
			observability.L().Error("service error",
				"method", "ResetPassword",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.ResetPassword: repo: %w", err)
		}
		s.closeAllSessions(ctx, account.UserID)
		s.auth.RevokeOtherSessions(ctx, account.UserID, "")
		result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
		if err != nil {
			observability.L().Error("service error",
				"method", "ResetPassword",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.ResetPassword: issue session: %w", err)
		}
	return result, nil
}

func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
		if err := s.repo.ChangePassword(ctx, principal.UserID, req.OldPassword, req.NewPassword); err != nil {
			return fmt.Errorf("auth.Service.ChangePassword: repo: %w", err)
		}
	s.closeOtherSessions(ctx, principal.UserID, principal.SessionID)
	s.auth.RevokeOtherSessions(ctx, principal.UserID, principal.SessionID)
	return nil
}

func (s *Service) RevokeAll(ctx context.Context) RevokeAllResult {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return RevokeAllResult{}
	}
	s.closeOtherSessions(ctx, principal.UserID, principal.SessionID)
	return RevokeAllResult{RevokedCount: s.auth.RevokeOtherSessions(ctx, principal.UserID, principal.SessionID)}
}

func (s *Service) profileDefaults(ctx context.Context) ProfileDefaults {
	now := time.Now().UTC()
	defaults := s.appConfig.Get(ctx).Defaults
	return ProfileDefaults{
		AvatarURL:    defaults.DefaultUserAvatar,
		LocationName: defaults.DefaultLocation.LocationName,
		Latitude:     defaults.DefaultLocation.Point.Latitude,
		Longitude:    defaults.DefaultLocation.Point.Longitude,
		CreatedAt:    now,
	}
}

func (s *Service) closeAllSessions(ctx context.Context, userID string) {
	sessions := s.auth.ListSessions(ctx, userID, "")
	for _, session := range sessions {
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent())
	}
}

func (s *Service) closeOtherSessions(ctx context.Context, userID, currentSessionID string) {
	sessions := s.auth.ListSessions(ctx, userID, currentSessionID)
	for _, session := range sessions {
		if session.SessionID == currentSessionID {
			continue
		}
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent())
	}
}

func sessionRevokedEvent() realtime.Event {
	return realtime.Event{Type: realtime.EventTypeError, Payload: realtime.ErrorPayload{Code: 20000, Message: "会话已失效", Severity: "fatal", Retryable: false}}
}

func maskIPAddress(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.Count(raw, ".") == 3 {
		parts := strings.Split(raw, ".")
		return parts[0] + "." + parts[1] + ".xxx.xxx"
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) > 2 {
			return strings.Join(parts[:2], ":") + ":xxxx:xxxx"
		}
	}
	return raw
}
