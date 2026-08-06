package notify

import (
	"errors"
	"strings"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/types"
)

const (
	defaultNotificationListLimit = 20
	maxNotificationListLimit     = 100
)

var ErrInvalidNotificationRequest = errors.New("notifications: invalid request")

type NotifyCategory = string

const (
	NotifyCategorySocial      NotifyCategory = "social"
	NotifyCategoryInteraction NotifyCategory = "interaction"
	NotifyCategoryGroup       NotifyCategory = "group"
	NotifyCategorySystem      NotifyCategory = "system"
	NotifyCategoryNearby      NotifyCategory = "nearby"
)

// NotifyItem is the notify/list item. The primary key is exposed as notify_id
// (spec api_spec_notify.md): it is shared with the realtime WS event payload of
// the same business event, so clients can deduplicate across the two paths.
type NotifyItem struct {
	NotifyID  string         `json:"notify_id"`
	Type      string         `json:"type"`
	IsRead    bool           `json:"is_read"`
	CreatedAt string         `json:"created_at"`
	Payload   map[string]any `json:"payload"`
}

type PaginatedNotify struct {
	Items      []NotifyItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type UnreadCountByType struct {
	Social      int `json:"social"`
	Interaction int `json:"interaction"`
	Group       int `json:"group"`
	Report      int `json:"report"`
	Nearby      int `json:"nearby"`
	System      int `json:"system"`
}

type UnreadCountResult struct {
	Total  int               `json:"total"`
	ByType UnreadCountByType `json:"by_type"`
}

type PaginationParams = types.PaginationParams

type NotificationListRequest struct {
	Type       string            `json:"type,omitempty"`
	IsRead     *bool             `json:"is_read,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

// MarkReadRequest marks notifications as read (spec api_spec_notify.md):
// notify_ids selects the notifications to mark (1-50); absent or empty means
// "mark everything read". Unknown or foreign ids are silently ignored.
type MarkReadRequest struct {
	NotifyIDs []string `json:"notify_ids,omitempty"`
}

type notificationListQuery struct {
	Type   string
	IsRead *bool
	Limit  int
	Cursor *notificationCursor
}

type notificationCursor struct {
	CreatedAt string `json:"created_at"`
	NotifyID  string `json:"notify_id"`
}

func normalizeNotificationType(value string) string {
	value = strings.TrimSpace(value)
	// 与 spec/api_spec_types.md NotifyType union 保持一致（15 种）。
	switch value {
	case "", "new_follower", "mutual_follow", "moment_commented", "comment_reply", "mentioned", "group_mentioned", "reaction_added", "group_invite_received", "group_application_received", "group_application_accepted", "group_application_rejected", "report_resolved", "report_dismissed", "system_announcement", "nearby_new_moment":
		return value
	default:
		return ""
	}
}

func normalizeNotificationListQuery(req NotificationListRequest) (notificationListQuery, error) {
	normalizedType := normalizeNotificationType(req.Type)
	if strings.TrimSpace(req.Type) != "" && normalizedType == "" {
		return notificationListQuery{}, ErrInvalidNotificationRequest
	}
	limit := defaultNotificationListLimit
	cursorValue := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > maxNotificationListLimit {
			limit = maxNotificationListLimit
		}
		cursorValue = strings.TrimSpace(req.Pagination.Cursor)
	}
	cursor, err := decodeNotificationCursor(cursorValue)
	if err != nil {
		return notificationListQuery{}, err
	}
	return notificationListQuery{Type: normalizedType, IsRead: req.IsRead, Limit: limit, Cursor: cursor}, nil
}

// notificationCursorNamespace 是通知列表分页游标的加密命名空间。
const notificationCursorNamespace = "notify.list"

func encodeNotificationCursor(c notificationCursor) string {
	value, err := cursor.Encode(notificationCursorNamespace, c)
	if err != nil {
		return ""
	}
	return value
}

func decodeNotificationCursor(value string) (*notificationCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var c notificationCursor
	if err := cursor.Decode(notificationCursorNamespace, value, &c); err != nil {
		return nil, ErrInvalidNotificationRequest
	}
	if c.CreatedAt == "" || c.NotifyID == "" {
		return nil, ErrInvalidNotificationRequest
	}
	return &c, nil
}
