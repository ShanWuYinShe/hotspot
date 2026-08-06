package privatemessages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

type ReadService interface {
	Recent(context.Context, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, *time.Time) (int, error)
	LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error)
}

type Runtime struct {
	handler     *Handler
	readService ReadService
}

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
	GetMySettings(context.Context) (users.UserSettings, error)
}

// shareValidator 校验 share_code 有效性（由 share 模块实现，见 module.go 装配）。
type shareValidator interface {
	ValidateShareCode(ctx context.Context, shareCode string) error
}

type Service struct {
	repo      Repository
	users     userReader
	appConfig appconfig.Source // optional; nil => stranger quota disabled (0)
	shares    shareValidator   // optional; nil => 不校验 share_code（测试/降级模式）
}

func NewService() *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(NewInMemoryRepository(), users.NewServiceWithRepo(userRepo))
}

func NewServiceWithRepo(repo Repository) *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(repo, users.NewServiceWithRepo(userRepo))
}

func NewServiceWithDeps(repo Repository, userService userReader) *Service {
	return &Service{repo: repo, users: userService}
}

// SetShareValidator wires the share_code validator (spec api_spec_chat.md：
// share 类型消息发送时校验有效性)。
func (s *Service) SetShareValidator(v shareValidator) { s.shares = v }

// SetAppConfig wires the app config source (used for the stranger message
// daily quota). Optional: nil disables the quota check.
func (s *Service) SetAppConfig(source appconfig.Source) { s.appConfig = source }

func NewRuntime(factory *persistence.Factory, source appconfig.Source) *Runtime {
	repo := NewPostgresRepository(factory)
	userService := users.NewServiceWithRepo(users.NewPersistenceRepository(factory))
	service := NewServiceWithDeps(repo, userService)
	service.SetAppConfig(source)
	return &Runtime{
		handler:     NewHandler(service),
		readService: service,
	}
}

func NewRuntimeWithRepo(repo Repository) *Runtime {
	service := NewServiceWithRepo(repo)
	return &Runtime{
		handler:     NewHandler(service),
		readService: service,
	}
}

// SetShareValidator wires the share_code validator into the runtime service.
func (r *Runtime) SetShareValidator(v shareValidator) {
	if svc, ok := r.readService.(*Service); ok {
		svc.SetShareValidator(v)
	}
}

func (r *Runtime) Handler() *Handler {
	return r.handler
}

func (r *Runtime) ReadService() ReadService {
	return r.readService
}

func (s *Service) Send(ctx context.Context, req SendPrivateMessageRequest) (MessageSent, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureCanSend(ctx, principal.UserID, req.TargetUserID); err != nil {
		return MessageSent{}, err
	}
	// share 类型：校验 share_code 存在且有效（spec api_spec_chat.md，防发送失效链接）
	if req.ShareCode != "" && s.shares != nil {
		if err := s.shares.ValidateShareCode(ctx, req.ShareCode); err != nil {
			return MessageSent{}, ErrInvalidShareCode
		}
	}
	msg, err := s.repo.Send(ctx, principal.UserID, req.TargetUserID, req)
	if err != nil {
		observability.L().Error("privatemessages.send failed", "error", err, "target_user_id", req.TargetUserID)
		return MessageSent{}, err
	}
	payload := map[string]any{
		"message_id":     msg.MessageID,
		"target_user_id": req.TargetUserID,
		"sender":         s.resolveUserInfo(ctx, principal.UserID),
		"type":           req.Type,
		"sent_at":        msg.SentAt,
	}
	if req.Content != "" {
		payload["content"] = req.Content
	}
	if req.ShareCode != "" {
		payload["share_code"] = req.ShareCode
	}
	if req.ReplyToMessageID != "" {
		if replyTo := s.buildReplyTo(ctx, req.ReplyToMessageID, principal.UserID); replyTo != nil {
			payload["reply_to"] = replyTo
		}
	}
	if req.FileID != "" {
		if media := s.buildMedia(ctx, req.FileID, principal.UserID); media != nil {
			payload["media"] = media
		}
	}
	event := realtime.Event{Type: realtime.EventTypePrivateMessageNew, Payload: payload}
	realtime.DefaultHub().BroadcastToUser(req.TargetUserID, event)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	return msg, nil
}

func (s *Service) History(ctx context.Context, req GetHistoryRequest) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.History(ctx, principal.UserID, req)
}

func (s *Service) Recent(ctx context.Context, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Recent(ctx, principal.UserID, targetUserID, limit, deletedAfter)
}

func (s *Service) Last(ctx context.Context, targetUserID string, deletedAfter *time.Time) (*PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Last(ctx, principal.UserID, targetUserID, deletedAfter)
}

func (s *Service) UnreadCount(ctx context.Context, targetUserID string, deletedAfter *time.Time) (int, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCount(ctx, principal.UserID, targetUserID, deletedAfter)
}

func (s *Service) Update(ctx context.Context, req UpdatePrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.Update(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.update failed", "error", err, "message_id", req.MessageID)
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	event := realtime.Event{Type: realtime.EventTypePrivateMessageEdited, Payload: map[string]any{
		"message_id":     req.MessageID,
		"target_user_id": meta.TargetUserID,
		"content":        meta.Content,
		"edited_at":      now,
	}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallPrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.Recall(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	event := realtime.Event{Type: realtime.EventTypePrivateMessageRecalled, Payload: map[string]any{
		"message_id":     req.MessageID,
		"target_user_id": meta.TargetUserID,
		"recalled_by":    s.resolveUserInfo(ctx, principal.UserID),
		"recalled_at":    now,
	}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkPrivateMessagesReadRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.MarkAsRead(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.mark_as_read failed", "error", err, "target_user_id", req.TargetUserID, "up_to_message_id", req.UpToMessageID)
		return err
	}
	event := realtime.Event{Type: realtime.EventTypePrivateMessageRead, Payload: map[string]any{
		"target_user_id":   meta.TargetUserID,
		"up_to_message_id": meta.MessageID,
		"reader_id":        principal.UserID,
		"read_at":          time.Now().UTC().Format(time.RFC3339Nano),
	}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	if principal.UserID != meta.TargetUserID {
		realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfReadSync, Payload: map[string]any{
		"chat_type":        "private",
		"target_id":        req.TargetUserID,
		"up_to_message_id": meta.MessageID,
	}})
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("privatemessages.mark_all_read failed", "error", err)
		return err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: realtime.EventTypeSelfReadSync, Payload: map[string]any{
		"chat_type": "private",
	}})
	return nil
}

func (s *Service) ensureCanSend(ctx context.Context, userID, targetUserID string) error {
	if s.users == nil {
		return nil
	}
	item, ok, err := s.users.OpenTheDoor(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		return users.ErrUserNotFound
	}
	if userID == targetUserID {
		return nil
	}
	// 拉黑明确返回 'blocked'（产品决策：拉黑就显示拉黑，见 api_spec_chat.md BlockReason）
	if item.Relation != nil && item.Relation.IsBlocked {
		return &MessageBlockedError{Reason: BlockReasonBlocked}
	}
	settings, settingsErr := s.users.GetMySettings(platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: targetUserID, SessionID: "private-message-probe"}))
	if settingsErr != nil {
		return settingsErr
	}
	isMutual := item.Relation != nil && item.Relation.IsFollowing && item.Relation.IsFollowedBy
	switch settings.AllowPrivateChat {
	case users.ChatPermNobody:
		return &MessageBlockedError{Reason: BlockReasonPrivacy}
	case users.ChatPermFriends:
		if !isMutual {
			return &MessageBlockedError{Reason: BlockReasonPrivacy}
		}
	}
	// 陌生人消息每日额度:仅对非互相关注用户生效(0 = 禁止任何陌生人消息)
	if !isMutual {
		if err := s.enforceStrangerQuota(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// enforceStrangerQuota checks AppConfig.limits.stranger_message_daily_quota:
// 0 disables stranger messages entirely; otherwise the total volume of
// messages sent today to non-mutual (stranger) targets counts against the
// quota (total-volume accounting across ALL stranger targets, not per-target).
// When no app config is wired (tests/degraded mode) the check is skipped.
func (s *Service) enforceStrangerQuota(ctx context.Context, userID string) error {
	if s.appConfig == nil {
		return nil
	}
	quota := s.appConfig.Get(ctx).Limits.StrangerMessageDailyQuota
	if quota < 0 {
		quota = 0
	}
	if quota == 0 {
		return &MessageBlockedError{Reason: BlockReasonStrangerQuotaExceeded}
	}
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	counts, err := s.repo.CountMessagesByTarget(ctx, userID, startOfDay)
	if err != nil {
		return err
	}
	strangerTotal := 0
	for targetID, count := range counts {
		if targetID == userID {
			continue
		}
		// 用发送者视角的 ctx 查询 relation:互相关注(双向)不算陌生人。
		item, ok, err := s.users.OpenTheDoor(ctx, targetID)
		if err != nil || !ok {
			// 目标查询失败/不存在:无法判定关系,跳过不计入。
			continue
		}
		isMutual := item.Relation != nil && item.Relation.IsFollowing && item.Relation.IsFollowedBy
		if !isMutual {
			strangerTotal += count
		}
	}
	if strangerTotal >= quota {
		return &MessageBlockedError{Reason: BlockReasonStrangerQuotaExceeded}
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

func (s *Service) buildReplyTo(ctx context.Context, messageID, viewerUserID string) map[string]any {
	info, ok, err := s.repo.GetMessageForReply(ctx, messageID, viewerUserID)
	if err != nil || !ok {
		return nil
	}
	return info
}

func (s *Service) buildMedia(ctx context.Context, fileID, ownerUserID string) map[string]any {
	info, ok, err := s.repo.GetMediaForMessage(ctx, fileID, ownerUserID)
	if err != nil || !ok {
		return nil
	}
	return info
}

func (s *Service) LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error) {
	return s.repo.MessageExists(ctx, messageID)
}
