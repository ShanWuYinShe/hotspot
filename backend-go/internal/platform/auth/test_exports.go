package auth

func (s *Service) UnsafeFirstSessionIDForUser(userID string) string {
	sessions := s.sessions.ListActiveByUser(userID)
	if len(sessions) == 0 {
		return ""
	}
	return sessions[0].SessionID
}
