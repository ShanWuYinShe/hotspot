package notify

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/idgen"
)

type Repository interface {
	Append(context.Context, string, NotifyItem) error
	AppendMany(context.Context, []string, NotifyItem) error
	// AppendBatchPerUser 批量落库多用户**各自独立**的通知（@ 提及场景）。
	// userIDs 与 items 按位对应、长度必须相等；与 AppendMany（全收件人共用
	// 同一 notify_id）语义不同，勿混用。
	AppendBatchPerUser(context.Context, []string, []NotifyItem) error
	List(context.Context, string, NotifyListRequest) (PaginatedNotify, error)
	UnreadCount(context.Context, string) (UnreadSummaryResult, error)
	MarkAsRead(context.Context, string, []string) error
	MarkAllRead(context.Context, string) error
	// PruneReadBefore 物理删除「已读且 created_at 早于 before」的通知，返回
	// 删除行数。供后台 prune sweeper 调用（防无限增长）。注意表无 updated_at
	// 列，以 created_at 近似已读年龄——实际已读时间不早于 created_at，该口径
	// 只会少删（保守）。
	PruneReadBefore(ctx context.Context, before time.Time) (int64, error)
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
