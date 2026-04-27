package groupmessages

import (
	"context"
	"time"

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
	payload := map[string]any{
		"message_id": msg.MessageID,
		"group_id":   req.GroupID,
		"sender":     map[string]any{"user_id": req.SenderUserID},
		"type":       req.Type,
		"sent_at":    msg.SentAt,
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.ShareCode != "" {
		payload["share_code"] = req.ShareCode
	}
	if len(req.Mentions) > 0 {
		payload["mentions"] = req.Mentions
	}
	if req.ReplyToMessageID != "" {
		payload["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.FileID != "" {
		payload["file_id"] = req.FileID
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: "GroupMessageNew", Payload: payload})
	return msg, nil
}
func (s *Service) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	return s.repo.History(ctx, groupID, pagination)
}

func (s *Service) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	meta, err := s.repo.Update(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.update failed", "error", err, "message_id", req.MessageID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: "GroupMessageEdited", Payload: map[string]any{
		"group_id":   meta.GroupID,
		"message_id": meta.MessageID,
		"content":    meta.Content,
		"edited_at":  time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	meta, err := s.repo.Recall(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: "GroupMessageRecalled", Payload: map[string]any{
		"group_id":    meta.GroupID,
		"message_id":  meta.MessageID,
		"recalled_by": map[string]any{"user_id": req.UserID},
		"recalled_at": time.Now().UTC().Format(time.RFC3339),
	}})
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
