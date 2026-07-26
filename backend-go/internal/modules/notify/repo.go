package notify

import (
	"context"
	"fmt"
	"time"
)

type Repository interface {
	Append(context.Context, string, NotifyItem) error
	AppendMany(context.Context, []string, NotifyItem) error
	List(context.Context, string, NotificationListRequest) (PaginatedNotify, error)
	UnreadCount(context.Context, string) (UnreadCountResult, error)
	MarkAsRead(context.Context, string, []string) error
	MarkAllRead(context.Context, string) error
	MarkAllReadByType(context.Context, string, string) error
}

func normalizeNotifyItem(item NotifyItem) NotifyItem {
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
