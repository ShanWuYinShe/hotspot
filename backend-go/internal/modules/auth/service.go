package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	auth      *platformauth.Service
	repo      Repository
	cfg       config.Config
	appConfig appconfig.Source
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
	return &Service{auth: auth, repo: repo, cfg: cfg, appConfig: source}
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

func (s *Service) Register(ctx context.Context, req RegisterRequest) (platformauth.AuthResult, error) {
	account, err := s.repo.Register(ctx, RegisterParams{
		LoginID:  req.LoginID,
		Nickname: req.Nickname,
		Password: req.Password,
		Defaults: s.profileDefaults(ctx),
	})
	if err != nil {
		return platformauth.AuthResult{}, err
	}
	return s.auth.IssueSession(account.UserID)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (platformauth.AuthResult, error) {
	account, err := s.repo.Authenticate(ctx, req.LoginID, req.Password)
	if err != nil {
		return platformauth.AuthResult{}, err
	}
	return s.auth.IssueSession(account.UserID)
}

func (s *Service) Guest(ctx context.Context, req GuestRequest) (platformauth.AuthResult, error) {
	account, err := s.repo.CreateGuest(ctx, GuestParams{Nickname: req.Nickname, Defaults: s.profileDefaults(ctx)})
	if err != nil {
		return platformauth.AuthResult{}, err
	}
	return s.auth.IssueSession(account.UserID)
}

func (s *Service) Refresh(req RefreshTokenRequest, refreshCookie string) (platformauth.AuthResult, error) {
	refreshToken := req.RefreshToken
	if req.ClientType == ClientTypeWeb {
		refreshToken = refreshCookie
	}
	return s.auth.RefreshSession(refreshToken)
}

func (s *Service) ListSessions(ctx context.Context) SessionList {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return SessionList{Items: []SessionItem{}}
	}
	sessions := s.auth.ListSessions(principal.UserID, principal.SessionID)
	items := make([]SessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, SessionItem{
			SessionID:    session.SessionID,
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
	if err := s.auth.RevokeSession(principal.UserID, principal.SessionID, req.SessionID); err != nil {
		return err
	}
	realtime.DefaultHub().CloseSession(req.SessionID, sessionRevokedEvent())
	return nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, req RequestPasswordResetRequest) error {
	err := s.repo.CreatePasswordReset(ctx, req.LoginID, time.Now().UTC().Add(10*time.Minute))
	if errors.Is(err, ErrPasswordResetThrottled) {
		return nil
	}
	return err
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) (platformauth.AuthResult, error) {
	account, err := s.repo.ResetPassword(ctx, req.LoginID, req.VerificationCode, req.NewPassword)
	if err != nil {
		return platformauth.AuthResult{}, err
	}
	s.closeAllSessions(account.UserID)
	s.auth.RevokeOtherSessions(account.UserID, "")
	return s.auth.IssueSession(account.UserID)
}

func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
	if err := s.repo.ChangePassword(ctx, principal.UserID, req.OldPassword, req.NewPassword); err != nil {
		return err
	}
	s.closeOtherSessions(principal.UserID, principal.SessionID)
	s.auth.RevokeOtherSessions(principal.UserID, principal.SessionID)
	return nil
}

func (s *Service) RevokeAll(ctx context.Context) RevokeAllResult {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return RevokeAllResult{}
	}
	s.closeOtherSessions(principal.UserID, principal.SessionID)
	return RevokeAllResult{RevokedCount: s.auth.RevokeOtherSessions(principal.UserID, principal.SessionID)}
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

func (s *Service) closeAllSessions(userID string) {
	sessions := s.auth.ListSessions(userID, "")
	for _, session := range sessions {
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent())
	}
}

func (s *Service) closeOtherSessions(userID, currentSessionID string) {
	sessions := s.auth.ListSessions(userID, currentSessionID)
	for _, session := range sessions {
		if session.SessionID == currentSessionID {
			continue
		}
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent())
	}
}

func sessionRevokedEvent() realtime.Event {
	return realtime.Event{Type: "Error", Payload: realtime.ErrorPayload{Code: 20000, Message: "会话已失效", Severity: "fatal", Retryable: false}}
}
