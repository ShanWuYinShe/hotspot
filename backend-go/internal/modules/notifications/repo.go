package notifications

import "context"

type Repository interface {
	List(context.Context, string, NotificationListRequest) PaginatedNotifications
	UnreadCount(context.Context, string) UnreadCountResult
	MarkAsRead(context.Context, string, []string)
	MarkAllRead(context.Context, string)
}
