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
}

// SessionMeta carries request-derived session metadata for auditing and UX.
type SessionMeta struct {
	DeviceType string
	DeviceName string
	IPAddress  string
	IPLocation string
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
	session, ok, err := s.sessions.Get(claims.SessionID)
	if err != nil || !ok || session.UserID != claims.UserID || session.RevokedAt != nil {
		return Principal{}, time.Time{}, false
	}
	return newPrincipal(claims.UserID, claims.SessionID, platformFromContext(rctx)), claims.ExpiresAt, true
}

func (s *Service) IssueSession(userID string) (AuthResult, error) {
	return s.IssueSessionWithMeta(userID, SessionMeta{})
}

func (s *Service) IssueSessionWithMeta(userID string, meta SessionMeta) (AuthResult, error) {
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
	refreshExpires := time.Now().UTC().Add(s.refreshTTL)
	if err := s.sessions.Save(SessionRecord{
		SessionID:    sessionID,
		UserID:       userID,
		CreatedAt:    createdAt,
		LastActiveAt: createdAt,
		DeviceType:   meta.DeviceType,
		DeviceName:   meta.DeviceName,
		IPAddress:    meta.IPAddress,
		IPLocation:   meta.IPLocation,
	}); err != nil {
		return AuthResult{}, err
	}
	if err := s.refreshes.Save(RefreshSession{UserID: userID, SessionID: sessionID, RefreshToken: refreshToken, ExpiresAt: refreshExpires}); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Principal: newPrincipal(userID, sessionID, ""), AccessToken: accessToken, RefreshToken: refreshToken, AccessExpires: accessExp, RefreshExpires: refreshExpires}, nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
