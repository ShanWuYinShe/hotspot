package notifications

import (
	"context"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, req NotificationListRequest) (PaginatedNotifications, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	normalizedReq := NotificationListRequest{Type: query.Type, Pagination: &PaginationParams{Limit: query.Limit}}
	if query.Cursor != nil {
		normalizedReq.Pagination.Cursor = encodeNotificationCursor(*query.Cursor)
	}
	items, err := s.repo.List(ctx, principal.UserID, normalizedReq)
	if err != nil {
		observability.L().Error("notifications.list failed", "error", err)
		return PaginatedNotifications{}, err
	}
	return items, nil
}

func (s *Service) UnreadCount(ctx context.Context) (UnreadCountResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	result, err := s.repo.UnreadCount(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("notifications.unread_count failed", "error", err)
		return UnreadCountResult{}, err
	}
	return result, nil
}

func (s *Service) MarkAsRead(ctx context.Context, ids []string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAsRead(ctx, principal.UserID, ids); err != nil {
		observability.L().Error("notifications.mark_as_read failed", "error", err, "count", len(ids))
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("notifications.mark_all_read failed", "error", err)
		return err
	}
	return nil
}

func (s *Service) AppendNewFollower(ctx context.Context, targetUserID string, actor map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:    "new_follower",
		IsRead:  false,
		Payload: map[string]any{"type": "new_follower", "actor": actor},
	})
}

func (s *Service) AppendMutualFollow(ctx context.Context, targetUserID string, actor map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:    "mutual_follow",
		IsRead:  false,
		Payload: map[string]any{"type": "mutual_follow", "actor": actor},
	})
}

func (s *Service) AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) error {
	return s.repo.AppendMany(ctx, userIDs, NotificationItem{
		Type:   "system_announcement",
		IsRead: false,
		Payload: map[string]any{
			"type":            "system_announcement",
			"announcement_id": item.AnnouncementID,
			"severity":        item.Severity,
		},
	})
}

func (s *Service) AppendMomentLiked(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:   "moment_liked",
		IsRead: false,
		Payload: map[string]any{
			"type":           "moment_liked",
			"actor":          actor,
			"moment_id":      momentID,
			"moment_summary": momentSummary,
		},
	})
}

func (s *Service) AppendMomentCommented(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:   "moment_commented",
		IsRead: false,
		Payload: map[string]any{
			"type":            "moment_commented",
			"actor":           actor,
			"moment_id":       momentID,
			"moment_summary":  momentSummary,
			"comment_id":      commentID,
			"comment_preview": commentPreview,
		},
	})
}

func (s *Service) AppendCommentLiked(ctx context.Context, targetUserID string, actor map[string]any, commentID string, commentPreview string, momentID string, momentSummary map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:   "comment_liked",
		IsRead: false,
		Payload: map[string]any{
			"type":            "comment_liked",
			"actor":           actor,
			"comment_id":      commentID,
			"comment_preview": commentPreview,
			"moment_id":       momentID,
			"moment_summary":  momentSummary,
		},
	})
}

func (s *Service) AppendCommentReply(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, parentCommentID string, parentPreview string, replyID string, replyPreview string) error {
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:   "comment_reply",
		IsRead: false,
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
}

func (s *Service) AppendMentioned(ctx context.Context, targetUserID string, actor map[string]any, targetType string, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error {
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
	return s.repo.Append(ctx, targetUserID, NotificationItem{
		Type:    "mentioned",
		IsRead:  false,
		Payload: payload,
	})
}

func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem) {
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: "SystemAnnouncement", Payload: item})
}
