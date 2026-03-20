package auth

import "hotspot/backend-go/internal/platform/httpx"

func (s *Service) ListSessions(userID, currentSessionID string) []SessionRecord {
	sessions := s.sessions.ListActiveByUser(userID)
	if currentSessionID == "" {
		return sessions
	}
	for i := range sessions {
		if sessions[i].SessionID == currentSessionID {
			s.sessions.Touch(currentSessionID)
			break
		}
	}
	return s.sessions.ListActiveByUser(userID)
}

func (s *Service) RevokeOtherSessions(userID, currentSessionID string) int {
	return s.sessions.RevokeOthers(userID, currentSessionID)
}

func (s *Service) RevokeSession(userID, currentSessionID, targetSessionID string) error {
	target, ok := s.sessions.Get(targetSessionID)
	if !ok || target.RevokedAt != nil {
		return &httpx.APIError{Code: 10300, Message: "会话不存在"}
	}
	if target.UserID != userID {
		return &httpx.APIError{Code: 10200, Message: "无权撤销该会话"}
	}
	if targetSessionID == currentSessionID {
		s.sessions.Revoke(targetSessionID)
		return nil
	}
	s.sessions.Revoke(targetSessionID)
	return nil
}
