package notifications

import (
	"context"
	"fmt"
	"time"
)

type Repository interface {
	Append(context.Context, string, NotificationItem) error
	AppendMany(context.Context, []string, NotificationItem) error
	List(context.Context, string, NotificationListRequest) (PaginatedNotifications, error)
	UnreadCount(context.Context, string) (UnreadCountResult, error)
	MarkAsRead(context.Context, string, []string) error
	MarkAllRead(context.Context, string) error
}

func normalizeNotificationItem(item NotificationItem) NotificationItem {
	now := time.Now().UTC()
	if item.NotificationID == "" {
		item.NotificationID = fmt.Sprintf("notif_%d", now.UnixNano())
	}
	if item.CreatedAt == "" {
		item.CreatedAt = now.Format(time.RFC3339)
	}
	if item.Payload == nil {
		item.Payload = map[string]any{}
	}
	item.Payload["type"] = item.Type
	return item
}
