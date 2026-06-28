package comments

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/notifications"
	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

var ErrMomentNotFound = errors.New("comments: moment not found")

type Service struct {
	repo          Repository
	moments       MomentProvider
	notifications commentNotificationWriter
	users         actor.Provider
}

type commentNotificationWriter interface {
	AppendMomentCommented(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error
	AppendCommentLiked(ctx context.Context, targetUserID string, actor map[string]any, commentID string, commentPreview string, momentID string, momentSummary map[string]any) error
	AppendCommentReply(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, parentCommentID string, parentPreview string, replyID string, replyPreview string) error
	AppendMentioned(ctx context.Context, targetUserID string, actor map[string]any, targetType string, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error
}

func NewService(repo Repository, moments MomentProvider) *Service {
	return NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
}

func NewServiceWithDeps(repo Repository, moments MomentProvider, notificationWriter commentNotificationWriter, actorProvider actor.Provider) *Service {
	return &Service{repo: repo, moments: moments, notifications: notificationWriter, users: actorProvider}
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
	if hasReplyTarget && replyTarget.ParentID != "" {
		normalizedReq.ParentID = replyTarget.ParentID
	}
	item, err := s.repo.Create(ctx, principal.UserID, normalizedReq)
	if err != nil {
		observability.L().Error("comments.create repo create failed", "error", err)
		return CommentItem{}, err
	}
	if err := s.moments.IncrementCommentCount(ctx, normalizedReq.MomentID); err != nil {
		observability.L().Error("comments.create increment comment count failed", "error", err, "moment_id", normalizedReq.MomentID, "comment_id", item.CommentID)
		if rollbackErr := s.repo.SoftDelete(ctx, item.CommentID); rollbackErr != nil {
			observability.L().Error("comments.create rollback soft delete failed", "error", rollbackErr, "comment_id", item.CommentID)
			return CommentItem{}, fmt.Errorf("increment comment count: %w; rollback soft delete comment: %v", err, rollbackErr)
		}
		return CommentItem{}, err
	}
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
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	return s.repo.List(ctx, req.MomentID, limit, cursor)
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
	isOwner := comment.User.UserID == principal.UserID
	isAdmin := principal.IsAdmin()
	if !isOwner && !isAdmin {
		return ErrPermissionDenied
	}
	// Already soft-deleted — no-op (idempotent)
	if comment.IsDeleted {
		return nil
	}
	if err := s.repo.SoftDelete(ctx, req.CommentID); err != nil {
		return err
	}
	_ = s.moments.DecrementCommentCount(ctx, comment.MomentID)
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
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListReplies(ctx, req.CommentID, limit, cursor)
}

func (s *Service) Like(ctx context.Context, req LikeCommentRequest) error {
	comment, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	newlyLiked, err := s.repo.Like(ctx, req.CommentID, principal.UserID)
	if err != nil {
		return err
	}
	if !newlyLiked || strings.TrimSpace(comment.User.UserID) == "" || comment.User.UserID == principal.UserID {
		return nil
	}
	momentContext, ok, err := s.moments.NotificationContext(ctx, comment.MomentID)
	if err != nil {
		observability.L().Error("comments.like load moment notification context failed", "error", err, "moment_id", comment.MomentID, "comment_id", comment.CommentID)
		return nil
	}
	if !ok {
		return nil
	}
	actor, actorErr := s.buildActorForUser(ctx, principal.UserID)
	if actorErr != nil {
		observability.L().Error("comments.like build actor failed", "error", actorErr, "user_id", principal.UserID)
		actor = buildCommentActor(CommentItem{User: UserInfo{UserID: principal.UserID, Nickname: principal.UserID}}, principal.UserID)
	}
	momentSummary := notificationMomentSummary(momentContext, comment.MomentID)
	commentPreview := summarizeCommentText(comment.Content, "")
	if s.notifications != nil {
		if err := s.notifications.AppendCommentLiked(ctx, comment.User.UserID, actor, comment.CommentID, commentPreview, comment.MomentID, momentSummary); err != nil {
			observability.L().Error("comments.like append comment_liked notification failed", "error", err, "moment_id", comment.MomentID, "comment_id", comment.CommentID, "target_user_id", comment.User.UserID)
		}
	}
	realtime.DefaultHub().BroadcastToUser(comment.User.UserID, realtime.Event{Type: realtime.EventTypeCommentLiked, Payload: map[string]any{
		"comment_id":      comment.CommentID,
		"moment_id":       comment.MomentID,
		"moment_summary":  cloneSummary(momentSummary),
		"comment_preview": commentPreview,
		"actor":           actor,
		"liked_at":        currentTimestamp(),
	}})
	return nil
}

func (s *Service) Unlike(ctx context.Context, req UnlikeCommentRequest) error {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unlike(ctx, req.CommentID, principal.UserID)
}

func (s *Service) ListLikes(ctx context.Context, req CommentLikesRequest) (PaginatedCommentLikes, error) {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return PaginatedCommentLikes{}, err
	}
	if !found {
		return PaginatedCommentLikes{}, ErrCommentNotFound
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	items, nextCursor, err := s.repo.ListLikes(ctx, req.CommentID, limit, cursor)
	if err != nil {
		return PaginatedCommentLikes{}, err
	}
	return PaginatedCommentLikes{Items: items, NextCursor: nextCursor}, nil
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

func (s *Service) notifyMomentAuthor(ctx context.Context, momentContext MomentNotificationContext, actor map[string]any, item CommentItem, mentionTargets map[string]struct{}) {
	targetUserID := strings.TrimSpace(momentContext.UserID)
	if targetUserID == "" || targetUserID == item.User.UserID {
		return
	}
	delete(mentionTargets, targetUserID)
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	commentPreview := summarizeCommentText(item.Content, "")
	if s.notifications != nil {
		if err := s.notifications.AppendMomentCommented(ctx, targetUserID, actor, item.MomentID, momentSummary, item.CommentID, commentPreview); err != nil {
			observability.L().Error("comments.create append moment_commented notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
		}
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMomentCommented, Payload: map[string]any{
		"moment_id":      item.MomentID,
		"moment_summary": cloneSummary(momentSummary),
		"comment_id":     item.CommentID,
		"actor":          actor,
		"content":        item.Content,
		"commented_at":   item.CreatedAt,
	}})
}

func (s *Service) notifyReplyTarget(ctx context.Context, momentContext MomentNotificationContext, actor map[string]any, replyTarget CommentItem, item CommentItem, mentionTargets map[string]struct{}) {
	targetUserID := strings.TrimSpace(replyTarget.User.UserID)
	if targetUserID == "" || targetUserID == item.User.UserID {
		return
	}
	delete(mentionTargets, targetUserID)
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	parentPreview := summarizeCommentText(replyTarget.Content, "")
	replyPreview := summarizeCommentText(item.Content, "")
	if s.notifications != nil {
		if err := s.notifications.AppendCommentReply(ctx, targetUserID, actor, item.MomentID, momentSummary, replyTarget.CommentID, parentPreview, item.CommentID, replyPreview); err != nil {
			observability.L().Error("comments.create append comment_reply notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
		}
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeCommentReply, Payload: map[string]any{
		"moment_id":               item.MomentID,
		"moment_summary":          cloneSummary(momentSummary),
		"comment_id":              replyTarget.CommentID,
		"replied_comment_preview": parentPreview,
		"reply_id":                item.CommentID,
		"actor":                   actor,
		"content":                 item.Content,
		"replied_at":              item.CreatedAt,
	}})
}

func (s *Service) notifyMentionedUsers(ctx context.Context, momentContext MomentNotificationContext, actor map[string]any, item CommentItem, mentionTargets map[string]struct{}) {
	momentSummary := notificationMomentSummary(momentContext, item.MomentID)
	commentPreview := summarizeCommentText(item.Content, "")
	for _, targetUserID := range sortedMentionTargets(mentionTargets) {
		if s.notifications != nil {
			if err := s.notifications.AppendMentioned(ctx, targetUserID, actor, "comment", item.MomentID, momentSummary, item.CommentID, commentPreview); err != nil {
				observability.L().Error("comments.create append mentioned notification failed", "error", err, "moment_id", item.MomentID, "comment_id", item.CommentID, "target_user_id", targetUserID)
			}
		}
		realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: realtime.EventTypeMessageMentioned, Payload: map[string]any{
			"target_type":     "comment",
			"moment_id":       item.MomentID,
			"comment_id":      item.CommentID,
			"moment_summary":  cloneSummary(momentSummary),
			"comment_preview": commentPreview,
			"actor":           actor,
			"mentioned_at":    item.CreatedAt,
		}})
	}
}

func buildCommentActor(item CommentItem, fallbackUserID string) map[string]any {
	userID := strings.TrimSpace(item.User.UserID)
	if userID == "" {
		userID = strings.TrimSpace(fallbackUserID)
	}
	nickname := strings.TrimSpace(item.User.Nickname)
	if nickname == "" {
		nickname = userID
	}
	actor := map[string]any{"user_id": userID, "nickname": nickname}
	if avatarURL := strings.TrimSpace(item.User.AvatarURL); avatarURL != "" {
		actor["avatar_url"] = avatarURL
	}
	return actor
}

func notificationMomentSummary(momentContext MomentNotificationContext, momentID string) map[string]any {
	summary := cloneSummary(momentContext.Summary)
	if len(summary) == 0 {
		summary["moment_id"] = momentID
	}
	return summary
}

func cloneSummary(summary map[string]any) map[string]any {
	cloned := make(map[string]any, len(summary))
	for key, value := range summary {
		cloned[key] = value
	}
	return cloned
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

func currentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
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
		return ErrMomentNotFound
	}
	if !canComment {
		return ErrMomentNoPermission
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

func (s *Service) buildActorForUser(ctx context.Context, userID string) (map[string]any, error) {
	if s.users == nil {
		return nil, nil
	}
	info, err := s.users.GetActorInfo(ctx, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"user_id": info.UserID, "nickname": info.Nickname, "avatar_url": info.AvatarURL}, nil
}
