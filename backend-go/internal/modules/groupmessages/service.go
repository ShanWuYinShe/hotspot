package groupmessages

import (
	"context"

	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	msg, err := s.repo.Send(ctx, req)
	if err != nil {
		return MessageSent{}, err
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: "GroupMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt, "group_id": req.GroupID}})
	return msg, nil
}
func (s *Service) History(ctx context.Context, groupID string) (PaginatedGroupMessages, error) {
	return s.repo.History(ctx, groupID)
}
