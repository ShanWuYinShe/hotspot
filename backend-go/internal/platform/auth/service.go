package auth

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

type Service struct {
	tokens     *TokenManager
	accessTTL  time.Duration
	refreshTTL time.Duration
	refreshes  RefreshTokenStore
	sessions   SessionStore
}

type AuthResult struct {
	Principal      Principal
	AccessToken    string
	RefreshToken   string
	AccessExpires  time.Time
	RefreshExpires time.Time
	CSRFToken      string
}

func NewService(cfg config.Config) *Service {
	return &Service{
		tokens:     NewTokenManager(cfg.AuthTokenSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 30 * 24 * time.Hour,
		refreshes:  NewInMemoryRefreshStore(),
		sessions:   NewInMemorySessionStore(),
	}
}

func NewServiceWithStores(cfg config.Config, sessions SessionStore, refreshes RefreshTokenStore) *Service {
	if sessions == nil {
		sessions = NewInMemorySessionStore()
	}
	if refreshes == nil {
		refreshes = NewInMemoryRefreshStore()
	}
	return &Service{
		tokens:     NewTokenManager(cfg.AuthTokenSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 30 * 24 * time.Hour,
		refreshes:  refreshes,
		sessions:   sessions,
	}
}

func (s *Service) AuthenticateRequest(rctx context.Context, authHeader string, accessCookie string) (Principal, time.Time, bool) {
	token := bearerToken(authHeader)
	if token == "" {
		token = accessCookie
	}
	if token == "" {
		return Principal{}, time.Time{}, false
	}
	claims, err := s.tokens.ParseAccessToken(token)
	if err != nil {
		return Principal{}, time.Time{}, false
	}
	principal := Principal{UserID: claims.UserID, SessionID: claims.SessionID, Platform: platformFromContext(rctx)}
	if claims.UserID == "admin" || strings.HasPrefix(claims.UserID, "admin_") {
		principal.IsAdmin = true
	}
	return principal, claims.ExpiresAt, true
}

func (s *Service) IssueSession(userID string) (AuthResult, error) {
	sessionID, err := NewSessionID()
	if err != nil {
		return AuthResult{}, err
	}
	createdAt := time.Now().UTC()
	accessToken, accessExp, err := s.tokens.MintAccessToken(userID, sessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	refreshToken, err := GenerateOpaqueToken()
	if err != nil {
		return AuthResult{}, err
	}
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return AuthResult{}, err
	}
	refreshExpires := time.Now().UTC().Add(s.refreshTTL)
	s.sessions.Save(SessionRecord{SessionID: sessionID, UserID: userID, CreatedAt: createdAt, LastActiveAt: createdAt})
	s.refreshes.Save(RefreshSession{UserID: userID, SessionID: sessionID, RefreshToken: refreshToken, ExpiresAt: refreshExpires})
	return AuthResult{Principal: Principal{UserID: userID, SessionID: sessionID}, AccessToken: accessToken, RefreshToken: refreshToken, AccessExpires: accessExp, RefreshExpires: refreshExpires, CSRFToken: csrfToken}, nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
