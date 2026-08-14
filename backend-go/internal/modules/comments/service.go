package comments

import (
	"context"
	"errors"
	"sort"
	"strings"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

var ErrMomentNotFound = errors.New("comments: moment not found")

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
	blockChecks func(callerID, authorID string) bool
}

// SetBlockChecker 注入拉黑判定（依赖倒置：comments 不直接 import users）。
func (s *Service) SetBlockChecker(fn func(callerID, authorID string) bool) { s.blockChecks = fn }

// ReactionAggregator computes per-emoji reaction aggregates for comments.
// Implemented by the reactions module (dependency inversion). viewerID drives
// the @auth-only reacted_by_me flag.
type ReactionAggregator interface {
	AggregateComments(ctx context.Context, commentIDs []string, viewerID string) (map[string][]Reaction, error)
}

type commentNotificationWriter interface {
	AppendMomentCommented(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (string, error)
	AppendCommentReply(ctx context.Context, targetUserID string, actor types.UserInfo, momentID string, momentSummary types.MomentSummary, parentCommentID string, parentPreview string, replyID string, replyPreview string) (string, error)
	AppendMentioned(ctx context.Context, targetUserID string, actor types.UserInfo, targetType string, momentID string, momentSummary types.MomentSummary, commentID string, commentPreview string) (string, error)
}

func NewService(repo Repository, moments MomentProvider) *Service {
	return NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
}

func NewServiceWithDeps(repo Repository, moments MomentProvider, notificationWriter commentNotificationWriter, actorProvider actor.Provider, appConfig appconfig.Source, runInTx persistence.TxRunner) *Service {
	return &Service{repo: repo, moments: moments, notifications: notificationWriter, users: actorProvider, appConfig: appConfig, runInTx: runInTx}
}

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) (CommentItem, error) {
	if err := s.ensureMomentCommentable(ctx, req.MomentID); err != nil {
		return CommentItem{}, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	normalizedReq := req
	replyTarget, hasReplyTarget, err := s.resolveReplyTarget(ctx, req.ParentID)
	if err != nil {
		return CommentItem{}, err
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
	if err := s.ensureMomentExists(ctx, req.MomentID); err != nil {
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
	s.fillReactionsBatch(ctx, []CommentItem{item}, viewerUserID(ctx))
	return item, true, nil
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
	// 平台审核删除统一走 /admin/v1/reports/judge，本端点没有「平台管理员」分支。
	isOwner := comment.User.UserID == principal.UserID
	if !isOwner {
		return ErrPermissionDenied
	}
	// Already soft-deleted — no-op (idempotent)
	if comment.IsDeleted {
		return nil
	}
	if err := s.repo.SoftDelete(ctx, req.CommentID); err != nil {
		return err
	}
	if err := s.moments.DecrementCommentCount(ctx, comment.MomentID); err != nil {
		observability.L().Warn("comments.delete decrement count failed", "error", err, "moment_id", comment.MomentID)
	}
	return nil
}

func (s *Service) ListReplies(ctx context.Context, req CommentRepliesRequest) (PaginatedComments, error) {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return PaginatedComments{}, err
	}
	if !found {
		return PaginatedComments{}, ErrCommentNotFound
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
	if !found {
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
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMomentCommented, Payload: map[string]any{
		"notify_id":      notificationID,
		"moment_id":      item.MomentID,
		"moment_summary": momentSummary,
		"comment_id":     item.CommentID,
		"actor":          actor,
		"content":        item.Content,
		"commented_at":   item.CreatedAt,
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
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeCommentReply, Payload: map[string]any{
		"notify_id":               notificationID,
		"moment_id":               item.MomentID,
		"moment_summary":          momentSummary,
		"comment_id":              replyTarget.CommentID,
		"replied_comment_preview": parentPreview,
		"reply_id":                item.CommentID,
		"actor":                   actor,
		"content":                 item.Content,
		"replied_at":              item.CreatedAt,
	}})
}

func (s *Service) notifyMentionedUsers(ctx context.Context, momentContext MomentNotificationContext, actor types.UserInfo, item CommentItem, mentionTargets map[string]struct{}) {
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	commentPreview := summarizeCommentText(item.Content, "")
	for _, targetUserID := range sortedMentionTargets(mentionTargets) {
		notificationID := ""
		if s.notifications != nil {
			id, err := s.notifications.AppendMentioned(ctx, targetUserID, actor, "comment", item.MomentID, momentSummary, item.CommentID, commentPreview)
			if err != nil {
				observability.L().Error("comments.create append mentioned notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
			} else {
				notificationID = id
			}
		}
		realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeContentMentioned, Payload: map[string]any{
			"notify_id":       notificationID,
			"target_type":     "comment",
			"moment_id":       item.MomentID,
			"comment_id":      item.CommentID,
			"moment_summary":  momentSummary,
			"comment_preview": commentPreview,
			"actor":           actor,
			"mentioned_at":    item.CreatedAt,
		}})
	}
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
	return types.UserInfo{UserID: userID, Nickname: nickname, AvatarURL: strings.TrimSpace(item.User.AvatarURL)}
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
	if len(value) > 80 {
		return value[:80]
	}
	return value
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
		if author != "" && caller != "" && s.blockChecks(caller, author) {
			// 被瞬间作者拉黑（spec comment create → 10200 + blocked_by_user）
			return ErrCommentBlockedByUser
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
