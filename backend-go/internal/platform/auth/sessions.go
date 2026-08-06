package auth

import (
	"context"

	"hotspot/backend-go/internal/platform/httpx"
)

func (s *Service) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	return s.sessions.Get(ctx, sessionID)
}

func (s *Service) RevokeByUserDevice(ctx context.Context, userID, deviceID string) (int, error) {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, session := range sessions {
		if session.DeviceID == deviceID && deviceID != "" && session.RevokedAt == nil {
			if revoked, err := s.sessions.Revoke(ctx, session.SessionID); err != nil {
				return count, err
			} else if revoked {
				count++
			}
		}
	}
	return count, nil
}

func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID string) []SessionRecord {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return []SessionRecord{}
	}
	if currentSessionID == "" {
		return sessions
	}
	for i := range sessions {
		if sessions[i].SessionID == currentSessionID {
			_ = s.sessions.Touch(ctx, currentSessionID)
			break
		}
	}
	updated, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return sessions
	}
	return updated
}

func (s *Service) RevokeOtherSessions(ctx context.Context, userID, currentSessionID string) int {
	count, err := s.sessions.RevokeOthers(ctx, userID, currentSessionID)
	if err != nil {
		return 0
	}
	return count
}

func (s *Service) RevokeSession(ctx context.Context, userID, _ string, targetSessionID string) error {
	target, ok, err := s.sessions.Get(ctx, targetSessionID)
	if err != nil {
		return err
	}
	if !ok || target.RevokedAt != nil {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "会话不存在"}
	}
	if target.UserID != userID {
		return &httpx.APIError{Code: httpx.CodeForbidden, Message: "无权撤销该会话"}
	}
	if _, err := s.sessions.Revoke(ctx, targetSessionID); err != nil {
		return err
	}
	return nil
}
