package notify

import (
	"context"
	"sync"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
	"hotspot/backend-go/internal/platform/worker"
)

// 通知 prune 参数（保守值）：已读通知保留 90 天后物理删除，每 24h 扫一轮。
// 通知只增不删会无限增长（notifications 表随交互历史线性膨胀），删除仅限
// 「已读 + 创建超保留期」——未读通知与近期通知不受影响。表无 updated_at 列，
// 以 created_at 近似已读年龄，实际只会少删（见 repo.PruneReadBefore）。
const (
	notifyPruneRetention = 90 * 24 * time.Hour
	notifyPruneInterval  = 24 * time.Hour
)

var (
	pruneMu      sync.Mutex
	pruneCancel  context.CancelFunc
	pruneStarted bool
)

type Service struct {
	repo                 Repository
	nearbyNotifyProvider NearbyNotifySettingsProvider
}

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

// StartPruneSweeper 启动后台 prune（对齐 idempotency.StartSweeper 模式）：
// 每 notifyPruneInterval 物理删除一次「已读且超 notifyPruneRetention」的
// 通知。重复调用为 no-op；goroutine 绑定 ctx（应用根上下文），优雅停机时
// 退出。调用方（bootstrap）经配置开关决定是否启动。
func (s *Service) StartPruneSweeper(ctx context.Context) {
	pruneMu.Lock()
	defer pruneMu.Unlock()
	if pruneStarted {
		return
	}
	sweepCtx, cancel := context.WithCancel(ctx)
	pruneCancel = cancel
	pruneStarted = true
	worker.GoRestartable("notify.prune_sweeper", func() { s.runPruneSweeper(sweepCtx) })
}

// StopPruneSweeper 停止后台 prune（测试用）。
func StopPruneSweeper() {
	pruneMu.Lock()
	defer pruneMu.Unlock()
	if pruneCancel != nil {
		pruneCancel()
		pruneCancel = nil
	}
	pruneStarted = false
}

func (s *Service) runPruneSweeper(ctx context.Context) {
	ticker := time.NewTicker(notifyPruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pruneOnce(time.Now())
		}
	}
}

// pruneOnce 执行一轮删除；失败只记日志（下轮重试，尽力而为）。
// 独立超时 ctx（审查 M-4）：DB 卡顿不得挂住 sweeper goroutine（配合
// GoRestartable，panic 也不会永久停转）。
func (s *Service) pruneOnce(now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pruned, err := s.repo.PruneReadBefore(ctx, now.Add(-notifyPruneRetention))
	if err != nil {
		observability.L().Warn("notify.prune failed", "error", err)
		return
	}
	if pruned > 0 {
		observability.L().Info("notify.prune completed", "pruned", pruned)
	}
}

func (s *Service) SetNearbyNotifyProvider(p NearbyNotifySettingsProvider) { s.nearbyNotifyProvider = p }

type NearbyNotifySettingsProvider interface {
	IsNearbyMomentNotifyEnabled(ctx context.Context, userID string) (bool, error)
}

func (s *Service) List(ctx context.Context, req NotifyListRequest) (PaginatedNotify, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	query, err := normalizeNotifyListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	normalizedReq := NotifyListRequest{Type: query.Type, IsRead: query.IsRead, Pagination: &PaginationParams{Limit: query.Limit}}
	if query.Cursor != nil {
		normalizedReq.Pagination.Cursor = encodeNotifyCursor(*query.Cursor)
	}
	items, err := s.repo.List(ctx, principal.UserID, normalizedReq)
	if err != nil {
		observability.L().Error("notifys.list failed", "error", err)
		return PaginatedNotify{}, err
	}
	return items, nil
}

func (s *Service) UnreadCount(ctx context.Context) (UnreadSummaryResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.UnreadCount(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("notifys.unread_count failed", "error", err)
		return UnreadSummaryResult{}, err
	}
	return result, nil
}

func (s *Service) MarkAsRead(ctx context.Context, ids []string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAsRead(ctx, principal.UserID, ids); err != nil {
		observability.L().Error("notifys.mark_as_read failed", "error", err, "count", len(ids))
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("notifys.mark_all_read failed", "error", err)
		return err
	}
	return nil
}

// AppendNewFollower records a new_follower notify and returns the
// generated notify_id (embedded as the first field of the pushed
// NewFollower payload, spec api_spec_websocket.md).
func (s *Service) AppendNewFollower(ctx context.Context, targetUserID string, actor types.UserInfo, source string, sourceMomentID string) (string, error) {
	payload := map[string]any{"type": "new_follower", "actor": actor}
	if source != "" {
		payload["source"] = source
	}
	if sourceMomentID != "" {
		payload["source_moment_id"] = sourceMomentID
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "new_follower",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendMutualFollow records a mutual_follow notify and returns the
// generated notify_id (embedded as the first field of the pushed
// MutualFollow payload, spec api_spec_websocket.md).
func (s *Service) AppendMutualFollow(ctx context.Context, targetUserID string, actor types.UserInfo) (string, error) {
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "mutual_follow",
		IsRead:   false,
		Payload:  map[string]any{"type": "mutual_follow", "actor": actor},
	})
	return id, err
}

// AppendSystemAnnouncement records a system_announcement notify for each
// user and returns the generated notify_id. The same id is shared by all
// recipients and is embedded as the first field of the broadcast
// SystemAnnouncement payload (spec api_spec_websocket.md).
func (s *Service) AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) (string, error) {
	return s.AppendSystemAnnouncementWithID(ctx, newNotifyID(), userIDs, item)
}

// NewAnnouncementNotifyID 预生成一条系统公告的 notify_id。全量 fanout 会按
// 用户分批落库，但整条公告必须共用同一个 notify_id（契约：WS notify_id 与
// notify/fetch 主键一致，客户端据此去重与 mark_read）。
func (s *Service) NewAnnouncementNotifyID() string { return newNotifyID() }

// AppendSystemAnnouncementWithID 是 AppendSystemAnnouncement 的显式 id 版本。
// 全量 fanout 公告按 user_id keyset 分批落库（每批 500 人），但契约要求 WS 的
// notify_id 与 notify/fetch 的主键一致，且**整条公告**（而非某一批）对应同一
// 逻辑通知——因此调用方先 newNotifyID 一次，各批次复用同一个 id。
// 若每批各自生成 id，客户端拿到的 WS notify_id 只能命中最后一批，
// 其余批次的通知既无法按 id 去重也无法 mark_read（审计 F-06）。
func (s *Service) AppendSystemAnnouncementWithID(ctx context.Context, id string, userIDs []string, item systemmodule.SystemAnnouncementItem) (string, error) {
	if id == "" {
		id = newNotifyID()
	}
	err := s.repo.AppendMany(ctx, userIDs, NotifyItem{
		NotifyID: id,
		Type:     "system_announcement",
		IsRead:   false,
		Payload: map[string]any{
			"type":            "system_announcement",
			"announcement_id": item.AnnouncementID,
			"title":           item.Title, // spec notify.md：客户端直接渲染通知行
			"severity":        item.Severity,
		},
	})
	return id, err
}

// AppendGroupInviteReceived records a group_invite_received notify and
// returns the generated notify_id (embedded as the first field of the
// pushed GroupInviteReceived payload, spec api_spec_websocket.md).
func (s *Service) AppendGroupInviteReceived(ctx context.Context, targetUserID string, actor types.UserInfo, groupID string, groupName string, inviteID string, message *string) (string, error) {
	payload := map[string]any{
		"type":       "group_invite_received",
		"actor":      actor,
		"group_id":   groupID,
		"group_name": groupName,
		"invite_id":  inviteID,
	}
	if message != nil && *message != "" {
		payload["message"] = *message
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_invite_received", IsRead: false, Payload: payload})
	return id, err
}

// AppendGroupApplicationReceived records a group_application_received
// notify and returns the generated notify_id (embedded as the
// first field of the pushed GroupApplicationReceived payload, spec
// api_spec_websocket.md).
func (s *Service) AppendGroupApplicationReceived(ctx context.Context, targetUserID string, applicant types.UserInfo, groupID string, groupName string, applicationID string, message *string) (string, error) {
	payload := map[string]any{"type": "group_application_received", "applicant": applicant, "group_id": groupID, "group_name": groupName, "application_id": applicationID}
	if message != nil && *message != "" {
		payload["message"] = *message // spec notify.md：申请附言与 WS GroupApplicationReceivedPayload 对齐
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_application_received",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendGroupApplicationAccepted records a group_application_accepted
// notify and returns the generated notify_id (embedded as the
// first field of the pushed GroupApplicationResponse payload, spec
// api_spec_websocket.md).
func (s *Service) AppendGroupApplicationAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string) (string, error) {
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_application_accepted",
		IsRead:   false,
		Payload:  map[string]any{"type": "group_application_accepted", "group_id": groupID, "group_name": groupName, "application_id": applicationID},
	})
	return id, err
}

// AppendGroupApplicationRejected records a group_application_rejected
// notify. Returns the generated notify_id for the realtime payload.
// F7: 不暴露审批者身份（与 accepted 对称），notify payload 不含 rejected_by。
func (s *Service) AppendGroupApplicationRejected(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, reason *string) (string, error) {
	payload := map[string]any{
		"type":           "group_application_rejected",
		"group_id":       groupID,
		"group_name":     groupName,
		"application_id": applicationID,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_application_rejected", IsRead: false, Payload: payload})
	return id, err
}

// AppendReportResolved records a report_resolved notify (admin handled
// the report). Returns the generated notify_id for the realtime payload.
func (s *Service) AppendReportResolved(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error) {
	payload := map[string]any{
		"type":        "report_resolved",
		"report_id":   reportID,
		"target_type": targetType,
		"target_id":   targetID,
	}
	if note != "" {
		payload["note"] = note
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "report_resolved", IsRead: false, Payload: payload})
	return id, err
}

// AppendReportDismissed records a report_dismissed notify (admin
// dismissed the report). Returns the generated notify_id for the
// realtime payload.
func (s *Service) AppendReportDismissed(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error) {
	payload := map[string]any{
		"type":        "report_dismissed",
		"report_id":   reportID,
		"target_type": targetType,
		"target_id":   targetID,
	}
	if note != "" {
		payload["note"] = note
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "report_dismissed", IsRead: false, Payload: payload})
	return id, err
}

// AppendMomentCommented records a moment_commented notify and returns
// the generated notify_id (embedded as the first field of the pushed
// MomentCommented payload, spec api_spec_websocket.md).
func (s *Service) AppendMomentCommented(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (string, error) {
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "moment_commented",
		IsRead:   false,
		Payload: map[string]any{
			"type":            "moment_commented",
			"actor":           actor,
			"moment_id":       momentID,
			"moment_summary":  momentSummary,
			"comment_id":      commentID,
			"comment_preview": commentPreview,
		},
	})
	return id, err
}

// AppendReactionAdded records a generic reaction_added notify. Used for
// all three target kinds (moment/comment/message) — the legacy moment_liked /
// comment_liked types were removed from the contract (spec api_spec_types.md
// NotifyType). Returns the generated notify_id so callers can embed it
// in the realtime ReactionAdded payload.
func (s *Service) AppendReactionAdded(ctx context.Context, targetUserID string, actor types.UserInfo, targetType string, targetID string, emoji string) (string, error) {
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "reaction_added",
		IsRead:   false,
		Payload: map[string]any{
			"type":        "reaction_added",
			"actor":       actor,
			"target_type": targetType,
			"target_id":   targetID,
			"emoji":       emoji,
		},
	})
	return id, err
}

// AppendCommentReply records a comment_reply notify and returns the
// generated notify_id (embedded as the first field of the pushed
// CommentReply payload, spec api_spec_websocket.md).
func (s *Service) AppendCommentReply(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, parentCommentID string, parentPreview string, replyID string, replyPreview string) (string, error) {
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "comment_reply",
		IsRead:   false,
		Payload: map[string]any{
			"type":                    "comment_reply",
			"actor":                   actor,
			"moment_id":               momentID,
			"moment_summary":          momentSummary,
			"comment_id":              parentCommentID,
			"replied_comment_preview": parentPreview,
			"reply_id":                replyID,
			"reply_content_preview":   replyPreview,
		},
	})
	return id, err
}

// AppendMentioned records a mentioned notify and returns the generated
// notify_id (embedded as the first field of the pushed ContentMentioned
// payload, spec api_spec_websocket.md).
func (s *Service) AppendMentioned(ctx context.Context, targetUserID string, actor types.UserInfo, targetType string, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (string, error) {
	payload := map[string]any{
		"type":           "mentioned",
		"actor":          actor,
		"target_type":    targetType,
		"moment_id":      momentID,
		"moment_summary": momentSummary,
	}
	if commentID != "" {
		payload["comment_id"] = commentID
	}
	if commentPreview != "" {
		payload["comment_preview"] = commentPreview
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "mentioned",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendMentionedBatch 批量落库多个被提及用户的 mentioned 通知（单次往返，
// @ 提及热路径用）。每个用户独立 notify_id 且各自内嵌 payload，语义与逐人
// AppendMentioned 完全一致；落库失败整体报错（调用方按通知缺失降级推送），
// 返回 userID → notify_id 映射供调用方构造 WS 推送 payload。
func (s *Service) AppendMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, targetType string, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (map[string]string, error) {
	items := make([]NotifyItem, 0, len(targetUserIDs))
	ids := make([]string, 0, len(targetUserIDs))
	for range targetUserIDs {
		payload := map[string]any{
			"type":           "mentioned",
			"actor":          actor,
			"target_type":    targetType,
			"moment_id":      momentID,
			"moment_summary": momentSummary,
		}
		if commentID != "" {
			payload["comment_id"] = commentID
		}
		if commentPreview != "" {
			payload["comment_preview"] = commentPreview
		}
		id := newNotifyID()
		items = append(items, NotifyItem{NotifyID: id, Type: "mentioned", IsRead: false, Payload: payload})
		ids = append(ids, id)
	}
	if err := s.repo.AppendBatchPerUser(ctx, targetUserIDs, items); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(ids))
	for i, targetUserID := range targetUserIDs {
		out[targetUserID] = ids[i]
	}
	return out, nil
}

// AppendGroupMentioned records a group-chat mention notify
// (NotifyType='group_mentioned', spec api_spec_notify.md) and returns the
// notify_id to embed in the pushed GroupMessageMentioned payload.
func (s *Service) AppendGroupMentioned(ctx context.Context, targetUserID string, actor types.UserInfo, groupID, groupName, messageID, contentPreview string) (string, error) {
	payload := map[string]any{
		"type":       "group_mentioned",
		"actor":      actor,
		"group_id":   groupID,
		"group_name": groupName,
		"message_id": messageID,
	}
	if contentPreview != "" {
		payload["content_preview"] = contentPreview
	}
	id := newNotifyID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_mentioned",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendGroupMentionedBatch 批量落库群聊 @ 通知（单次往返）。与
// AppendMentionedBatch 同构：每用户独立 notify_id 且各自内嵌 payload，
// 失败整体报错，返回 userID → notify_id 映射。
func (s *Service) AppendGroupMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, groupID, groupName, messageID, contentPreview string) (map[string]string, error) {
	items := make([]NotifyItem, 0, len(targetUserIDs))
	ids := make([]string, 0, len(targetUserIDs))
	for range targetUserIDs {
		payload := map[string]any{
			"type":       "group_mentioned",
			"actor":      actor,
			"group_id":   groupID,
			"group_name": groupName,
			"message_id": messageID,
		}
		if contentPreview != "" {
			payload["content_preview"] = contentPreview
		}
		id := newNotifyID()
		items = append(items, NotifyItem{NotifyID: id, Type: "group_mentioned", IsRead: false, Payload: payload})
		ids = append(ids, id)
	}
	if err := s.repo.AppendBatchPerUser(ctx, targetUserIDs, items); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(ids))
	for i, targetUserID := range targetUserIDs {
		out[targetUserID] = ids[i]
	}
	return out, nil
}

// PushSystemAnnouncement broadcasts a system announcement to all online
// connections. The payload carries the same notify_id that
// AppendSystemAnnouncement persisted, so clients can deduplicate the realtime
// event against the HTTP notify/fetch record (spec api_spec_websocket.md).
func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem, notifyID string) {
	// 契约(api_spec_websocket.md → SystemAnnouncement)：payload 复用 SystemAnnouncementItem
	announcement := map[string]any{
		"announcement_id": item.AnnouncementID,
		"title":           item.Title,
		"content":         item.Content,
		"severity":        item.Severity,
		"created_at":      item.CreatedAt,
	}
	if item.ActionURL != nil && *item.ActionURL != "" {
		announcement["action_url"] = *item.ActionURL
	}
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: realtime.EventTypeSystemAnnouncement, Payload: map[string]any{"notify_id": notifyID, "announcement": announcement}})
}

func (s *Service) PublishNearbyNewMoment(ctx context.Context, targetUserID, actorID, momentID string, distance int, actorPayload types.UserInfo, momentSummaryPayload types.MomentSummary) error {
	if s.nearbyNotifyProvider != nil {
		enabled, err := s.nearbyNotifyProvider.IsNearbyMomentNotifyEnabled(ctx, targetUserID)
		if err != nil {
			observability.L().Error("notifys.publish_nearby_new_moment settings check failed", "error", err)
			return err
		}
		if !enabled {
			return nil
		}
	}
	payload := map[string]any{
		"type":           "nearby_new_moment",
		"actor":          actorPayload,
		"moment_id":      momentID,
		"moment_summary": momentSummaryPayload,
		"distance":       distance,
	}
	// notify_id 先生成后共用（对齐 AppendGroupMentioned 等模式）：落库记录与
	// NearbyNewMoment WS payload 携带同一 id，客户端可去重并对齐 HTTP 侧通知
	//（spec api_spec_websocket.md）。
	id := newNotifyID()
	payload["notify_id"] = id
	if err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "nearby_new_moment",
		IsRead:   false,
		Payload:  payload,
	}); err != nil {
		observability.L().Error("notifys.publish_nearby_new_moment append failed", "error", err)
		return err
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{
		Type:    realtime.EventTypeNearbyNewMoment,
		Payload: payload,
	})
	return nil
}
