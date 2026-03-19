package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"backend-go/internal/auth"
	"backend-go/internal/domain"
	apierr "backend-go/internal/errors"
	"backend-go/internal/repository"
	"backend-go/internal/types"
)

// AuthTokens 为内部使用的 Token 返回值，由 Handler 根据 client_type 转换为对外类型。
type AuthTokens struct {
	UserID                string
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
}

// AuthService 定义认证模块行为。
type AuthService interface {
	Register(ctx context.Context, req types.RegisterRequest) (*AuthTokens, error)
	Login(ctx context.Context, req types.LoginRequest) (*AuthTokens, error)
	Logout(ctx context.Context) error
	Guest(ctx context.Context, req types.GuestRequest) (*AuthTokens, error)
	RefreshToken(ctx context.Context, req types.RefreshTokenRequest, tokenStr string) (*AuthTokens, error)
	CheckStatus(ctx context.Context, tokenStr string) (*types.AuthStatus, error)
	RequestPasswordReset(ctx context.Context, req types.RequestPasswordResetRequest) error
	ResetPassword(ctx context.Context, req types.ResetPasswordRequest) (*AuthTokens, error)
	ChangePassword(ctx context.Context, userID string, req types.ChangePasswordRequest) error
	ListSessions(ctx context.Context, userID string) ([]types.SessionItem, error)
	RevokeSession(ctx context.Context, userID string, req types.RevokeSessionRequest) error
	RevokeAllSessions(ctx context.Context, userID string) (*types.RevokeAllResult, error)
}

type authService struct {
	users      repository.UserRepository
	sessions   repository.SessionRepository
	tokens     auth.TokenManager
	blacklists repository.TokenBlacklistRepository
}

func NewAuthService(users repository.UserRepository, sessions repository.SessionRepository, tokens auth.TokenManager, blacklists repository.TokenBlacklistRepository) AuthService {
	return &authService{
		users:      users,
		sessions:   sessions,
		tokens:     tokens,
		blacklists: blacklists,
	}
}

// --- 核心登录/注册逻辑 ---

func (s *authService) Register(ctx context.Context, req types.RegisterRequest) (*AuthTokens, error) {
	if req.LoginID == "" || req.Nickname == "" || req.Password == "" {
		return nil, apierr.ErrInvalidRequest("missing_required_fields")
	}

	existing, err := s.users.FindByLoginID(ctx, req.LoginID)
	if err != nil {
		return nil, apierr.ErrInternal("db_error")
	}
	if existing != nil {
		// 10400: 业务冲突，例如登录标识已存在
		return nil, apierr.New(10400, "login_id_already_exists")
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, apierr.ErrInternal("hash_password_failed")
	}

	now := time.Now().UTC()
	user := &domain.User{
		UserID:    uuid.NewString(),
		LoginID:   &req.LoginID,
		Nickname:  req.Nickname,
		Password:  &passwordHash,
		UserType:  "normal",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return nil, apierr.ErrInternal("create_user_failed")
	}

	sessionID := uuid.NewString()
	sess := &domain.Session{
		SessionID:    sessionID,
		UserID:       user.UserID,
		CreatedAt:    now,
		LastActiveAt: now,
	}
	if err := s.sessions.CreateSession(ctx, sess); err != nil {
		return nil, apierr.ErrInternal("create_session_failed")
	}

	access, refresh, accessExp, refreshExp, err := s.tokens.GenerateTokens(user.UserID, sessionID)
	if err != nil {
		return nil, apierr.ErrInternal("generate_token_failed")
	}

	return &AuthTokens{
		UserID:                user.UserID,
		AccessToken:           access,
		RefreshToken:          refresh,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

func (s *authService) Login(ctx context.Context, req types.LoginRequest) (*AuthTokens, error) {
	if req.LoginID == "" || req.Password == "" {
		return nil, apierr.ErrInvalidRequest("missing_required_fields")
	}

	user, err := s.users.FindByLoginID(ctx, req.LoginID)
	if err != nil {
		return nil, apierr.ErrInternal("db_error")
	}
	// 统一返回 10100，防止枚举登录标识
	if user == nil || user.Password == nil {
		return nil, apierr.New(10100, "auth_failed")
	}

	if err := auth.CheckPassword(*user.Password, req.Password); err != nil {
		return nil, apierr.New(10100, "auth_failed")
	}

	now := time.Now().UTC()
	sessionID := uuid.NewString()
	sess := &domain.Session{
		SessionID:    sessionID,
		UserID:       user.UserID,
		CreatedAt:    now,
		LastActiveAt: now,
	}
	if err := s.sessions.CreateSession(ctx, sess); err != nil {
		return nil, apierr.ErrInternal("create_session_failed")
	}

	access, refresh, accessExp, refreshExp, err := s.tokens.GenerateTokens(user.UserID, sessionID)
	if err != nil {
		return nil, apierr.ErrInternal("generate_token_failed")
	}

	return &AuthTokens{
		UserID:                user.UserID,
		AccessToken:           access,
		RefreshToken:          refresh,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

func (s *authService) Guest(ctx context.Context, req types.GuestRequest) (*AuthTokens, error) {
	if req.Nickname == "" {
		return nil, apierr.ErrInvalidRequest("missing_nickname")
	}
	now := time.Now().UTC()
	user := &domain.User{
		UserID:    uuid.NewString(),
		LoginID:   nil,
		Nickname:  req.Nickname,
		Password:  nil,
		UserType:  "temp",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return nil, apierr.ErrInternal("create_guest_failed")
	}

	sessionID := uuid.NewString()
	sess := &domain.Session{
		SessionID:    sessionID,
		UserID:       user.UserID,
		CreatedAt:    now,
		LastActiveAt: now,
	}
	if err := s.sessions.CreateSession(ctx, sess); err != nil {
		return nil, apierr.ErrInternal("create_session_failed")
	}

	access, refresh, accessExp, refreshExp, err := s.tokens.GenerateTokens(user.UserID, sessionID)
	if err != nil {
		return nil, apierr.ErrInternal("generate_token_failed")
	}

	return &AuthTokens{
		UserID:                user.UserID,
		AccessToken:           access,
		RefreshToken:          refresh,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, req types.RefreshTokenRequest, tokenStr string) (*AuthTokens, error) {
	if req.ClientType == types.ClientTypeApp {
		if req.RefreshToken == nil || *req.RefreshToken == "" {
			return nil, apierr.ErrInvalidRequest("missing_refresh_token")
		}
		tokenStr = *req.RefreshToken
	}
	if tokenStr == "" {
		return nil, apierr.New(10100, "auth_failed")
	}

	userID, sessionID, _, err := s.tokens.ParseRefreshToken(tokenStr)
	if err != nil {
		return nil, apierr.New(10100, "auth_failed")
	}

	access, refresh, accessExp, refreshExp, err := s.tokens.GenerateTokens(userID, sessionID)
	if err != nil {
		return nil, apierr.ErrInternal("generate_token_failed")
	}

	return &AuthTokens{
		UserID:                userID,
		AccessToken:           access,
		RefreshToken:          refresh,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
	}, nil
}

func (s *authService) CheckStatus(ctx context.Context, tokenStr string) (*types.AuthStatus, error) {
	if tokenStr == "" {
		return &types.AuthStatus{IsValid: false}, nil
	}
	_, _, exp, err := s.tokens.ParseAccessToken(tokenStr)
	if err != nil {
		return &types.AuthStatus{IsValid: false}, nil
	}
	expStr := exp.UTC().Format(time.RFC3339Nano)
	return &types.AuthStatus{IsValid: true, ExpiresAt: &expStr}, nil
}

func (s *authService) Logout(ctx context.Context) error {
	info, ok := auth.FromContext(ctx)
	if !ok {
		// 缺少认证上下文，视为未登录 / 认证失败
		return apierr.New(10100, "auth_failed")
	}

	// 当前 access_token / refresh_token 写入黑名单（最佳努力，忽略 Redis 错误）
	_ = s.blacklists.BlacklistToken(ctx, info.AccessToken, info.AccessExpiresAt)
	_ = s.blacklists.BlacklistToken(ctx, info.RefreshToken, info.RefreshExpiresAt)

	// 撤销当前会话
	if err := s.sessions.RevokeSession(ctx, info.UserID, info.SessionID); err != nil {
		return apierr.ErrInternal("revoke_session_failed")
	}
	return nil
}

func (s *authService) RequestPasswordReset(ctx context.Context, req types.RequestPasswordResetRequest) error {
	return apierr.ErrInternal("request_password_reset_not_implemented")
}

func (s *authService) ResetPassword(ctx context.Context, req types.ResetPasswordRequest) (*AuthTokens, error) {
	return nil, apierr.ErrInternal("reset_password_not_implemented")
}

func (s *authService) ChangePassword(ctx context.Context, userID string, req types.ChangePasswordRequest) error {
	return apierr.ErrInternal("change_password_not_implemented")
}

func (s *authService) ListSessions(ctx context.Context, userID string) ([]types.SessionItem, error) {
	info, _ := auth.FromContext(ctx)
	uid := userID
	if uid == "" && info != nil {
		uid = info.UserID
	}
	if uid == "" {
		return nil, apierr.New(10100, "auth_failed")
	}

	sessions, err := s.sessions.ListSessionsByUser(ctx, uid)
	if err != nil {
		return nil, apierr.ErrInternal("db_error")
	}

	currentSessionID := ""
	if info != nil {
		currentSessionID = info.SessionID
	}

	items := make([]types.SessionItem, 0, len(sessions))
	for _, s := range sessions {
		// 仅返回未被撤销的会话
		if s.RevokedAt != nil {
			continue
		}
		item := types.SessionItem{
			SessionID:    s.SessionID,
			CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339Nano),
			LastActiveAt: s.LastActiveAt.UTC().Format(time.RFC3339Nano),
			IsCurrent:    s.SessionID == currentSessionID,
		}
		item.DeviceType = s.DeviceType
		item.DeviceName = s.DeviceName
		item.IPAddress = s.IPAddress
		item.IPLocation = s.IPLocation
		items = append(items, item)
	}
	return items, nil
}

func (s *authService) RevokeSession(ctx context.Context, userID string, req types.RevokeSessionRequest) error {
	info, ok := auth.FromContext(ctx)
	if !ok {
		return apierr.New(10100, "auth_failed")
	}
	if req.SessionID == "" {
		return apierr.ErrInvalidRequest("missing_session_id")
	}
	if req.SessionID == info.SessionID {
		// 当前会话需使用 Logout，而不是 revoke
		return apierr.ErrInvalidRequest("cannot_revoke_current_session")
	}
	if err := s.sessions.RevokeSession(ctx, info.UserID, req.SessionID); err != nil {
		return apierr.ErrInternal("revoke_session_failed")
	}
	return nil
}

func (s *authService) RevokeAllSessions(ctx context.Context, userID string) (*types.RevokeAllResult, error) {
	info, ok := auth.FromContext(ctx)
	if !ok {
		return nil, apierr.New(10100, "auth_failed")
	}

	count, err := s.sessions.RevokeAllExcept(ctx, info.UserID, info.SessionID)
	if err != nil {
		return nil, apierr.ErrInternal("revoke_all_sessions_failed")
	}
	return &types.RevokeAllResult{RevokedCount: int(count)}, nil
}
