package auth

import (
	"errors"
	"time"
)

func (s *Service) RefreshSession(refreshToken string) (AuthResult, error) {
	session, ok, err := s.refreshes.Get(refreshToken)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errors.New("invalid refresh token")
	}
	now := time.Now().UTC()
	if now.After(session.ExpiresAt) {
		return AuthResult{}, errors.New("expired refresh token")
	}
	if storedSession, ok, err := s.sessions.Get(session.SessionID); err != nil {
		return AuthResult{}, err
	} else if !ok || storedSession.RevokedAt != nil {
		return AuthResult{}, errors.New("revoked session")
	}
	if session.RotatedTo != "" {
		if !now.After(session.GraceUntil) {
			rotated, ok, err := s.refreshes.Get(session.RotatedTo)
			if err != nil {
				return AuthResult{}, err
			}
			if !ok {
				return AuthResult{}, errors.New("invalid refresh token")
			}
			accessToken, accessExp, err := s.tokens.MintAccessToken(rotated.UserID, rotated.SessionID, s.accessTTL)
			if err != nil {
				return AuthResult{}, err
			}
			csrfToken, err := GenerateCSRFToken()
			if err != nil {
				return AuthResult{}, err
			}
			return AuthResult{
				Principal:      Principal{UserID: rotated.UserID, SessionID: rotated.SessionID},
				AccessToken:    accessToken,
				RefreshToken:   rotated.RefreshToken,
				AccessExpires:  accessExp,
				RefreshExpires: rotated.ExpiresAt,
				CSRFToken:      csrfToken,
			}, nil
		}
		return AuthResult{}, errors.New("expired refresh rotation window")
	}
	newRefreshToken, err := GenerateOpaqueToken()
	if err != nil {
		return AuthResult{}, err
	}
	newAccessToken, accessExp, err := s.tokens.MintAccessToken(session.UserID, session.SessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return AuthResult{}, err
	}
	next := RefreshSession{
		UserID:       session.UserID,
		SessionID:    session.SessionID,
		RefreshToken: newRefreshToken,
		ExpiresAt:    now.Add(s.refreshTTL),
	}
	if err := s.refreshes.Rotate(refreshToken, session, next); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Principal:      Principal{UserID: session.UserID, SessionID: session.SessionID},
		AccessToken:    newAccessToken,
		RefreshToken:   newRefreshToken,
		AccessExpires:  accessExp,
		RefreshExpires: next.ExpiresAt,
		CSRFToken:      csrfToken,
	}, nil
}
