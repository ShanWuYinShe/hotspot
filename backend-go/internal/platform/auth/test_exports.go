package auth

func (s *Service) UnsafeFirstSessionIDForUser(userID string) string {
	sessions, err := s.sessions.ListActiveByUser(userID)
	if err != nil || len(sessions) == 0 {
		return ""
	}
	return sessions[0].SessionID
}
