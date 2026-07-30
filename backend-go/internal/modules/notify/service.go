package notify

import (
	"context"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	repo                 Repository
	nearbyNotifyProvider NearbyNotifySettingsProvider
}

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) SetNearbyNotifyProvider(p NearbyNotifySettingsProvider) { s.nearbyNotifyProvider = p }

type NearbyNotifySettingsProvider interface {
	IsNearbyMomentNotifyEnabled(ctx context.Context, userID string) (bool, error)
}

func (s *Service) List(ctx context.Context, req NotificationListRequest) (PaginatedNotify, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	normalizedReq := NotificationListRequest{Type: query.Type, Pagination: &PaginationParams{Limit: query.Limit}}
	if query.Cursor != nil {
		normalizedReq.Pagination.Cursor = encodeNotificationCursor(*query.Cursor)
	}
	items, err := s.repo.List(ctx, principal.UserID, normalizedReq)
	if err != nil {
		observability.L().Error("notifications.list failed", "error", err)
		return PaginatedNotify{}, err
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

func (s *Service) MarkAllReadByType(ctx context.Context, notificationType string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllReadByType(ctx, principal.UserID, notificationType); err != nil {
		observability.L().Error("notifications.mark_all_read_by_type failed", "error", err, "type", notificationType)
		return err
	}
	return nil
}

func (s *Service) AppendNewFollower(ctx context.Context, targetUserID string, actor map[string]any, source string, sourceMomentID string) error {
	payload := map[string]any{"type": "new_follower", "actor": actor}
	if source != "" {
		payload["source"] = source
	}
	if sourceMomentID != "" {
		payload["source_moment_id"] = sourceMomentID
	}
	return s.repo.Append(ctx, targetUserID, NotifyItem{
		Type:    "new_follower",
		IsRead:  false,
		Payload: payload,
	})
}

func (s *Service) AppendMutualFollow(ctx context.Context, targetUserID string, actor map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{
		Type:    "mutual_follow",
		IsRead:  false,
		Payload: map[string]any{"type": "mutual_follow", "actor": actor},
	})
}

func (s *Service) AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) error {
	return s.repo.AppendMany(ctx, userIDs, NotifyItem{
		Type:   "system_announcement",
		IsRead: false,
		Payload: map[string]any{
			"type":            "system_announcement",
			"announcement_id": item.AnnouncementID,
			"severity":        item.Severity,
		},
	})
}

func (s *Service) AppendGroupInviteReceived(ctx context.Context, targetUserID string, actor map[string]any, groupID string, groupName string, inviteID string, message *string) error {
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
	return s.repo.Append(ctx, targetUserID, NotifyItem{Type: "group_invite_received", IsRead: false, Payload: payload})
}

func (s *Service) AppendGroupInviteAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, user map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{Type: "group_invite_accepted", IsRead: false, Payload: map[string]any{"type": "group_invite_accepted", "group_id": groupID, "group_name": groupName, "invite_id": inviteID, "user": user}})
}

func (s *Service) AppendGroupApplicationReceived(ctx context.Context, targetUserID string, applicant map[string]any, groupID string, groupName string, applicationID string) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{Type: "group_application_received", IsRead: false, Payload: map[string]any{"type": "group_application_received", "applicant": applicant, "group_id": groupID, "group_name": groupName, "application_id": applicationID}})
}

func (s *Service) AppendGroupApplicationAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{Type: "group_application_accepted", IsRead: false, Payload: map[string]any{"type": "group_application_accepted", "group_id": groupID, "group_name": groupName, "application_id": applicationID}})
}

func (s *Service) AppendMomentLiked(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{
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
	return s.repo.Append(ctx, targetUserID, NotifyItem{
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
	return s.repo.Append(ctx, targetUserID, NotifyItem{
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

// AppendReactionAdded records a generic reaction_added notification. Used for
// message-target reactions (moment/comment reactions take the more specific
// moment_liked / comment_liked paths). Spec api_spec_types.md NotifyType.
func (s *Service) AppendReactionAdded(ctx context.Context, targetUserID string, actor map[string]any, targetType string, targetID string, emoji string) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{
		Type:   "reaction_added",
		IsRead: false,
		Payload: map[string]any{
			"type":        "reaction_added",
			"actor":       actor,
			"target_type": targetType,
			"target_id":   targetID,
			"emoji":       emoji,
		},
	})
}

func (s *Service) AppendCommentReply(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, parentCommentID string, parentPreview string, replyID string, replyPreview string) error {
	return s.repo.Append(ctx, targetUserID, NotifyItem{
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
	return s.repo.Append(ctx, targetUserID, NotifyItem{
		Type:    "mentioned",
		IsRead:  false,
		Payload: payload,
	})
}

func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem) {
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: realtime.EventTypeSystemAnnouncement, Payload: item})
}

func (s *Service) PublishNearbyNewMoment(ctx context.Context, targetUserID, actorID, momentID string, distance int, actorPayload map[string]any, momentSummaryPayload map[string]any) error {
	if s.nearbyNotifyProvider != nil {
		enabled, err := s.nearbyNotifyProvider.IsNearbyMomentNotifyEnabled(ctx, targetUserID)
		if err != nil {
			observability.L().Error("notifications.publish_nearby_new_moment settings check failed", "error", err)
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
	if err := s.repo.Append(ctx, targetUserID, NotifyItem{
		Type:    "nearby_new_moment",
		IsRead:  false,
		Payload: payload,
	}); err != nil {
		observability.L().Error("notifications.publish_nearby_new_moment append failed", "error", err)
		return err
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{
		Type:    realtime.EventTypeNearbyNewMoment,
		Payload: payload,
	})
	return nil
}
