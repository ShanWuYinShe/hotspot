package notifications

import "context"

type Repository interface {
	List(context.Context, string, NotificationListRequest) (PaginatedNotifications, error)
	UnreadCount(context.Context, string) (UnreadCountResult, error)
	MarkAsRead(context.Context, string, []string) error
	MarkAllRead(context.Context, string) error
}
