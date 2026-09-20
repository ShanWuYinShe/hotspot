package privatemessages

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"

	"golang.org/x/sync/errgroup"
)

type ReadService interface {
	Recent(context.Context, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, *time.Time) (int, error)
	// LastByChatKeys 批量取当前用户各会话的最近一条可见消息（会话列表一次
	// 往返，替代逐会话 Last 的 N+1）。boundaries 按 chat_key 给出可见性下界
	//（visibleBoundary；缺失的 key 视为 nil 不限）。会话无消息时不出现在
	// 结果 map 中。
	LastByChatKeys(ctx context.Context, chatKeys []string, boundaries map[string]*time.Time) (map[string]PrivateMessageItem, error)
	// UnreadCountsByChatKeys 批量取当前用户各会话未读数（单条 GROUP BY
	// 往返，口径与 UnreadCount 一致；无未读的 key 不出现在结果 map 中）。
	UnreadCountsByChatKeys(ctx context.Context, chatKeys []string, boundaries map[string]*time.Time) (map[string]int, error)
	LookupMessage(ctx context.Context, messageID string) (found bool, senderID string, err error)
	// IsReactionTargetVisible 判定消息对当前调用方可反应（spec reaction.md 目标可见性）
	IsReactionTargetVisible(ctx context.Context, messageID string) (found bool, visible bool, err error)
	// GetMessageReportSnapshot 暴露私聊消息快照给 bootstrap 接线举报模块
	//（reports 模块经 MessageSnapshotSource 接口消费，不直接依赖本包）。
	GetMessageReportSnapshot(ctx context.Context, messageID string) (MessageReportSnapshot, bool, error)
}

type Runtime struct {
	handler     *Handler
	readService ReadService
}

type userReader interface {
	// OpenTheDoor 全量公开主页（9+ 查询）。仅保留给低频路径；发送链路已改用
	// BatchGet/Relationship 轻量查询（审计 perf P1-1），不要新增调用。
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
	// BatchGet 批量取基础信息（nickname/avatar/存在性，单查询）。
	BatchGet(context.Context, []string) (map[string]users.UserInfo, error)
	// SettingsFor 按 user_id 纯读目标用户设置（无写副作用）。此前用
	// GetMySettings 伪造对方 Principal 探测，其「设置行缺失即写默认行」的
	// 副作用会为对方凭空插入 settings 行（审计 F-10）。
	SettingsFor(context.Context, string) (users.UserSettings, bool, error)
	BlockStateBetween(context.Context, string, string) (callerBlocked, targetBlocked bool, err error)
	Relationship(context.Context, string, string) (users.Relationship, error)
	// RelationshipBatch 批量互关判定（单次往返，陌生人配额消费）。
	RelationshipBatch(context.Context, string, []string) (map[string]users.Relationship, error)
}

// shareValidator 校验 share_code 有效性（由 share 模块实现，见 module.go 装配）。
type shareValidator interface {
	ValidateShareCode(ctx context.Context, shareCode string) error
}

type Service struct {
	repo      Repository
	users     userReader
	appConfig appconfig.Source       // optional; nil => stranger quota disabled (0)
	shares    shareValidator         // optional; nil => 不校验 share_code（测试/降级模式）
	reactions MessageReactionsSource // optional; nil => reactions 恒空数组（降级）
	// tx 事务执行器（bootstrap/NewRuntime 注入）：陌生人配额的检查与消息写入
	// 须在同一事务内经咨询锁串行化；nil（测试）时 fn 直接执行。
	tx persistence.TxRunner
}

// SetTxRunner 注入事务执行器（装配时调用一次）。配额检查与 repo.Send 共享
// 同一事务是 check-then-act 原子化的前提（见 Send）。
func (s *Service) SetTxRunner(tx persistence.TxRunner) { s.tx = tx }

// runInTx 在事务内执行 fn；执行器未接线（测试/降级）时直接执行。
func (s *Service) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx(ctx, fn)
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

func NewRuntime(factory *persistence.Factory, source appconfig.Source, userService userReader) *Runtime {
	// userService 由调用方注入（bootstrap 传共享的 users.Service 实例）：
	// 模块边界约定「不 import 对方 repo」，此前在此处自建 users repo/service
	// 造成 bootstrap 之外多出一份无谓实例。
	repo := NewPostgresRepository(factory)
	service := NewServiceWithDeps(repo, userService)
	service.SetAppConfig(source)
	// 陌生人配额原子化：配额检查与消息写入须在同一事务内经咨询锁串行化（见 Send）。
	service.SetTxRunner(persistence.NewTxRunner(factory))
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

// SetReactionsSource 把消息表情反应聚合源接进 runtime service（bootstrap 经
// privatechats.Module 转发）。
func (r *Runtime) SetReactionsSource(src MessageReactionsSource) {
	if svc, ok := r.readService.(*Service); ok {
		svc.SetReactionsSource(src)
	}
}

func (r *Runtime) Handler() *Handler {
	return r.handler
}

func (r *Runtime) ReadService() ReadService {
	return r.readService
}

func (s *Service) Send(ctx context.Context, req SendPrivateMessageRequest) (MessageSentResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	var sent MessageSentResult
	// 附件归属校验：file_id 必须属于发送者且已完成上传，防止把他人/不存在的
	// 媒体标识固化进消息行（对齐 moments 图片归属校验，spec 10300）。
	// 放事务外：该校验不参与配额原子性，失败早退。
	if req.FileID != "" {
		if _, ok, err := s.repo.GetMediaForMessage(ctx, req.FileID, principal.UserID); err != nil {
			observability.L().Error("privatemessages.send media check failed", "error", err, "file_id", req.FileID)
			return MessageSentResult{}, err
		} else if !ok {
			return MessageSentResult{}, ErrFileNotOwned
		}
	}
	// 前置校验（事务外）：拉黑/隐私/互关判定与配额原子性无关，留在事务内
	// 只会让每条消息占住连接池连接执行十余次查询（审计 perf P1-1）。事务内
	// 仅保留配额咨询锁 + COUNT + INSERT 的原子序列。
	quotaApplies, err := s.ensureCanSend(ctx, principal.UserID, req.TargetUserID)
	if err != nil {
		return MessageSentResult{}, err
	}
	// 配额检查（enforceStrangerQuota 的 COUNT）与消息写入（repo.Send 的 INSERT）
	// 必须在同一事务内：先取「发送者 + UTC 日期」维度的咨询锁，再 COUNT、
	// 再 INSERT——并发同发送者的请求在此串行，后到者的 COUNT 能看到先到者
	// 已提交的消息，消除 check-then-act 双双超额的竞态。
	err = s.runInTx(ctx, func(txCtx context.Context) error {
		// share 类型：校验 share_code 存在且有效（spec api_spec_chat.md，防发送失效链接）
		if req.ShareCode != "" && s.shares != nil {
			if err := s.shares.ValidateShareCode(txCtx, req.ShareCode); err != nil {
				return ErrInvalidShareCode
			}
		}
		if quotaApplies {
			if err := s.repo.LockStrangerQuota(txCtx, principal.UserID, utcDayKey(time.Now())); err != nil {
				return err
			}
			if err := s.enforceStrangerQuota(txCtx, principal.UserID); err != nil {
				return err
			}
		}
		msg, err := s.repo.Send(txCtx, principal.UserID, req.TargetUserID, req)
		if err != nil {
			observability.L().Error("privatemessages.send failed", "error", err, "target_user_id", req.TargetUserID)
			return err
		}
		sent = msg
		return nil
	})
	if err != nil {
		return MessageSentResult{}, err
	}
	msg := sent
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
	// 收发双方各一次 BroadcastToUser（每次内部 record+publish 两次 Redis 往返），
	// 串行即 4 RTT（审计 perf P1-4）。两条广播目标集合不相交、同一事件，
	// 并行无顺序语义；同步等待两条完成才返回，失败处理与现状一致（广播失败
	// 不影响消息已落库的返回结果）。不异步化：HTTP 响应前必须保证投递已发起。
	group, _ := errgroup.WithContext(context.Background())
	group.Go(func() error {
		realtime.DefaultHub().BroadcastToUser(req.TargetUserID, event)
		return nil
	})
	group.Go(func() error {
		realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
		return nil
	})
	_ = group.Wait()
	return msg, nil
}

// SetReactionsSource 注入消息表情反应聚合源（bootstrap 经 reactions 模块适配）。
func (s *Service) SetReactionsSource(src MessageReactionsSource) { s.reactions = src }

// attachReactions 按契约补齐 reactions 恒在数组（spec api_spec_chat.md：
// 空数组 = 无反应）：无数据源时全部置空数组；有数据源时单次批量聚合
// （reacted_by_me 按当前查看者）。聚合失败降级为空数组并记日志，不让
// 历史拉取整体失败。
func (s *Service) attachReactions(ctx context.Context, items []PrivateMessageItem) {
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
	agg, err := s.reactions.AggregatedMessageReactions(ctx, ids)
	if err != nil {
		observability.L().Warn("privatemessages.attach_reactions failed", "error", err)
		return
	}
	for i := range items {
		if rs, ok := agg[items[i].MessageID]; ok && len(rs) > 0 {
			items[i].Reactions = rs
		}
	}
}

func (s *Service) History(ctx context.Context, req ListPrivateMessagesRequest) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.History(ctx, principal.UserID, req)
	if err != nil {
		return PaginatedMessages{}, err
	}
	s.attachReactions(ctx, result.Items)
	return result, nil
}

func (s *Service) Recent(ctx context.Context, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.Recent(ctx, principal.UserID, targetUserID, limit, deletedAfter)
	if err != nil {
		return PaginatedMessages{}, err
	}
	s.attachReactions(ctx, result.Items)
	return result, nil
}

func (s *Service) Last(ctx context.Context, targetUserID string, deletedAfter *time.Time) (*PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Last(ctx, principal.UserID, targetUserID, deletedAfter)
}

func (s *Service) UnreadCount(ctx context.Context, targetUserID string, deletedAfter *time.Time) (int, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCount(ctx, principal.UserID, targetUserID, deletedAfter)
}

// LastByChatKeys 会话列表批量读（viewer 经 principal 解析；chat_keys 须为
// 当前用户自己的会话，越权 key 查不到任何行，天然无泄露面）。
func (s *Service) LastByChatKeys(ctx context.Context, chatKeys []string, boundaries map[string]*time.Time) (map[string]PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.LastByChatKeys(ctx, principal.UserID, chatKeys, boundaries)
}

// UnreadCountsByChatKeys 会话列表批量读（viewer 经 principal 解析）。
func (s *Service) UnreadCountsByChatKeys(ctx context.Context, chatKeys []string, boundaries map[string]*time.Time) (map[string]int, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCountsByChatKeys(ctx, principal.UserID, chatKeys, boundaries)
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
	meta, err := s.repo.Recall(ctx, principal.UserID, s.recallWindow(ctx), req)
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
	// 契约(api_spec_websocket.md):PrivateMessageRead 仅推给私聊对端（让对方得知已读）。
	// 不推给已读方自己的其他设备——跨设备已读同步不在本协议范围（self-sync 已移除）。
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("privatemessages.mark_all_read failed", "error", err)
		return err
	}
	// 不推送 self-sync 事件（已读状态不做跨设备实时同步，见 api_spec_index.md 预留）
	return nil
}

// ensureCanSend 校验发送前置条件：目标存在、非拉黑、对方隐私设置允许。
// 返回 quotaApplies 表示随后需要做陌生人配额检查：self-send 有意绕过配额
// （给自己发的消息不占陌生人额度，配额的总量口径在 enforceStrangerQuota 中
// 同样排除 self 目标）；互相关注不在配额约束范围（spec chat.md 陌生人消息
// 每日额度仅约束非互关目标）。
//
// 轻量化（审计 perf P1-1）：仅用 BatchGet（存在性，1 查询）+ BlockStateBetween +
// SettingsFor + Relationship（互关）共 4 次点查，不再走 OpenTheDoor 全量主页
// （9+ 查询，含 stats 五重 COUNT 与社交上下文拼装）。调用方在事务外调用——
// 这些前置校验不参与配额原子性，留在事务内只会占住连接池连接。
func (s *Service) ensureCanSend(ctx context.Context, userID, targetUserID string) (quotaApplies bool, err error) {
	if s.users == nil {
		return false, nil
	}
	existing, err := s.users.BatchGet(ctx, []string{targetUserID})
	if err != nil {
		return false, err
	}
	if _, ok := existing[targetUserID]; !ok {
		return false, users.ErrUserNotFound
	}
	if userID == targetUserID {
		// self-send 绕过配额（有意口径，见函数注释）；配额不约束自己给自己发。
		return false, nil
	}
	// 拉黑明确返回 blocked_by_user（产品决策：拉黑就显示拉黑，见 api_spec_errors.md ForbiddenReason）。
	// 必须查双向黑名单：被对方拉黑的发送者才是拦截目标。
	callerBlocked, targetBlocked, blockErr := s.users.BlockStateBetween(ctx, userID, targetUserID)
	if blockErr != nil {
		return false, blockErr
	}
	if targetBlocked {
		return false, httpx.Forbidden(httpx.ReasonBlockedByUser, "已被对方拉黑")
	}
	if callerBlocked {
		return false, httpx.Forbidden(httpx.ReasonBlockedByUser, "你已拉黑对方，无法发送")
	}
	settings, settingsFound, settingsErr := s.users.SettingsFor(ctx, targetUserID)
	if settingsErr != nil {
		return false, settingsErr
	}
	allowPrivateChat := users.ChatPermEveryone
	if settingsFound {
		allowPrivateChat = settings.AllowPrivateChat
	}
	// 对方从未保存设置时按默认处理：允许所有人私聊（与 users.defaultSettings 的
	// ChatPermEveryone 一致，见 spec 用户设置默认值）。
	isMutual := false
	if rel, relErr := s.users.Relationship(ctx, userID, targetUserID); relErr == nil {
		isMutual = rel.IsFollowing && rel.IsFollowedBy
	} else {
		return false, relErr
	}
	switch allowPrivateChat {
	case users.ChatPermNobody:
		return false, httpx.Forbidden(httpx.ReasonPrivacySetting, "对方隐私设置不允许")
	case users.ChatPermFriends:
		if !isMutual {
			return false, httpx.Forbidden(httpx.ReasonPrivacySetting, "对方隐私设置不允许")
		}
	}
	// 陌生人消息每日额度:仅对非互相关注用户生效(0 = 禁止任何陌生人消息)。
	// 检查本身在 Send 的事务闭包内、咨询锁之后执行（配额原子化）。
	return !isMutual, nil
}

// utcDayKey 返回 t 的 UTC 日期键（yyyy-mm-dd），配额咨询锁与计数窗口共用
// 同一日界。
func utcDayKey(t time.Time) string {
	u := t.UTC()
	return fmt.Sprintf("%04d-%02d-%02d", u.Year(), u.Month(), u.Day())
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
		return httpx.Forbidden(httpx.ReasonStrangerQuotaExceeded, "陌生人消息限额已用完")
	}
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	counts, err := s.repo.CountMessagesByTarget(ctx, userID, startOfDay)
	if err != nil {
		return err
	}
	targetIDs := make([]string, 0, len(counts))
	for targetID := range counts {
		targetIDs = append(targetIDs, targetID)
	}
	// 批量互关判定（单次往返）：此前按目标逐条 Relationship，今日发过 50 个
	// 陌生人即 50 次点查，且全部在事务内执行。批量失败回退逐条（单条错误
	// 跳过不计入，与历史语义一致）。
	mutual := make(map[string]bool, len(targetIDs))
	if len(targetIDs) > 0 && s.users != nil {
		rels, err := s.users.RelationshipBatch(ctx, userID, targetIDs)
		if err != nil {
			observability.L().Warn("privatemessages.quota relationship batch failed, falling back", "error", err, "user_id", userID)
			for _, targetID := range targetIDs {
				rel, relErr := s.users.Relationship(ctx, userID, targetID)
				if relErr == nil {
					mutual[targetID] = rel.IsFollowing && rel.IsFollowedBy
				}
			}
		} else {
			for targetID, rel := range rels {
				mutual[targetID] = rel.IsFollowing && rel.IsFollowedBy
			}
		}
	}
	strangerTotal := 0
	for targetID, count := range counts {
		if targetID == userID {
			continue
		}
		if !mutual[targetID] {
			strangerTotal += count
		}
	}
	if strangerTotal >= quota {
		return httpx.Forbidden(httpx.ReasonStrangerQuotaExceeded, "陌生人消息限额已用完")
	}
	return nil
}

func (s *Service) resolveUserInfo(ctx context.Context, userID string) map[string]any {
	info := map[string]any{"user_id": userID}
	if s.users != nil {
		// 轻量取昵称/头像（单查询）：此前走 OpenTheDoor 全量主页（9+ 查询，
		// 含 stats 五重 COUNT），而 WS 负载只需要 basic info（审计 perf P1-1）。
		if items, err := s.users.BatchGet(ctx, []string{userID}); err == nil {
			if item, ok := items[userID]; ok {
				info["nickname"] = item.Nickname
				if item.AvatarURL != nil && *item.AvatarURL != "" {
					info["avatar_url"] = *item.AvatarURL
				}
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

// GetMessageReportSnapshot 暴露私聊消息快照给 bootstrap 接线举报模块
// （reports 模块经 MessageSnapshotSource 接口消费，不直接依赖本包）。
func (s *Service) GetMessageReportSnapshot(ctx context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	return s.repo.GetMessageReportSnapshot(ctx, messageID)
}

// IsReactionTargetVisible 判定私聊消息对当前调用方可反应（spec reaction.md 目标可见性）：
// 仅会话参与者可见；不可见按不存在处理。
func (s *Service) IsReactionTargetVisible(ctx context.Context, messageID string) (found bool, visible bool, err error) {
	senderID, receiverID, found, err := s.repo.MessageParticipants(ctx, messageID)
	if err != nil || !found {
		return found, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return true, principal.UserID == senderID || principal.UserID == receiverID, nil
}
