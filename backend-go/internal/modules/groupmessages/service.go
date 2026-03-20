package groupmessages

import (
	"context"

	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Send(ctx context.Context, req SendGroupMessageRequest) MessageSent {
	msg := s.repo.Send(ctx, req)
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: "GroupMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt, "group_id": req.GroupID}})
	return msg
}
func (s *Service) History(ctx context.Context, groupID string) PaginatedGroupMessages {
	return s.repo.History(ctx, groupID)
}
