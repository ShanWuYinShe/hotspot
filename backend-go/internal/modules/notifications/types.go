package notifications

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

type NotificationItem struct {
	NotificationID string         `json:"notification_id"`
	Type           string         `json:"type"`
	IsRead         bool           `json:"is_read"`
	CreatedAt      string         `json:"created_at"`
	Payload        map[string]any `json:"payload"`
}

type PaginatedNotifications struct {
	Items      []NotificationItem `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type UnreadCountByType struct {
	Social      int `json:"social"`
	Interaction int `json:"interaction"`
	Group       int `json:"group"`
	System      int `json:"system"`
}

type UnreadCountResult struct {
	Total  int               `json:"total"`
	ByType UnreadCountByType `json:"by_type"`
}

type PaginationParams = types.PaginationParams

type NotificationListRequest struct {
	Type       string            `json:"type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type MarkAsReadRequest struct {
	NotificationIDs []string `json:"notification_ids"`
}

type notificationListQuery struct {
	Type   string
	Limit  int
	Cursor *notificationCursor
}

type notificationCursor struct {
	CreatedAt      string `json:"created_at"`
	NotificationID string `json:"notification_id"`
}

func normalizeNotificationType(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "", "new_follower", "mutual_follow", "moment_liked", "moment_commented", "comment_liked", "comment_reply", "mentioned", "group_invite_received", "group_invite_accepted", "group_application_received", "group_application_accepted", "system_announcement":
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
	return notificationListQuery{Type: normalizedType, Limit: limit, Cursor: cursor}, nil
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
	if cursor.CreatedAt == "" || cursor.NotificationID == "" {
		return nil, ErrInvalidNotificationRequest
	}
	return &cursor, nil
}
