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
	normalizedReq := NotificationListRequest{Type: query.Type, IsRead: query.IsRead, Pagination: &PaginationParams{Limit: query.Limit}}
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

// AppendNewFollower records a new_follower notification and returns the
// generated notify_id (embedded as the first field of the pushed
// NewFollower payload, spec api_spec_websocket.md).
func (s *Service) AppendNewFollower(ctx context.Context, targetUserID string, actor map[string]any, source string, sourceMomentID string) (string, error) {
	payload := map[string]any{"type": "new_follower", "actor": actor}
	if source != "" {
		payload["source"] = source
	}
	if sourceMomentID != "" {
		payload["source_moment_id"] = sourceMomentID
	}
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "new_follower",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendMutualFollow records a mutual_follow notification and returns the
// generated notify_id (embedded as the first field of the pushed
// MutualFollow payload, spec api_spec_websocket.md).
func (s *Service) AppendMutualFollow(ctx context.Context, targetUserID string, actor map[string]any) (string, error) {
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "mutual_follow",
		IsRead:   false,
		Payload:  map[string]any{"type": "mutual_follow", "actor": actor},
	})
	return id, err
}

// AppendSystemAnnouncement records a system_announcement notification for each
// user and returns the generated notify_id. The same id is shared by all
// recipients and is embedded as the first field of the broadcast
// SystemAnnouncement payload (spec api_spec_websocket.md).
func (s *Service) AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) (string, error) {
	id := newNotificationID()
	err := s.repo.AppendMany(ctx, userIDs, NotifyItem{
		NotifyID: id,
		Type:     "system_announcement",
		IsRead:   false,
		Payload: map[string]any{
			"type":            "system_announcement",
			"announcement_id": item.AnnouncementID,
			"severity":        item.Severity,
		},
	})
	return id, err
}

// AppendGroupInviteReceived records a group_invite_received notification and
// returns the generated notify_id (embedded as the first field of the
// pushed GroupInviteReceived payload, spec api_spec_websocket.md).
func (s *Service) AppendGroupInviteReceived(ctx context.Context, targetUserID string, actor map[string]any, groupID string, groupName string, inviteID string, message *string) (string, error) {
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
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_invite_received", IsRead: false, Payload: payload})
	return id, err
}

// AppendGroupInviteAccepted records a group_invite_accepted notification and
// returns the generated notify_id (embedded as the first field of the
// pushed GroupInviteResponse payload, spec api_spec_websocket.md).
func (s *Service) AppendGroupInviteAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, user map[string]any) (string, error) {
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_invite_accepted",
		IsRead:   false,
		Payload:  map[string]any{"type": "group_invite_accepted", "group_id": groupID, "group_name": groupName, "invite_id": inviteID, "user": user},
	})
	return id, err
}

// AppendGroupApplicationReceived records a group_application_received
// notification and returns the generated notify_id (embedded as the
// first field of the pushed GroupApplicationReceived payload, spec
// api_spec_websocket.md).
func (s *Service) AppendGroupApplicationReceived(ctx context.Context, targetUserID string, applicant map[string]any, groupID string, groupName string, applicationID string) (string, error) {
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_application_received",
		IsRead:   false,
		Payload:  map[string]any{"type": "group_application_received", "applicant": applicant, "group_id": groupID, "group_name": groupName, "application_id": applicationID},
	})
	return id, err
}

// AppendGroupApplicationAccepted records a group_application_accepted
// notification and returns the generated notify_id (embedded as the
// first field of the pushed GroupApplicationResponse payload, spec
// api_spec_websocket.md).
func (s *Service) AppendGroupApplicationAccepted(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string) (string, error) {
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_application_accepted",
		IsRead:   false,
		Payload:  map[string]any{"type": "group_application_accepted", "group_id": groupID, "group_name": groupName, "application_id": applicationID},
	})
	return id, err
}

// AppendGroupInviteRevoked records a group_invite_revoked notification
// (invite revoked by the inviter, or expired automatically). Returns the
// generated notify_id for the realtime payload.
func (s *Service) AppendGroupInviteRevoked(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, revokedBy map[string]any, reason *string) (string, error) {
	payload := map[string]any{
		"type":       "group_invite_revoked",
		"group_id":   groupID,
		"group_name": groupName,
		"invite_id":  inviteID,
	}
	if revokedBy != nil {
		payload["revoked_by"] = revokedBy
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_invite_revoked", IsRead: false, Payload: payload})
	return id, err
}

// AppendGroupInviteRejected records a group_invite_rejected notification
// (invite rejected by the invitee). Returns the generated notify_id for
// the realtime payload.
func (s *Service) AppendGroupInviteRejected(ctx context.Context, targetUserID string, groupID string, groupName string, inviteID string, rejectedBy map[string]any, reason *string) (string, error) {
	payload := map[string]any{
		"type":        "group_invite_rejected",
		"group_id":    groupID,
		"group_name":  groupName,
		"invite_id":   inviteID,
		"rejected_by": rejectedBy,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_invite_rejected", IsRead: false, Payload: payload})
	return id, err
}

// AppendGroupApplicationCancelled records a group_application_cancelled
// notification (application cancelled by the applicant, or expired
// automatically). Returns the generated notify_id for the realtime payload.
func (s *Service) AppendGroupApplicationCancelled(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, applicant map[string]any, reason *string) (string, error) {
	payload := map[string]any{
		"type":           "group_application_cancelled",
		"group_id":       groupID,
		"group_name":     groupName,
		"application_id": applicationID,
		"applicant":      applicant,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_application_cancelled", IsRead: false, Payload: payload})
	return id, err
}

// AppendGroupApplicationRejected records a group_application_rejected
// notification. Returns the generated notify_id for the realtime payload.
func (s *Service) AppendGroupApplicationRejected(ctx context.Context, targetUserID string, groupID string, groupName string, applicationID string, rejectedBy map[string]any, reason *string) (string, error) {
	payload := map[string]any{
		"type":           "group_application_rejected",
		"group_id":       groupID,
		"group_name":     groupName,
		"application_id": applicationID,
		"rejected_by":    rejectedBy,
	}
	if reason != nil && *reason != "" {
		payload["reason"] = *reason
	}
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "group_application_rejected", IsRead: false, Payload: payload})
	return id, err
}

// AppendReportResolved records a report_resolved notification (admin handled
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
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "report_resolved", IsRead: false, Payload: payload})
	return id, err
}

// AppendReportDismissed records a report_dismissed notification (admin
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
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{NotifyID: id, Type: "report_dismissed", IsRead: false, Payload: payload})
	return id, err
}

// AppendMomentCommented records a moment_commented notification and returns
// the generated notify_id (embedded as the first field of the pushed
// MomentCommented payload, spec api_spec_websocket.md).
func (s *Service) AppendMomentCommented(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, commentID string, commentPreview string) (string, error) {
	id := newNotificationID()
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

// AppendReactionAdded records a generic reaction_added notification. Used for
// all three target kinds (moment/comment/message) — the legacy moment_liked /
// comment_liked types were removed from the contract (spec api_spec_types.md
// NotifyType). Returns the generated notify_id so callers can embed it
// in the realtime ReactionAdded payload.
func (s *Service) AppendReactionAdded(ctx context.Context, targetUserID string, actor map[string]any, targetType string, targetID string, emoji string) (string, error) {
	id := newNotificationID()
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

// AppendCommentReply records a comment_reply notification and returns the
// generated notify_id (embedded as the first field of the pushed
// CommentReply payload, spec api_spec_websocket.md).
func (s *Service) AppendCommentReply(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, parentCommentID string, parentPreview string, replyID string, replyPreview string) (string, error) {
	id := newNotificationID()
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

// AppendMentioned records a mentioned notification and returns the generated
// notify_id (embedded as the first field of the pushed MessageMentioned
// payload, spec api_spec_websocket.md).
func (s *Service) AppendMentioned(ctx context.Context, targetUserID string, actor map[string]any, targetType string, momentID string, momentSummary map[string]any, commentID string, commentPreview string) (string, error) {
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
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "mentioned",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// AppendGroupMentioned records a group-chat mention notification
// (NotifyType='group_mentioned', spec api_spec_notify.md) and returns the
// notify_id to embed in the pushed GroupMessageMentioned payload.
func (s *Service) AppendGroupMentioned(ctx context.Context, targetUserID string, actor map[string]any, groupID, groupName, messageID, contentPreview string) (string, error) {
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
	id := newNotificationID()
	err := s.repo.Append(ctx, targetUserID, NotifyItem{
		NotifyID: id,
		Type:     "group_mentioned",
		IsRead:   false,
		Payload:  payload,
	})
	return id, err
}

// PushSystemAnnouncement broadcasts a system announcement to all online
// connections. The payload carries the same notify_id that
// AppendSystemAnnouncement persisted, so clients can deduplicate the realtime
// event against the HTTP notify/list record (spec api_spec_websocket.md).
func (s *Service) PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem, notificationID string) {
	payload := map[string]any{
		"notify_id":       notificationID,
		"announcement_id": item.AnnouncementID,
		"title":           item.Title,
		"content":         item.Content,
		"severity":        item.Severity,
		"created_at":      item.CreatedAt,
	}
	if item.ActionURL != "" {
		payload["action_url"] = item.ActionURL
	}
	realtime.DefaultHub().BroadcastToAll(realtime.Event{Type: realtime.EventTypeSystemAnnouncement, Payload: payload})
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
