package groupmessages

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	msg, err := s.repo.Send(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.send failed", "error", err, "group_id", req.GroupID)
		return MessageSent{}, err
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: "GroupMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt, "group_id": req.GroupID}})
	return msg, nil
}
func (s *Service) History(ctx context.Context, groupID string) (PaginatedGroupMessages, error) {
	return s.repo.History(ctx, groupID)
}

func (s *Service) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	if err := s.repo.Update(ctx, req); err != nil {
		observability.L().Error("groupmessages.update failed", "error", err, "message_id", req.MessageID)
		return err
	}
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	if err := s.repo.Recall(ctx, req); err != nil {
		observability.L().Error("groupmessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkGroupMessagesReadRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAsRead(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groupmessages.mark_as_read failed", "error", err, "group_id", req.GroupID, "up_to_message_id", req.UpToMessageID, "user_id", principal.UserID)
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("groupmessages.mark_all_read failed", "error", err, "user_id", principal.UserID)
		return err
	}
	return nil
}
