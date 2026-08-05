package notify

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

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

type NearbyNewMomentPayload struct {
	Actor         NearbyUserInfo `json:"actor"`
	MomentID      string         `json:"moment_id"`
	MomentSummary MomentSummary  `json:"moment_summary"`
	Distance      int            `json:"distance"`
}

type NearbyUserInfo struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

type MomentSummary struct {
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
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
	switch value {
	case "", "new_follower", "mutual_follow", "moment_commented", "comment_reply", "mentioned", "reaction_added", "group_invite_received", "group_invite_revoked", "group_invite_rejected", "group_invite_accepted", "group_application_received", "group_application_cancelled", "group_application_accepted", "group_application_rejected", "report_resolved", "report_dismissed", "system_announcement", "nearby_new_moment":
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

func encodeNotificationCursor(cursor notificationCursor) string {
	payload, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeNotificationCursor(value string) (*notificationCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidNotificationRequest
	}
	var cursor notificationCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, ErrInvalidNotificationRequest
	}
	if cursor.CreatedAt == "" || cursor.NotifyID == "" {
		return nil, ErrInvalidNotificationRequest
	}
	return &cursor, nil
}
