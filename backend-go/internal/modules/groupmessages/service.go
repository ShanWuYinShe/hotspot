package groupmessages

import (
	"context"
	"errors"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// groupMentionNotifier 落库群聊 @ 通知（由 notify 模块实现，见 module.go 装配）。
type groupMentionNotifier interface {
	AppendGroupMentioned(ctx context.Context, targetUserID string, actor types.UserInfo, groupID, groupName, messageID, contentPreview string) (string, error)
}

// shareValidator 校验 share_code 有效性（由 share 模块实现，见 module.go 装配）。
type shareValidator interface {
	ValidateShareCode(ctx context.Context, shareCode string) error
}

var ErrNotGroupMember = errors.New("groupmessages: not a group member")

var ErrUserMuted = errors.New("groupmessages: user is muted")

// ErrInvalidShareCode：share 类型消息的 share_code 无效（不存在/已撤销/已过期）。
var ErrInvalidShareCode = errors.New("groupmessages: invalid share code")

type MutedError struct {
	MutedUntil string
}

func (e *MutedError) Error() string {
	if e == nil {
		return "groupmessages: user is muted"
	}
	return "groupmessages: user is muted"
}

func (e *MutedError) Unwrap() error {
	return ErrUserMuted
}

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
}

type Service struct {
	repo     Repository
	users    userReader
	mentions groupMentionNotifier // optional; nil => 群聊 @ 不落库通知（测试/降级模式）
	shares   shareValidator       // optional; nil => 不校验 share_code（测试/降级模式）
}

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo, users: nil} }
func NewServiceWithRepoAndUsers(repo Repository, users userReader) *Service {
	return &Service{repo: repo, users: users}
}
func NewServiceWithDeps(repo Repository, users userReader, mentions groupMentionNotifier, shares shareValidator) *Service {
	return &Service{repo: repo, users: users, mentions: mentions, shares: shares}
}
func (s *Service) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSentResult, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return MessageSentResult{}, ErrNotGroupMember
	}
	userID := principal.UserID
	if err := s.ensureMember(ctx, req.GroupID, userID); err != nil {
		return MessageSentResult{}, err
	}
	if isMuted, mutedUntil, found, err := s.repo.MemberMuteStatus(ctx, req.GroupID, userID); err != nil {
		observability.L().Error("groupmessages.send mute check failed", "error", err, "group_id", req.GroupID)
		return MessageSentResult{}, err
	} else if found && isMuted {
		return MessageSentResult{}, &MutedError{MutedUntil: mutedUntil}
	}
	// share 类型：校验 share_code 存在且有效（spec api_spec_chat.md，防发送失效链接）
	if req.ShareCode != "" && s.shares != nil {
		if err := s.shares.ValidateShareCode(ctx, req.ShareCode); err != nil {
			return MessageSentResult{}, ErrInvalidShareCode
		}
	}
	msg, err := s.repo.Send(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.send failed", "error", err, "group_id", req.GroupID)
		return MessageSentResult{}, err
	}
	payload := map[string]any{
		"message_id": msg.MessageID,
		"group_id":   req.GroupID,
		"sender":     s.resolveUserInfo(ctx, userID),
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
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageNew, Payload: payload})
	// 群聊 @ 通知（spec api_spec_chat.md「群聊 @ 通知」）：对被 @ 的每个成员分别落库+推送
	s.notifyMentions(ctx, userID, req, msg)
	return msg, nil
}

// notifyMentions 对被 @ 的成员逐人落库 group_mentioned 通知并推送
// GroupMessageMentioned 事件（自动推送，无需订阅该群频道）。
func (s *Service) notifyMentions(ctx context.Context, actorUserID string, req SendGroupMessageRequest, msg MessageSentResult) {
	if s.mentions == nil || len(req.Mentions) == 0 {
		return
	}
	actor := s.resolveUserInfo(ctx, actorUserID)
	groupName := s.resolveGroupName(ctx, req.GroupID)
	contentPreview := strings.TrimSpace(req.Content)
	if len(contentPreview) > 80 {
		contentPreview = contentPreview[:80]
	}
	for _, mentionedUserID := range req.Mentions {
		if mentionedUserID == actorUserID {
			continue
		}
		notificationID := ""
		id, err := s.mentions.AppendGroupMentioned(ctx, mentionedUserID, actor, req.GroupID, groupName, msg.MessageID, contentPreview)
		if err != nil {
			observability.L().Error("groupmessages.mentions append failed", "error", err, "group_id", req.GroupID, "mentioned_user_id", mentionedUserID)
		} else {
			notificationID = id
		}
		realtime.DefaultHub().BroadcastToUser(mentionedUserID, realtime.Event{Type: realtime.EventTypeGroupMessageMentioned, Payload: map[string]any{
			"notify_id":       notificationID,
			"group_id":        req.GroupID,
			"group_name":      groupName, // spec websocket.md：与 notify payload 对齐，直接渲染通知标题
			"message_id":      msg.MessageID,
			"actor":           actor,
			"content_preview": contentPreview,
			"mentioned_at":    msg.SentAt,
		}})
	}
}
func (s *Service) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureMember(ctx, groupID, principal.UserID); err != nil {
		return PaginatedGroupMessages{}, err
	}
	return s.repo.History(ctx, groupID, pagination)
}

func (s *Service) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	if err := s.authorizeMessageMutation(ctx, req.MessageID, req.UserID); err != nil {
		return err
	}
	meta, err := s.repo.Update(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.update failed", "error", err, "message_id", req.MessageID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageEdited, Payload: map[string]any{
		"group_id":   meta.GroupID,
		"message_id": meta.MessageID,
		"content":    meta.Content,
		"edited_at":  time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	if err := s.authorizeMessageMutation(ctx, req.MessageID, req.UserID); err != nil {
		return err
	}
	meta, err := s.repo.Recall(ctx, req)
	if err != nil {
		observability.L().Error("groupmessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageRecalled, Payload: map[string]any{
		"group_id":    meta.GroupID,
		"message_id":  meta.MessageID,
		"recalled_by": map[string]any{"user_id": req.UserID},
		"recalled_at": time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

// RecallMessageAsAdmin 供 groups/admin/messages/recall 调用（spec api_spec_group.md）：
// 群管理员/群主撤回群内任意成员发送的消息。与 group_messages/recall（仅作者、受时间窗
// 约束）角色分离：本函数不做作者校验、不受撤回时间窗约束，但消息必须属于 groupID、
// 存在且未被撤回。守卫（owner/admin）在 groups 模块完成，此处不重复鉴权。
// 注意：本函数不注册任何 HTTP 路由；groups 模块的群管理守卫端点直接调用它。
func (s *Service) RecallMessageAsAdmin(ctx context.Context, operatorID, groupID, messageID string) error {
	meta, found, err := s.repo.LookupMessageMeta(ctx, messageID)
	if err != nil {
		return err
	}
	if !found || meta.GroupID != groupID {
		return ErrMessageNotFound
	}
	meta2, err := s.repo.RecallAsAdmin(ctx, messageID)
	if err != nil {
		observability.L().Error("groupmessages.recall_as_admin failed", "error", err, "message_id", messageID, "group_id", groupID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(groupID, realtime.Event{Type: realtime.EventTypeGroupMessageRecalled, Payload: map[string]any{
		"group_id":    groupID,
		"message_id":  messageID,
		"recalled_by": map[string]any{"user_id": operatorID},
		"recalled_at": meta2.SentAt,
	}})
	return nil
}

// authorizeMessageMutation enforces the two invariants that Update/Recall
// require before touching a message:
//  1. the caller must still be a member of the group the message lives in
//     (kicked/left users cannot edit history);
//  2. the caller must be the message's sender — NULL/anonymous senders are
//     refused for everyone, so legacy rows can never be mutated.
func (s *Service) authorizeMessageMutation(ctx context.Context, messageID, userID string) error {
	meta, found, err := s.repo.LookupMessageMeta(ctx, messageID)
	if err != nil {
		return err
	}
	if !found {
		return ErrMessageNotFound
	}
	if err := s.ensureMember(ctx, meta.GroupID, userID); err != nil {
		return err
	}
	if meta.SenderUserID == "" || meta.SenderUserID != userID {
		return ErrPermissionDenied
	}
	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkGroupMessagesReadRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAsRead(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groupmessages.mark_as_read failed", "error", err, "group_id", req.GroupID, "up_to_message_id", req.UpToMessageID, "user_id", principal.UserID)
		return err
	}
	// 不推送 self-sync 事件（已读状态不做跨设备实时同步，见 api_spec_index.md 预留）
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("groupmessages.mark_all_read failed", "error", err, "user_id", principal.UserID)
		return err
	}
	// 不推送 self-sync 事件（已读状态不做跨设备实时同步，见 api_spec_index.md 预留）
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

func (s *Service) resolveUserInfo(ctx context.Context, userID string) types.UserInfo {
	info := types.UserInfo{UserID: userID, Nickname: userID}
	if s.users != nil {
		if item, ok, err := s.users.OpenTheDoor(ctx, userID); err == nil && ok {
			info.Nickname = item.User.Nickname
			info.AvatarURL = item.User.AvatarURL
		}
	}
	return info
}

func (s *Service) buildReplyTo(ctx context.Context, messageID string) *MessageBrief {
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

func (s *Service) LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error) {
	return s.repo.MessageExists(ctx, messageID)
}

// IsReactionTargetVisible 判定群消息对当前调用方可反应（spec reaction.md 目标可见性）：
// 仅群成员可见（被移除/退群用户对历史消息不可见）；不可见按不存在处理。
func (s *Service) IsReactionTargetVisible(ctx context.Context, messageID string) (found bool, visible bool, err error) {
	groupID, found, err := s.repo.MessageGroupID(ctx, messageID)
	if err != nil || !found {
		return found, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	isMember, err := s.repo.IsMember(ctx, groupID, principal.UserID)
	if err != nil {
		return false, false, err
	}
	return true, isMember, nil
}

// resolveGroupName 解析群名用于 @ 通知负载（查询失败时回退为空串，不阻塞发送）。
func (s *Service) resolveGroupName(ctx context.Context, groupID string) string {
	name, ok, err := s.repo.GetGroupName(ctx, groupID)
	if err != nil || !ok {
		return ""
	}
	return name
}
