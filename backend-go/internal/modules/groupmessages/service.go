package groupmessages

import (
	"context"
	"errors"
	"sync"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"

	"golang.org/x/sync/errgroup"
)

// groupMentionNotifier 落库群聊 @ 通知（由 notify 模块实现，见 module.go 装配）。
type groupMentionNotifier interface {
	AppendGroupMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, groupID, groupName, messageID, contentPreview string) (map[string]string, error)
}

// shareValidator 校验 share_code 有效性（由 share 模块实现，见 module.go 装配）。
type shareValidator interface {
	ValidateShareCode(ctx context.Context, shareCode string) error
}

var ErrNotGroupMember = errors.New("groupmessages: not a group member")

var ErrUserMuted = errors.New("groupmessages: user is muted")

// ErrInvalidShareCode：share 类型消息的 share_code 无效（不存在/已撤销/已过期）。
var ErrInvalidShareCode = errors.New("groupmessages: invalid share code")

// ErrFileNotOwned：媒体消息的 file_id 不存在、不属于发送者或未完成上传
// （对齐 moments 图片归属口径，spec 10300）。
var ErrFileNotOwned = errors.New("groupmessages: file not owned by sender")

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
	// OpenTheDoor 全量公开主页（9+ 查询），仅保留给低频路径。
	// BatchGet 批量取基础信息（单查询）：发送链路的 WS 负载只需要
	// 昵称/头像，走全量主页是每条消息 3+ 次无谓点查（审计 perf P1-2）。
	BatchGet(context.Context, []string) (map[string]users.UserInfo, error)
}

type Service struct {
	repo      Repository
	users     userReader
	mentions  groupMentionNotifier   // optional; nil => 群聊 @ 不落库通知（测试/降级模式）
	reactions MessageReactionsSource // optional; nil => reactions 恒空数组（降级）
	shares    shareValidator         // optional; nil => 不校验 share_code（测试/降级模式）
	appConfig appconfig.Source       // optional; nil => 撤回窗口回退默认 120s
	// readFlusher 节流聚合 GroupMessageReadCountUpdated 推送（spec chat.md：
	// 3-5 秒窗口最终一致，替代逐人广播）
	readFlusher *readCountFlusher
}

// SetAppConfig wires the dynamic app config source（消息撤回时间窗
// AppConfig.limits.message_recall_window_seconds，spec chat.md）。
func (s *Service) SetAppConfig(source appconfig.Source) { s.appConfig = source }

// recallWindow 返回消息撤回时间窗（spec chat.md → AppConfig.limits.
// message_recall_window_seconds）。appConfig 未接线时回退默认 120s（与
// appconfig 默认配置一致）；配置 0/负值按 0 处理 = 撤回禁用。
func (s *Service) recallWindow(ctx context.Context) time.Duration {
	const fallbackSeconds = 120
	seconds := fallbackSeconds
	if s.appConfig != nil {
		seconds = s.appConfig.Get(ctx).Limits.MessageRecallWindowSeconds
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

// readCountFlusher 按群聚合待推送的已读计数更新，4 秒窗口批量 flush。
type readCountFlusher struct {
	mu      sync.Mutex
	pending map[string]map[string]struct{} // groupID -> messageID set
	timers  map[string]*time.Timer
}

func newReadCountFlusher() *readCountFlusher {
	return &readCountFlusher{pending: map[string]map[string]struct{}{}, timers: map[string]*time.Timer{}}
}

const readCountFlushDelay = 4 * time.Second

// schedule 合并待推送目标；每群至多一个未触发的 timer，窗口内多次 mark_read
// 聚合为一次广播（最终一致：窗口尾必发最新值）。
func (f *readCountFlusher) schedule(groupID string, messageIDs []string, flush func(string, []string)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	set, ok := f.pending[groupID]
	if !ok {
		set = map[string]struct{}{}
		f.pending[groupID] = set
	}
	for _, id := range messageIDs {
		set[id] = struct{}{}
	}
	if _, hasTimer := f.timers[groupID]; hasTimer {
		return
	}
	f.timers[groupID] = time.AfterFunc(readCountFlushDelay, func() {
		// AfterFunc 回调在独立 goroutine 执行：flush 内含 DB 查询与 WS 广播，
		// panic 会击穿整个进程（platform/worker 等后台 goroutine 均有同款
		// recover 口径），此处补齐。
		defer func() {
			if r := recover(); r != nil {
				observability.L().Error("read count flusher panic recovered",
					"group_id", groupID, "panic", r)
			}
		}()
		f.mu.Lock()
		ids := make([]string, 0, len(f.pending[groupID]))
		for id := range f.pending[groupID] {
			ids = append(ids, id)
		}
		delete(f.pending, groupID)
		delete(f.timers, groupID)
		f.mu.Unlock()
		if len(ids) > 0 {
			flush(groupID, ids)
		}
	})
}

func NewService() *Service { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo, users: nil, readFlusher: newReadCountFlusher()}
}
func NewServiceWithRepoAndUsers(repo Repository, users userReader) *Service {
	return &Service{repo: repo, users: users, readFlusher: newReadCountFlusher()}
}
func NewServiceWithDeps(repo Repository, users userReader, mentions groupMentionNotifier, shares shareValidator) *Service {
	return &Service{repo: repo, users: users, mentions: mentions, shares: shares, readFlusher: newReadCountFlusher()}
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
	// 附件归属校验：file_id 必须属于发送者且已完成上传，防止把他人/不存在的
	// 媒体标识固化进消息行（对齐 moments 图片归属校验，spec 10300）。
	if req.FileID != "" {
		if _, ok, err := s.repo.GetMediaForMessage(ctx, req.FileID, userID); err != nil {
			observability.L().Error("groupmessages.send media check failed", "error", err, "file_id", req.FileID)
			return MessageSentResult{}, err
		} else if !ok {
			return MessageSentResult{}, ErrFileNotOwned
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
		if media := s.buildMedia(ctx, req.FileID, userID); media != nil {
			payload["media"] = media
		}
	}
	realtime.DefaultHub().BroadcastToGroup(req.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageNew, Payload: payload})
	// 群聊 @ 通知（spec api_spec_chat.md「群聊 @ 通知」）：对被 @ 的每个成员分别落库+推送
	s.notifyMentions(ctx, userID, req, msg)
	return msg, nil
}

// notifyMentions 对被 @ 的成员批量落库 group_mentioned 通知（单次往返）并以
// 有界并发推送 GroupMessageMentioned 事件（自动推送，无需订阅该群频道）。
func (s *Service) notifyMentions(ctx context.Context, actorUserID string, req SendGroupMessageRequest, msg MessageSentResult) {
	if s.mentions == nil || len(req.Mentions) == 0 {
		return
	}
	// 只对真实群成员落库/推送 @ 通知：非成员收到 content_preview 会泄漏其
	// 无权查看的群消息内容，也构成通知轰炸向量（spec：Mentions 为成员列表）。
	mentionedMembers, err := s.repo.FilterMemberIDs(ctx, req.GroupID, req.Mentions)
	if err != nil {
		observability.L().Error("groupmessages.mentions filter members failed", "error", err, "group_id", req.GroupID)
		return
	}
	targets := make([]string, 0, len(mentionedMembers))
	for _, mentionedUserID := range mentionedMembers {
		if mentionedUserID != actorUserID {
			targets = append(targets, mentionedUserID)
		}
	}
	if len(targets) == 0 {
		return
	}
	actor := s.resolveUserInfo(ctx, actorUserID)
	groupName := s.resolveGroupName(ctx, req.GroupID)
	contentPreview := textutil.TruncateRunes(req.Content, 80)
	// 批量落库（单次往返）：失败时记日志并按通知缺失降级（notify_id 置空），
	// 广播仍照常进行——与此前逐人失败只影响该人的口径一致，只是失败粒度为整批。
	notifyIDs, err := s.mentions.AppendGroupMentionedBatch(ctx, targets, actor, req.GroupID, groupName, msg.MessageID, contentPreview)
	if err != nil {
		observability.L().Error("groupmessages.mentions batch append failed", "error", err, "group_id", req.GroupID, "count", len(targets))
	}
	// 并行推送（有界并发）：每条 BroadcastToUser 内部含 resume Record + 跨实例
	// publish 两次网络往返，逐条串行在多 @ 场景放大发送延迟。
	broadcast := func(mentionedUserID, notificationID string) func() error {
		return func() error {
			realtime.DefaultHub().BroadcastToUser(mentionedUserID, realtime.Event{Type: realtime.EventTypeGroupMessageMentioned, Payload: GroupMessageMentionedPayload{
				NotifyID:       notificationID,
				GroupID:        req.GroupID,
				GroupName:      groupName, // spec websocket.md：与 notify payload 对齐，直接渲染通知标题
				MessageID:      msg.MessageID,
				Actor:          actor,
				ContentPreview: contentPreview,
				MentionedAt:    msg.SentAt,
			}})
			return nil
		}
	}
	group, _ := errgroup.WithContext(context.Background())
	group.SetLimit(8)
	for _, mentionedUserID := range targets {
		group.Go(broadcast(mentionedUserID, notifyIDs[mentionedUserID]))
	}
	_ = group.Wait()
}

// SetReactionsSource 注入消息表情反应聚合源（bootstrap 经 reactions 模块适配）。
func (s *Service) SetReactionsSource(src MessageReactionsSource) { s.reactions = src }

// attachReactions 按契约补齐 reactions 恒在数组（spec api_spec_chat.md：
// 空数组 = 无反应）：无数据源时全部置空数组；有数据源时单次批量聚合
// （reacted_by_me 按当前查看者）。聚合失败降级为空数组并记日志，不使
// 历史拉取整体失败。
func (s *Service) attachReactions(ctx context.Context, items []GroupMessageItem) {
	for i := range items {
		items[i].Reactions = []Reaction{}
	}
	if s.reactions == nil || len(items) == 0 {
		return
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MessageID)
	}
	agg, err := s.reactions.AggregatedGroupMessageReactions(ctx, ids)
	if err != nil {
		observability.L().Warn("groupmessages.attach_reactions failed", "error", err)
		return
	}
	for i := range items {
		if rs, ok := agg[items[i].MessageID]; ok && len(rs) > 0 {
			items[i].Reactions = rs
		}
	}
}

func (s *Service) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureMember(ctx, groupID, principal.UserID); err != nil {
		return PaginatedGroupMessages{}, err
	}
	result, err := s.repo.History(ctx, groupID, pagination)
	if err != nil {
		return PaginatedGroupMessages{}, err
	}
	s.attachReactions(ctx, result.Items)
	return result, nil
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
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageEdited, Payload: GroupMessageEditedPayload{
		GroupID:   meta.GroupID,
		MessageID: meta.MessageID,
		Content:   meta.Content,
		EditedAt:  time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	if err := s.authorizeMessageMutation(ctx, req.MessageID, req.UserID); err != nil {
		return err
	}
	meta, err := s.repo.Recall(ctx, s.recallWindow(ctx), req)
	if err != nil {
		observability.L().Error("groupmessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(meta.GroupID, realtime.Event{Type: realtime.EventTypeGroupMessageRecalled, Payload: GroupMessageRecalledPayload{
		GroupID:    meta.GroupID,
		MessageID:  meta.MessageID,
		RecalledBy: RecalledByUser{UserID: req.UserID},
		RecalledAt: time.Now().UTC().Format(time.RFC3339),
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
	meta2, err := s.repo.RecallAsAdmin(ctx, messageID, operatorID)
	if err != nil {
		observability.L().Error("groupmessages.recall_as_admin failed", "error", err, "message_id", messageID, "group_id", groupID)
		return err
	}
	realtime.DefaultHub().BroadcastToGroup(groupID, realtime.Event{Type: realtime.EventTypeGroupMessageRecalled, Payload: GroupMessageRecalledPayload{
		GroupID:    groupID,
		MessageID:  messageID,
		RecalledBy: RecalledByUser{UserID: operatorID},
		RecalledAt: meta2.SentAt,
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
	// 成员校验对齐 Send/History：非成员此前依赖 SQL 的 JOIN group_members 隐式
	// 拦截并返回 10404，而契约要求 10200 + forbidden_reason=not_group_member。
	if err := s.ensureMember(ctx, req.GroupID, principal.UserID); err != nil {
		return err
	}
	if err := s.repo.MarkAsRead(ctx, principal.UserID, req); err != nil {
		observability.L().Error("groupmessages.mark_as_read failed", "error", err, "group_id", req.GroupID, "up_to_message_id", req.UpToMessageID, "user_id", principal.UserID)
		return err
	}
	// 已读计数变化经节流聚合推送（spec chat.md GroupMessageReadCountUpdated）；
	// 不推送 self-sync 事件（已读状态不做跨设备实时同步，见 api_spec_index.md 预留）
	target := req.UpToMessageID
	if target == "" {
		latest, err := s.repo.LatestMessageID(ctx, req.GroupID)
		if err != nil {
			return nil // 计数推送为尽力而为：拿不到目标消息（空群）不阻塞已读标记
		}
		target = latest
	}
	s.scheduleReadCountUpdate(req.GroupID, []string{target})
	return nil
}

// scheduleReadCountUpdate 进入节流窗口，窗口尾广播该消息的最新 read_count。
func (s *Service) scheduleReadCountUpdate(groupID string, messageIDs []string) {
	if s.readFlusher == nil {
		return
	}
	s.readFlusher.schedule(groupID, messageIDs, s.flushReadCounts)
}

func (s *Service) flushReadCounts(groupID string, messageIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 批量一次取回整窗消息的已读计数（替代逐消息查询的 N 次往返）。
	counts, err := s.repo.GetReadCounts(ctx, groupID, messageIDs)
	if err != nil {
		observability.L().Warn("groupmessages.read_count lookup failed", "error", err, "group_id", groupID, "messages", len(messageIDs))
		return
	}
	for _, messageID := range messageIDs {
		count, ok := counts[messageID]
		if !ok {
			continue
		}
		realtime.DefaultHub().BroadcastToGroup(groupID, realtime.Event{
			Type: realtime.EventTypeGroupMessageReadCountUpdated,
			Payload: GroupMessageReadCountUpdatedPayload{
				GroupID:   groupID,
				MessageID: messageID,
				ReadCount: count,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	groupIDs, err := s.repo.MarkAllRead(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("groupmessages.mark_all_read failed", "error", err, "user_id", principal.UserID)
		return err
	}
	// 每个受影响群以最新消息为计数目标进入节流窗口
	for _, groupID := range groupIDs {
		latest, err := s.repo.LatestMessageID(ctx, groupID)
		if err != nil {
			continue
		}
		s.scheduleReadCountUpdate(groupID, []string{latest})
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

func (s *Service) resolveUserInfo(ctx context.Context, userID string) types.UserInfo {
	info := types.UserInfo{UserID: userID, Nickname: userID}
	if s.users != nil {
		if items, err := s.users.BatchGet(ctx, []string{userID}); err == nil {
			if item, ok := items[userID]; ok {
				info.Nickname = item.Nickname
				info.AvatarURL = item.AvatarURL
			}
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

// GetMessageReportSnapshot 暴露群消息快照给 bootstrap 接线举报模块
// （reports 模块经 MessageSnapshotSource 接口消费，不直接依赖本包）。
func (s *Service) GetMessageReportSnapshot(ctx context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	return s.repo.GetMessageReportSnapshot(ctx, messageID)
}

// IsGroupMember 判定用户是否为群成员（举报受理用它校验举报人是会话
// 参与方；非成员按目标不存在处理，不通过快照泄漏群聊内容）。
func (s *Service) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	return s.repo.IsMember(ctx, groupID, userID)
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
