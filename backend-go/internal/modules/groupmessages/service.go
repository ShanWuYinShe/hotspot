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

func (s *Service) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	return s.repo.Update(ctx, req)
}

func (s *Service) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	return s.repo.Recall(ctx, req)
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkGroupMessagesReadRequest) error {
	return s.repo.MarkAsRead(ctx, req.GroupID, req.UpToMessageID)
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	return s.repo.MarkAllRead(ctx)
}
