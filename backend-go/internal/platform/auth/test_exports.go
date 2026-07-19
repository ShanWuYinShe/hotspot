package auth

import "context"

func (s *Service) UnsafeFirstSessionIDForUser(ctx context.Context, userID string) string {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil || len(sessions) == 0 {
		return ""
	}
	return sessions[0].SessionID
}
