package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend-go/internal/auth"
	"backend-go/internal/domain"
	apierr "backend-go/internal/errors"
	"backend-go/internal/repository"
	"backend-go/internal/types"
)

// --- 测试用假实现 ---

type fakeUserRepository struct {
	usersByID    map[string]*domain.User
	usersByLogin map[string]*domain.User
	createErr    error
	findErr      error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByLogin: make(map[string]*domain.User),
	}
}

func (r *fakeUserRepository) CreateUser(_ context.Context, u *domain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	copy := *u
	r.usersByID[u.UserID] = &copy
	if u.LoginID != nil {
		r.usersByLogin[*u.LoginID] = &copy
	}
	return nil
}

func (r *fakeUserRepository) FindByLoginID(_ context.Context, loginID string) (*domain.User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if u, ok := r.usersByLogin[loginID]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, userID string) (*domain.User, error) {
	if u, ok := r.usersByID[userID]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeUserRepository) UpdatePasswordAndType(_ context.Context, userID string, passwordHash *string, userType string) error {
	u, ok := r.usersByID[userID]
	if !ok {
		return errors.New("user not found")
	}
	u.Password = passwordHash
	u.UserType = userType
	return nil
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)

type fakeSessionRepository struct {
	sessions  []domain.Session
	createErr error
}

func newFakeSessionRepository() *fakeSessionRepository {
	return &fakeSessionRepository{}
}

func (r *fakeSessionRepository) CreateSession(_ context.Context, s *domain.Session) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.sessions = append(r.sessions, *s)
	return nil
}

func (r *fakeSessionRepository) ListSessionsByUser(_ context.Context, userID string) ([]domain.Session, error) {
	var result []domain.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *fakeSessionRepository) RevokeSession(_ context.Context, userID, sessionID string) error {
	for i := range r.sessions {
		if r.sessions[i].UserID == userID && r.sessions[i].SessionID == sessionID {
			now := time.Now().UTC()
			r.sessions[i].RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (r *fakeSessionRepository) RevokeAllExcept(_ context.Context, userID, currentSessionID string) (int64, error) {
	var count int64
	for i := range r.sessions {
		if r.sessions[i].UserID == userID && r.sessions[i].SessionID != currentSessionID && r.sessions[i].RevokedAt == nil {
			now := time.Now().UTC()
			r.sessions[i].RevokedAt = &now
			count++
		}
	}
	return count, nil
}

var _ repository.SessionRepository = (*fakeSessionRepository)(nil)

// fakeTokenBlacklistRepository 记录被写入黑名单的 token，用于验证 Logout 行为。
type fakeTokenBlacklistRepository struct {
	blacklisted []struct {
		token string
	}
}

func newFakeTokenBlacklistRepository() *fakeTokenBlacklistRepository {
	return &fakeTokenBlacklistRepository{}
}

func (r *fakeTokenBlacklistRepository) BlacklistToken(_ context.Context, token string, _ time.Time) error {
	if token == "" {
		return nil
	}
	r.blacklisted = append(r.blacklisted, struct{ token string }{token: token})
	return nil
}

func (r *fakeTokenBlacklistRepository) IsTokenBlacklisted(_ context.Context, token string) (bool, error) {
	for _, entry := range r.blacklisted {
		if entry.token == token {
			return true, nil
		}
	}
	return false, nil
}

var _ repository.TokenBlacklistRepository = (*fakeTokenBlacklistRepository)(nil)

type fakeTokenManager struct {
	generateTokensFn func(userID, sessionID string) (string, string, time.Time, time.Time, error)
	parseAccessFn    func(token string) (string, string, time.Time, error)
	parseRefreshFn   func(token string) (string, string, time.Time, error)
}

func newFakeTokenManager() *fakeTokenManager {
	return &fakeTokenManager{
		generateTokensFn: func(userID, sessionID string) (string, string, time.Time, time.Time, error) {
			now := time.Now().UTC()
			return "access-token", "refresh-token", now.Add(time.Minute), now.Add(24 * time.Hour), nil
		},
		parseAccessFn: func(token string) (string, string, time.Time, error) {
			if token == "valid-access" {
				return "user-1", "session-1", time.Now().UTC().Add(time.Minute), nil
			}
			return "", "", time.Time{}, errors.New("invalid access token")
		},
		parseRefreshFn: func(token string) (string, string, time.Time, error) {
			if token == "valid-refresh" {
				return "user-1", "session-1", time.Now().UTC().Add(24 * time.Hour), nil
			}
			return "", "", time.Time{}, errors.New("invalid refresh token")
		},
	}
}

func (m *fakeTokenManager) GenerateTokens(userID, sessionID string) (string, string, time.Time, time.Time, error) {
	return m.generateTokensFn(userID, sessionID)
}

func (m *fakeTokenManager) ParseAccessToken(token string) (string, string, time.Time, error) {
	return m.parseAccessFn(token)
}

func (m *fakeTokenManager) ParseRefreshToken(token string) (string, string, time.Time, error) {
	return m.parseRefreshFn(token)
}

var _ auth.TokenManager = (*fakeTokenManager)(nil)

type authServiceTestDeps struct {
	svc      AuthService
	users    *fakeUserRepository
	sessions *fakeSessionRepository
	tokens   *fakeTokenManager
	black    *fakeTokenBlacklistRepository
}

func newAuthServiceTestDeps() *authServiceTestDeps {
	users := newFakeUserRepository()
	sessions := newFakeSessionRepository()
	tokens := newFakeTokenManager()
	black := newFakeTokenBlacklistRepository()
	svc := NewAuthService(users, sessions, tokens, black)
	return &authServiceTestDeps{svc: svc, users: users, sessions: sessions, tokens: tokens, black: black}
}

// --- Register ---

func TestAuthService_Register(t *testing.T) {
	t.Run("success_creates_user_and_session", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()

		req := types.RegisterRequest{
			LoginID:    "alice",
			Nickname:   "Alice",
			Password:   "secret",
			ClientType: types.ClientTypeApp,
		}
		tokens, err := deps.svc.Register(ctx, req)
		if err != nil {
			t.Fatalf("Register returned error: %v", err)
		}
		if tokens == nil || tokens.UserID == "" || tokens.AccessToken == "" || tokens.RefreshToken == "" {
			t.Fatalf("expected non-nil tokens with filled fields, got %+v", tokens)
		}
		user, ok := deps.users.usersByLogin["alice"]
		if !ok {
			t.Fatalf("expected user to be stored in repository")
		}
		if user.UserType != "normal" {
			t.Fatalf("expected user_type=normal, got %s", user.UserType)
		}
		if len(deps.sessions.sessions) != 1 {
			t.Fatalf("expected 1 session to be created, got %d", len(deps.sessions.sessions))
		}
	})

	t.Run("duplicate_login_id_returns_10400", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		req := types.RegisterRequest{
			LoginID:    "alice",
			Nickname:   "Alice",
			Password:   "secret",
			ClientType: types.ClientTypeApp,
		}
		if _, err := deps.svc.Register(ctx, req); err != nil {
			t.Fatalf("first Register returned error: %v", err)
		}
		_, err := deps.svc.Register(ctx, req)
		if err == nil {
			t.Fatalf("expected error on duplicate login_id, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10400 {
			t.Fatalf("expected code 10400 for duplicate, got %d", apiErr.Code)
		}
	})

	t.Run("missing_fields_returns_10000", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		_, err := deps.svc.Register(ctx, types.RegisterRequest{})
		if err == nil {
			t.Fatalf("expected error for missing fields, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10000 {
			t.Fatalf("expected code 10000 for invalid request, got %d", apiErr.Code)
		}
	})
}

// --- Login ---

func TestAuthService_Login(t *testing.T) {
	t.Run("success_with_correct_password", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()

		loginID := "bob"
		password := "secret"
		hash, err := auth.HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword error: %v", err)
		}
		now := time.Now().UTC()
		deps.users.CreateUser(ctx, &domain.User{
			UserID:    "user-bob",
			LoginID:   &loginID,
			Nickname:  "Bob",
			Password:  &hash,
			UserType:  "normal",
			CreatedAt: now,
			UpdatedAt: now,
		})

		tokens, err := deps.svc.Login(ctx, types.LoginRequest{
			LoginID:    loginID,
			Password:   password,
			ClientType: types.ClientTypeApp,
		})
		if err != nil {
			t.Fatalf("Login returned error: %v", err)
		}
		if tokens == nil || tokens.UserID != "user-bob" {
			t.Fatalf("expected tokens for user-bob, got %+v", tokens)
		}
		if len(deps.sessions.sessions) != 1 {
			t.Fatalf("expected 1 session created, got %d", len(deps.sessions.sessions))
		}
	})

	t.Run("unknown_user_returns_10100", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		_, err := deps.svc.Login(ctx, types.LoginRequest{
			LoginID:    "unknown",
			Password:   "pw",
			ClientType: types.ClientTypeApp,
		})
		if err == nil {
			t.Fatalf("expected error for unknown user, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10100 {
			t.Fatalf("expected code 10100 for auth failure, got %d", apiErr.Code)
		}
	})

	t.Run("wrong_password_returns_10100", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		loginID := "bob"
		hash, err := auth.HashPassword("correct")
		if err != nil {
			t.Fatalf("HashPassword error: %v", err)
		}
		now := time.Now().UTC()
		deps.users.CreateUser(ctx, &domain.User{
			UserID:    "user-bob",
			LoginID:   &loginID,
			Nickname:  "Bob",
			Password:  &hash,
			UserType:  "normal",
			CreatedAt: now,
			UpdatedAt: now,
		})

		_, err = deps.svc.Login(ctx, types.LoginRequest{
			LoginID:    loginID,
			Password:   "wrong",
			ClientType: types.ClientTypeApp,
		})
		if err == nil {
			t.Fatalf("expected error for wrong password, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10100 {
			t.Fatalf("expected code 10100 for auth failure, got %d", apiErr.Code)
		}
	})
}

// --- Guest ---

func TestAuthService_Guest(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()

	tokens, err := deps.svc.Guest(ctx, types.GuestRequest{
		Nickname:   "Guest1",
		ClientType: types.ClientTypeApp,
	})
	if err != nil {
		t.Fatalf("Guest returned error: %v", err)
	}
	if tokens == nil || tokens.UserID == "" {
		t.Fatalf("expected non-nil tokens with user_id, got %+v", tokens)
	}
	if len(deps.users.usersByID) != 1 {
		t.Fatalf("expected 1 user created, got %d", len(deps.users.usersByID))
	}
	var u *domain.User
	for _, user := range deps.users.usersByID {
		u = user
	}
	if u.UserType != "temp" {
		t.Fatalf("expected temp user_type for guest, got %s", u.UserType)
	}
	if u.LoginID != nil {
		t.Fatalf("expected guest LoginID to be nil")
	}
	if len(deps.sessions.sessions) != 1 {
		t.Fatalf("expected 1 session created for guest, got %d", len(deps.sessions.sessions))
	}
}

// --- RefreshToken ---

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("app_missing_refresh_token_returns_10000", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		_, err := deps.svc.RefreshToken(ctx, types.RefreshTokenRequest{ClientType: types.ClientTypeApp}, "")
		if err == nil {
			t.Fatalf("expected error for missing refresh_token, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10000 {
			t.Fatalf("expected code 10000 for invalid request, got %d", apiErr.Code)
		}
	})

	t.Run("web_invalid_token_returns_10100", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		_, err := deps.svc.RefreshToken(ctx, types.RefreshTokenRequest{ClientType: types.ClientTypeWeb}, "invalid-refresh")
		if err == nil {
			t.Fatalf("expected error for invalid refresh token, got nil")
		}
		apiErr, ok := err.(*apierr.APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != 10100 {
			t.Fatalf("expected code 10100 for auth failure, got %d", apiErr.Code)
		}
	})

	t.Run("app_valid_refresh_token_returns_tokens", func(t *testing.T) {
		deps := newAuthServiceTestDeps()
		ctx := context.Background()
		refresh := "valid-refresh"
		req := types.RefreshTokenRequest{ClientType: types.ClientTypeApp, RefreshToken: &refresh}
		tokens, err := deps.svc.RefreshToken(ctx, req, "")
		if err != nil {
			t.Fatalf("expected no error for valid refresh token, got %v", err)
		}
		if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
			t.Fatalf("expected non-empty tokens, got %+v", tokens)
		}
	})
}

// --- CheckStatus ---

func TestAuthService_CheckStatus(t *testing.T) {
	deps := newAuthServiceTestDeps()
	ctx := context.Background()

	t.Run("no_token_returns_invalid_false", func(t *testing.T) {
		status, err := deps.svc.CheckStatus(ctx, "")
		if err != nil {
			t.Fatalf("CheckStatus error: %v", err)
		}
		if status.IsValid {
			t.Fatalf("expected is_valid=false for empty token")
		}
	})

	t.Run("invalid_token_returns_invalid_false", func(t *testing.T) {
		status, err := deps.svc.CheckStatus(ctx, "invalid-access")
		if err != nil {
			t.Fatalf("CheckStatus error: %v", err)
		}
		if status.IsValid {
			t.Fatalf("expected is_valid=false for invalid token")
		}
	})

	t.Run("valid_token_returns_true_and_expires_at", func(t *testing.T) {
		status, err := deps.svc.CheckStatus(ctx, "valid-access")
		if err != nil {
			t.Fatalf("CheckStatus error: %v", err)
		}
		if !status.IsValid {
			t.Fatalf("expected is_valid=true for valid token")
		}
		if status.ExpiresAt == nil || *status.ExpiresAt == "" {
			t.Fatalf("expected expires_at to be set for valid token")
		}
	})
}
