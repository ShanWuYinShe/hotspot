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
}

func normalizeNotifyItem(item NotifyItem) NotifyItem {
	now := time.Now().UTC()
	if item.NotifyID == "" {
		item.NotifyID = newNotificationID()
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

// newNotificationID generates the notification primary key. Callers that need
// the id for realtime event payloads (e.g. ReactionAdded.notify_id) generate
// it up front and pass it into the NotifyItem.
func newNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UTC().UnixNano())
}
