package auth

import "hotspot/backend-go/internal/platform/httpx"

func (s *Service) ListSessions(userID, currentSessionID string) []SessionRecord {
	sessions, err := s.sessions.ListActiveByUser(userID)
	if err != nil {
		return []SessionRecord{}
	}
	if currentSessionID == "" {
		return sessions
	}
	for i := range sessions {
		if sessions[i].SessionID == currentSessionID {
			_ = s.sessions.Touch(currentSessionID)
			break
		}
	}
	updated, err := s.sessions.ListActiveByUser(userID)
	if err != nil {
		return sessions
	}
	return updated
}

func (s *Service) RevokeOtherSessions(userID, currentSessionID string) int {
	count, err := s.sessions.RevokeOthers(userID, currentSessionID)
	if err != nil {
		return 0
	}
	return count
}

func (s *Service) RevokeSession(userID, _ string, targetSessionID string) error {
	target, ok, err := s.sessions.Get(targetSessionID)
	if err != nil {
		return err
	}
	if !ok || target.RevokedAt != nil {
		return &httpx.APIError{Code: 10300, Message: "会话不存在"}
	}
	if target.UserID != userID {
		return &httpx.APIError{Code: 10200, Message: "无权撤销该会话"}
	}
	if _, err := s.sessions.Revoke(targetSessionID); err != nil {
		return err
	}
	return nil
}
