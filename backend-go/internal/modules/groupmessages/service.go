package groupmessages

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

var ErrNotGroupMember = errors.New("groupmessages: not a group member")

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
}

type Service struct {
	repo  Repository
	users userReader
}

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo, users: nil} }
func NewServiceWithRepoAndUsers(repo Repository, users userReader) *Service {
	return &Service{repo: repo, users: users}
}
func (s *Service) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	userID := req.SenderUserID
	if userID == "" {
		if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
			userID = principal.UserID
		}
	}
	if err := s.ensureMember(ctx, req.GroupID, userID); err != nil {
		return MessageSent{}, err
	}
	msg, err := s.repo.Send(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.send failed", "error", err, "group_id", req.GroupID)
		return MessageSent{}, err
	}
	payload := map[string]any{
		"message_id": msg.MessageID,
		"group_id":   req.GroupID,
		"sender":     s.resolveUserInfo(ctx, req.SenderUserID),
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
		if replyTo := s.buildReplyTo(ctx, req.ReplyToMessageID); replyTo != nil {
			payload["reply_to"] = replyTo
		}
	}
	if req.FileID != "" {
		if media := s.buildMedia(ctx, req.FileID); media != nil {
			payload["media"] = media
		}
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: "GroupMessageNew", Payload: payload})
	return msg, nil
}
func (s *Service) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureMember(ctx, groupID, principal.UserID); err != nil {
		return PaginatedGroupMessages{}, err
	}
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

func (s *Service) ensureMember(ctx context.Context, groupID, userID string) error {
	if userID == "" {
		return ErrNotGroupMember
	}
	isMember, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotGroupMember
	}
	return nil
}

func (s *Service) resolveUserInfo(ctx context.Context, userID string) map[string]any {
	info := map[string]any{"user_id": userID}
	if s.users != nil {
		if item, ok, err := s.users.OpenTheDoor(ctx, userID); err == nil && ok {
			info["nickname"] = item.User.Nickname
			if item.User.AvatarURL != "" {
				info["avatar_url"] = item.User.AvatarURL
			}
		}
	}
	return info
}

func (s *Service) buildReplyTo(ctx context.Context, messageID string) map[string]any {
	info, ok, err := s.repo.GetMessageForReply(ctx, messageID)
	if err != nil || !ok {
		return nil
	}
	return info
}

func (s *Service) buildMedia(ctx context.Context, fileID string) map[string]any {
	info, ok, err := s.repo.GetMediaForMessage(ctx, fileID)
	if err != nil || !ok {
		return nil
	}
	return info
}
