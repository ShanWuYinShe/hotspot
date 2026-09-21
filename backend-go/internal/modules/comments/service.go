package comments

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"

	"golang.org/x/sync/errgroup"
)

var ErrMomentNotFound = errors.New("comments: moment not found")

// mentionsMaxCount 契约级上限（api_spec_base.md 字段校验约束：mentions 最多 20 个）。
// 与 moments/groupmessages 对齐：每个 mention 触发一次通知落库 + 一次 WS 推送，
// 不限制会让单条评论放大为数万条通知（轰炸向量）。
const mentionsMaxCount = 20

type Service struct {
	repo          Repository
	moments       MomentProvider
	notifications commentNotificationWriter
	users         actor.Provider
	appConfig     appconfig.Source
	runInTx       persistence.TxRunner // optional; nil => no transaction wrapping
	reactions     ReactionAggregator   // optional; nil => reactions[] not filled
	// blockChecks 可选注入：判定 caller 是否被目标瞬间作者拉黑（spec comment create →
	// 被拉黑 → 10200 + blocked_by_user）。生产由 users 侧实现注入；nil 表示不启用
	//（不产生误判，仅暴露 privacy_setting 路径）。返回 true 表示被拉黑。
	blockChecks func(callerID, authorID string) (bool, error)
}

// SetBlockChecker 注入拉黑判定（依赖倒置：comments 不直接 import users）。
func (s *Service) SetBlockChecker(fn func(callerID, authorID string) (bool, error)) {
	s.blockChecks = fn
}

// ReactionAggregator computes per-emoji reaction aggregates for comments.
// Implemented by the reactions module (dependency inversion). viewerID drives
// the @auth-only reacted_by_me flag.
type ReactionAggregator interface {
	AggregateComments(ctx context.Context, commentIDs []string, viewerID string) (map[string][]Reaction, error)
}

type commentNotificationWriter interface {
	AppendMomentCommented(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (string, error)
	AppendCommentReply(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, parentCommentID string, parentPreview string, replyID string, replyPreview string) (string, error)
	AppendMentionedBatch(ctx context.Context, targetUserIDs []string, actor types.UserInfo, targetType string, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (map[string]string, error)
}

func NewService(repo Repository, moments MomentProvider) *Service {
	return NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
}

func NewServiceWithDeps(repo Repository, moments MomentProvider, notificationWriter commentNotificationWriter, actorProvider actor.Provider, appConfig appconfig.Source, runInTx persistence.TxRunner) *Service {
	return &Service{repo: repo, moments: moments, notifications: notificationWriter, users: actorProvider, appConfig: appConfig, runInTx: runInTx}
}

// defaultCommentMaxRunes 与 ClientConfig.comment_max_length 默认值一致；
// 仅当 AppConfig 不可用或未配置时作为兜底。
const defaultCommentMaxRunes = 500

// ensureContentLength 按 ClientConfig.comment_max_length（rune 数）校验评论内容。
// 此前 Create 无任何长度校验，可写入接近 body 上限（1MB）的评论。
func (s *Service) ensureContentLength(ctx context.Context, content string) error {
	max := defaultCommentMaxRunes
	if s.appConfig != nil {
		if n := s.appConfig.Get(ctx).Limits.CommentMaxLength; n > 0 {
			max = n
		}
	}
	if utf8.RuneCountInString(content) > max {
		return ErrCommentTooLong
	}
	return nil
}

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) (CommentItem, error) {
	if err := s.ensureContentLength(ctx, req.Content); err != nil {
		return CommentItem{}, err
	}
	if len(req.Mentions) > mentionsMaxCount {
		return CommentItem{}, ErrTooManyMentions
	}
	if err := s.ensureMomentCommentable(ctx, req.MomentID); err != nil {
		return CommentItem{}, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	normalizedReq := req
	replyTarget, hasReplyTarget, err := s.resolveReplyTarget(ctx, req.ParentID)
	if err != nil {
		return CommentItem{}, err
	}
	// 回复必须指向同一瞬间下的评论：跨瞬间组合（moment A 请求体 + moment B
	// 的 comment_id）会污染 replies_count 并给被回复者发错误上下文的通知。
	// 归属不符按 ErrCommentNotFound 处理（不泄漏 B 瞬间评论的存在性）。
	if hasReplyTarget && replyTarget.MomentID != req.MomentID {
		return CommentItem{}, ErrCommentNotFound
	}
	if hasReplyTarget && replyTarget.ParentID != nil {
		normalizedReq.ParentID = *replyTarget.ParentID
	}

	// 评论插入与 moment 计数是跨模块两步写（无共享事务）：计数失败**不阻断**
	// 评论创建——评论已落库，若返回错误，客户端重试会产生重复评论。计数陈旧
	// 为已知限制（moment 侧读取时容忍、删除评论时递减修正）。
	createAndCount := func(ctx context.Context) (CommentItem, error) {
		item, err := s.repo.Create(ctx, principal.UserID, normalizedReq)
		if err != nil {
			observability.L().Error("comments.create repo create failed", "error", err)
			return CommentItem{}, err
		}
		if err := s.moments.IncrementCommentCount(ctx, normalizedReq.MomentID); err != nil {
			observability.L().Warn("comments.create increment comment count failed", "error", err, "moment_id", normalizedReq.MomentID, "comment_id", item.CommentID)
		}
		return item, nil
	}

	var item CommentItem
	if s.runInTx != nil {
		err = s.runInTx(ctx, func(txCtx context.Context) error {
			var txErr error
			item, txErr = createAndCount(txCtx)
			return txErr
		})
		if err != nil {
			return CommentItem{}, err
		}
	} else {
		item, err = createAndCount(ctx)
		if err != nil {
			return CommentItem{}, err
		}
	}

	// Notifications are side-effects: they run after commit and must not roll
	// the comment back if they fail.
	momentContext, ok, err := s.moments.NotificationContext(ctx, normalizedReq.MomentID)
	if err != nil {
		observability.L().Error("comments.create load moment notification context failed", "error", err, "moment_id", normalizedReq.MomentID, "comment_id", item.CommentID)
		return item, nil
	}
	if !ok {
		return item, nil
	}
	actor := buildCommentActor(item, principal.UserID)
	mentionTargets := uniqueMentionTargets(normalizedReq.Mentions, principal.UserID)
	if hasReplyTarget {
		s.notifyReplyTarget(ctx, momentContext, actor, replyTarget, item, mentionTargets)
	} else {
		s.notifyMomentAuthor(ctx, momentContext, actor, item, mentionTargets)
	}
	s.notifyMentionedUsers(ctx, momentContext, actor, item, mentionTargets)
	return item, nil
}

func (s *Service) List(ctx context.Context, req ListCommentsRequest) (PaginatedComments, error) {
	// 可见性校验（软删/private 瞬间按不存在处理）：此前仅查存在性，知道
	// moment_id 的任意登录用户可读 private 瞬间的全部评论内容。
	if err := s.ensureMomentVisible(ctx, req.MomentID); err != nil {
		return PaginatedComments{}, err
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
		}
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.List(ctx, req.MomentID, limit, cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	s.fillTopReplies(ctx, result.Items)
	s.fillReactionsBatch(ctx, result.Items, viewerUserID(ctx))
	return result, nil
}

func (s *Service) fillTopReplies(ctx context.Context, items []CommentItem) {
	topLimit := 3
	if s.appConfig != nil {
		if n := s.appConfig.Get(ctx).ResponseDefaults.CommentTopRepliesCount; n > 0 {
			topLimit = n
		}
	}
	// Collect parent IDs that have replies, then fetch their top replies in a
	// single batched query (avoids an N+1 of ListReplies per top-level comment).
	parentIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.RepliesCount > 0 {
			parentIDs = append(parentIDs, item.CommentID)
		}
	}
	if len(parentIDs) == 0 {
		return
	}
	repliesByParent, err := s.repo.ListTopRepliesByParentIDs(ctx, parentIDs, topLimit)
	if err != nil {
		observability.L().Error("comments.fill_top_replies batch failed", "error", err)
		return
	}
	for i := range items {
		if items[i].RepliesCount > 0 {
			items[i].TopReplies = repliesByParent[items[i].CommentID]
		}
	}
}

// Get returns a single comment by id. found=false (no error) when the comment
// does not exist. Exposed for cross-module lookups (e.g. reactions target
// resolution). Does NOT fill reactions[].
func (s *Service) Get(ctx context.Context, commentID string) (CommentItem, bool, error) {
	return s.repo.Get(ctx, commentID)
}

// GetForDetail returns a single comment with reactions[] filled (backs
// comments/open_the_door). Use Get for internal lookups that don't need reactions.
func (s *Service) GetForDetail(ctx context.Context, commentID string) (CommentItem, bool, error) {
	item, found, err := s.repo.Get(ctx, commentID)
	if err != nil || !found {
		return CommentItem{}, found, err
	}
	// 评论详情同样受瞬间可见性约束：private 瞬间的评论对非拥有者不可见
	//（按不存在处理，不泄漏存在性）。
	if err := s.ensureMomentVisible(ctx, item.MomentID); err != nil {
		return CommentItem{}, false, nil
	}
	s.fillReactionsBatch(ctx, []CommentItem{item}, viewerUserID(ctx))
	return item, true, nil
}

// ensureMomentVisible 校验瞬间对当前 viewer 可见；不可见/不存在统一返回
// ErrMomentNotFound（存在性不泄漏）。
func (s *Service) ensureMomentVisible(ctx context.Context, momentID string) error {
	found, visible, err := s.moments.IsReactionTargetVisible(ctx, momentID)
	if err != nil {
		return err
	}
	if !found || !visible {
		return ErrMomentNotFound
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, req DeleteCommentRequest) error {
	comment, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// 评论删除仅作者（spec api_spec_comment.md「删除评论（仅评论作者）」）。
	// 平台审核删除统一走 /admin/api/v1/reports/judge，本端点没有「平台管理员」分支。
	isOwner := comment.User.UserID == principal.UserID
	if !isOwner {
		return ErrPermissionDenied
	}
	// Already soft-deleted — no-op (idempotent)
	if comment.IsDeleted {
		return nil
	}
	// 软删与计数递减同一事务（数据治理审计：两步此前独立执行且递减失败仅
	// Warn，moments.comment_count 漂移不可自愈；创建路径已同款事务）。
	// 递减在此成为事务内失败即整体回滚，不再吞错。
	deleteAll := func(txCtx context.Context) error {
		if err := s.repo.SoftDelete(txCtx, req.CommentID); err != nil {
			return err
		}
		return s.moments.DecrementCommentCount(txCtx, comment.MomentID)
	}
	if s.runInTx != nil {
		return s.runInTx(ctx, deleteAll)
	}
	return deleteAll(ctx)
}

func (s *Service) ListReplies(ctx context.Context, req CommentRepliesRequest) (PaginatedComments, error) {
	comment, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return PaginatedComments{}, err
	}
	if !found {
		return PaginatedComments{}, ErrCommentNotFound
	}
	// 回复列表与评论详情同受瞬间可见性约束：private/软删瞬间的回复对
	// 非拥有者按评论不存在处理（存在性不泄漏），与 List/GetForDetail 一致。
	if err := s.ensureMomentVisible(ctx, comment.MomentID); err != nil {
		if errors.Is(err, ErrMomentNotFound) {
			return PaginatedComments{}, ErrCommentNotFound
		}
		return PaginatedComments{}, err
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
		}
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.ListReplies(ctx, req.CommentID, limit, cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	s.fillReactionsBatch(ctx, result.Items, viewerUserID(ctx))
	return result, nil
}

// SetReactionAggregator wires the reactions aggregator (called by bootstrap).
func (s *Service) SetReactionAggregator(ra ReactionAggregator) { s.reactions = ra }

// fillReactionsBatch populates CommentItem.reactions[] for a list in one query,
// including any nested TopReplies.
func (s *Service) fillReactionsBatch(ctx context.Context, items []CommentItem, viewerID string) {
	if s.reactions == nil || len(items) == 0 {
		return
	}
	// Collect all comment ids across top-level items and their top replies so a
	// single aggregate query covers the whole tree.
	idSet := make(map[string]struct{}, len(items)*2)
	collect := func(items []CommentItem) {
		for i := range items {
			idSet[items[i].CommentID] = struct{}{}
			for _, r := range items[i].TopReplies {
				idSet[r.CommentID] = struct{}{}
			}
		}
	}
	collect(items)
	if len(idSet) == 0 {
		return
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	byID, err := s.reactions.AggregateComments(ctx, ids, viewerID)
	if err != nil {
		observability.L().Warn("comments.fillReactionsBatch failed", "error", err)
		return
	}
	apply := func(items []CommentItem) {
		for i := range items {
			if recs := byID[items[i].CommentID]; len(recs) > 0 {
				items[i].Reactions = recs
			}
			for j := range items[i].TopReplies {
				if recs := byID[items[i].TopReplies[j].CommentID]; len(recs) > 0 {
					items[i].TopReplies[j].Reactions = recs
				}
			}
		}
	}
	apply(items)
}

// viewerUserID returns the authenticated user id, or "" for anonymous callers.
func viewerUserID(ctx context.Context) string {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return ""
	}
	return principal.UserID
}

func (s *Service) resolveReplyTarget(ctx context.Context, parentID string) (CommentItem, bool, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return CommentItem{}, false, nil
	}
	parent, found, err := s.repo.Get(ctx, parentID)
	if err != nil {
		return CommentItem{}, false, err
	}
	// 已软删评论按不存在处理：不允许对已删除的评论继续回复（与 Delete 的
	// 幂等软删语义一致，回复目标消失即视为评论不存在）。
	if !found || parent.IsDeleted {
		return CommentItem{}, false, ErrCommentNotFound
	}
	return parent, true, nil
}

func (s *Service) notifyMomentAuthor(ctx context.Context, momentContext MomentNotificationContext, actor types.UserInfo, item CommentItem, mentionTargets map[string]struct{}) {
	targetUserID := strings.TrimSpace(momentContext.UserID)
	if targetUserID == "" || targetUserID == item.User.UserID {
		return
	}
	delete(mentionTargets, targetUserID)
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	commentPreview := summarizeCommentText(item.Content, "")
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendMomentCommented(ctx, targetUserID, actor, item.MomentID, momentSummary, item.CommentID, commentPreview)
		if err != nil {
			observability.L().Error("comments.create append moment_commented notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
		} else {
			notificationID = id
		}
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMomentCommented, Payload: MomentCommentedPayload{
		NotifyID:      notificationID,
		MomentID:      item.MomentID,
		MomentSummary: momentSummary,
		CommentID:     item.CommentID,
		Actor:         actor,
		Content:       item.Content,
		CommentedAt:   item.CreatedAt,
	}})
}

func (s *Service) notifyReplyTarget(ctx context.Context, momentContext MomentNotificationContext, actor types.UserInfo, replyTarget CommentItem, item CommentItem, mentionTargets map[string]struct{}) {
	targetUserID := strings.TrimSpace(replyTarget.User.UserID)
	if targetUserID == "" || targetUserID == item.User.UserID {
		return
	}
	delete(mentionTargets, targetUserID)
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	parentPreview := summarizeCommentText(replyTarget.Content, "")
	replyPreview := summarizeCommentText(item.Content, "")
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendCommentReply(ctx, targetUserID, actor, item.MomentID, momentSummary, replyTarget.CommentID, parentPreview, item.CommentID, replyPreview)
		if err != nil {
			observability.L().Error("comments.create append comment_reply notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
		} else {
			notificationID = id
		}
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeCommentReply, Payload: CommentReplyPayload{
		NotifyID:              notificationID,
		MomentID:              item.MomentID,
		MomentSummary:         momentSummary,
		CommentID:             replyTarget.CommentID,
		RepliedCommentPreview: parentPreview,
		ReplyID:               item.CommentID,
		Actor:                 actor,
		Content:               item.Content,
		RepliedAt:             item.CreatedAt,
	}})
}

// notifyMentionedUsers 对被提及用户批量落库 mentioned 通知（单次往返）并以
// 有界并发推送 ContentMentioned 事件。
func (s *Service) notifyMentionedUsers(ctx context.Context, momentContext MomentNotificationContext, actor types.UserInfo, item CommentItem, mentionTargets map[string]struct{}) {
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	commentPreview := summarizeCommentText(item.Content, "")
	targets := sortedMentionTargets(mentionTargets)
	if len(targets) == 0 {
		return
	}
	// 批量落库（单次往返）：失败时记日志并按通知缺失降级（notify_id 置空），
	// 广播仍照常进行——与此前逐人失败只影响该人的口径一致，只是失败粒度为整批。
	var notifyIDs map[string]string
	var appendErr error
	if s.notifications != nil {
		notifyIDs, appendErr = s.notifications.AppendMentionedBatch(ctx, targets, actor, "comment", item.MomentID, momentSummary, item.CommentID, commentPreview)
		if appendErr != nil {
			observability.L().Error("comments.create batch append mentioned notifications failed", "error", appendErr, "moment_id", item.MomentID, "comment_id", item.CommentID, "count", len(targets))
		}
	}
	// 并行推送（有界并发）：每条 BroadcastToUser 内部含 resume Record + 跨实例
	// publish 两次网络往返，逐条串行在多 @ 场景放大发送延迟。
	broadcast := func(targetUserID, notificationID string) func() error {
		return func() error {
			realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeContentMentioned, Payload: ContentMentionedPayload{
				NotifyID:       notificationID,
				TargetType:     "comment",
				MomentID:       item.MomentID,
				CommentID:      item.CommentID,
				MomentSummary:  momentSummary,
				CommentPreview: commentPreview,
				Actor:          actor,
				MentionedAt:    item.CreatedAt,
			}})
			return nil
		}
	}
	group, _ := errgroup.WithContext(context.Background())
	group.SetLimit(8)
	for _, targetUserID := range targets {
		var notificationID string
		if notifyIDs != nil {
			notificationID = notifyIDs[targetUserID]
		}
		group.Go(broadcast(targetUserID, notificationID))
	}
	_ = group.Wait()
}

func buildCommentActor(item CommentItem, fallbackUserID string) types.UserInfo {
	userID := strings.TrimSpace(item.User.UserID)
	if userID == "" {
		userID = strings.TrimSpace(fallbackUserID)
	}
	nickname := strings.TrimSpace(item.User.Nickname)
	if nickname == "" {
		nickname = userID
	}
	avatarURL := ""
	if item.User.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*item.User.AvatarURL)
	}
	return types.UserInfo{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)}
}

func notificationMomentSummary(momentContext MomentNotificationContext, momentID string) types.MomentSummary {
	summary := momentContext.Summary
	if summary.MomentID == "" {
		summary.MomentID = momentID
	}
	return summary
}

func uniqueMentionTargets(mentions []string, selfUserID string) map[string]struct{} {
	targets := make(map[string]struct{}, len(mentions))
	for _, mention := range mentions {
		userID := strings.TrimSpace(mention)
		if userID == "" || userID == selfUserID {
			continue
		}
		targets[userID] = struct{}{}
	}
	return targets
}

func sortedMentionTargets(targets map[string]struct{}) []string {
	userIDs := make([]string, 0, len(targets))
	for userID := range targets {
		userIDs = append(userIDs, userID)
	}
	sort.Strings(userIDs)
	return userIDs
}

func summarizeCommentText(primary string, fallback string) string {
	value := strings.TrimSpace(primary)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	// 按 rune 截断：字节切片会把中文等多字节字符拦腰截断产生非法 UTF-8。
	return textutil.TruncateRunes(value, 80)
}

func (s *Service) ensureMomentCommentable(ctx context.Context, momentID string) error {
	found, canComment, err := s.moments.CheckCommentable(ctx, momentID)
	if err != nil {
		observability.L().Error("comments.ensure_moment_commentable failed", "error", err, "moment_id", momentID)
		return err
	}
	if !found {
		// 瞬间不存在或不可见（private 非拥有者）→ 10300
		return ErrMomentNotFound
	}
	if !canComment {
		// 瞬间作者限制评论（spec comment create → 10200 + privacy_setting）
		return ErrCommentPrivacyDenied
	}
	if s.blockChecks != nil {
		caller := ""
		author := ""
		if p, ok := platformauth.PrincipalFromContext(ctx); ok {
			caller = p.UserID
		}
		if mc, ok, err := s.moments.NotificationContext(ctx, momentID); err == nil && ok {
			author = mc.UserID
		}
		if author != "" && caller != "" {
			blocked, err := s.blockChecks(caller, author)
			if err != nil {
				// 黑名单查询失败时 fail-closed：返回错误（统一 10600）而非放行，
				// 避免 DB 抖动期间被拉黑用户可继续评论。
				observability.L().Error("comments.block_check failed", "error", err, "author_id", author)
				return err
			}
			if blocked {
				// 被瞬间作者拉黑（spec comment create → 10200 + blocked_by_user）
				return ErrCommentBlockedByUser
			}
		}
	}
	return nil
}

func (s *Service) ensureMomentExists(ctx context.Context, momentID string) error {
	exists, err := s.moments.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("comments.ensure_moment_exists failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return nil
}
