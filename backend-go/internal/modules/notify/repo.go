package notify

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/idgen"
)

type Repository interface {
	Append(context.Context, string, NotifyItem) error
	AppendMany(context.Context, []string, NotifyItem) error
	List(context.Context, string, NotifyListRequest) (PaginatedNotify, error)
	UnreadCount(context.Context, string) (UnreadSummaryResult, error)
	MarkAsRead(context.Context, string, []string) error
	MarkAllRead(context.Context, string) error
}

func normalizeNotifyItem(item NotifyItem) NotifyItem {
	now := time.Now().UTC()
	if item.NotifyID == "" {
		item.NotifyID = newNotifyID()
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

// newNotifyID generates the notify primary key. Callers that need
// the id for realtime event payloads (e.g. ReactionAdded.notify_id) generate
// it up front and pass it into the NotifyItem.
func newNotifyID() string {
	// idgen 随机后缀：notif_<UnixNano> 在多 goroutine/多实例同纳秒生成会
	// 主键冲突，通知插入失败且只被记日志（静默丢失）。
	return idgen.MustWithPrefix("notif_")
}
