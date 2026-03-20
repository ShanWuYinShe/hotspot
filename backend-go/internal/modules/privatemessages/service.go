package private_messages

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	repo Repository
}

func NewService() *Service {
	return NewServiceWithRepo(NewInMemoryRepository())
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Send(ctx context.Context, req SendPrivateMessageRequest) MessageSent {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	msg := s.repo.Send(ctx, principal.UserID, req.TargetUserID, req)
	realtime.DefaultHub().BroadcastToUser(req.TargetUserID, realtime.Event{Type: "PrivateMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt}})
	return msg
}

func (s *Service) History(ctx context.Context, targetUserID string) PaginatedMessages {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.History(ctx, principal.UserID, targetUserID)
}

func (s *Service) Recent(ctx context.Context, targetUserID string, limit int) PaginatedMessages {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Recent(ctx, principal.UserID, targetUserID, limit)
}

func (s *Service) Last(ctx context.Context, targetUserID string) *PrivateMessageItem {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Last(ctx, principal.UserID, targetUserID)
}
