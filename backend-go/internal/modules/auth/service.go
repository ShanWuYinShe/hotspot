package auth

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct {
	auth *platformauth.Service
}

func NewService(auth *platformauth.Service) *Service {
	return &Service{auth: auth}
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

func (s *Service) Register(req RegisterRequest) (platformauth.AuthResult, error) {
	return s.auth.IssueSession(mockUserID(req.LoginID))
}

func (s *Service) Login(req LoginRequest) (platformauth.AuthResult, error) {
	return s.auth.IssueSession(mockUserID(req.LoginID))
}

func (s *Service) Guest(req GuestRequest) (platformauth.AuthResult, error) {
	return s.auth.IssueSession(mockUserID(req.Nickname))
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
	return s.auth.RevokeSession(principal.UserID, principal.SessionID, req.SessionID)
}

func (s *Service) RequestPasswordReset(req RequestPasswordResetRequest) {}

func (s *Service) ResetPassword(req ResetPasswordRequest) (platformauth.AuthResult, error) {
	return s.auth.IssueSession(mockUserID(req.LoginID))
}

func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
	s.auth.RevokeOtherSessions(principal.UserID, principal.SessionID)
	return nil
}

func (s *Service) RevokeAll(ctx context.Context) RevokeAllResult {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return RevokeAllResult{}
	}
	return RevokeAllResult{RevokedCount: s.auth.RevokeOtherSessions(principal.UserID, principal.SessionID)}
}

func mockUserID(loginID string) string {
	if loginID == "" {
		return "user_mock"
	}
	return "user_" + loginID
}
