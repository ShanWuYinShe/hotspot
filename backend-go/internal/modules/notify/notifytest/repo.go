// Package notifytest 提供 notify.Repository 的测试专用内存实现。
//
// 架构审计 P0-2 后生产构建不再包含内存仓储；但 comments/moments/admin/groups/
// users 等跨包测试需要同一套「Append 后按用户 List」语义。集中在此，避免每个
// 消费者复制一份容易漂移的假实现；本包只被 *_test.go 引用，不进入服务端二进制。
package notifytest

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/platform/idgen"
)

// Repository 是 notify.Repository 的测试实现。
type Repository struct {
	mu    sync.RWMutex
	items map[string][]notify.NotifyItem
}

// New 返回空的通知测试仓储。
func New() *Repository {
	return &Repository{items: make(map[string][]notify.NotifyItem)}
}

func normalizeItem(item notify.NotifyItem) notify.NotifyItem {
	if item.NotifyID == "" {
		item.NotifyID = idgen.MustWithPrefix("notif_")
	}
	if item.CreatedAt == "" {
		item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if item.Payload == nil {
		item.Payload = map[string]any{}
	}
	item.Payload["type"] = item.Type
	return item
}

// Append 落库单用户通知。
func (r *Repository) Append(_ context.Context, userID string, item notify.NotifyItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[userID] = append(r.items[userID], normalizeItem(item))
	return nil
}

// AppendMany 给多个用户落库同一条通知。
func (r *Repository) AppendMany(ctx context.Context, userIDs []string, item notify.NotifyItem) error {
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if err := r.Append(ctx, userID, item); err != nil {
			return err
		}
	}
	return nil
}

// AppendBatchPerUser 按位落库多用户各自独立的通知。
func (r *Repository) AppendBatchPerUser(ctx context.Context, userIDs []string, items []notify.NotifyItem) error {
	if len(userIDs) != len(items) {
		return errors.New("notifytest.AppendBatchPerUser: userIDs/items 长度不一致")
	}
	for i, userID := range userIDs {
		if userID == "" {
			continue
		}
		if err := r.Append(ctx, userID, items[i]); err != nil {
			return err
		}
	}
	return nil
}

// List 按用户返回通知（type/is_read 过滤 + created_at DESC 排序 + limit 截断）。
// 测试用例只用空请求或纯 limit 分页；游标语义不在测试桩范围（真实游标由
// notify 包单测与 PG IT 覆盖）。
func (r *Repository) List(_ context.Context, userID string, req notify.NotifyListRequest) (notify.PaginatedNotify, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	items := make([]notify.NotifyItem, 0)
	for _, item := range r.items[userID] {
		if req.Type != "" && item.Type != req.Type {
			continue
		}
		if req.IsRead != nil && item.IsRead != *req.IsRead {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].NotifyID > items[j].NotifyID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return notify.PaginatedNotify{Items: items}, nil
}

// UnreadCount 返回未读汇总（分类口径与 notify 内存实现一致）。
func (r *Repository) UnreadCount(_ context.Context, userID string) (notify.UnreadSummaryResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := notify.UnreadSummaryResult{ByType: notify.UnreadCountByType{}}
	for _, item := range r.items[userID] {
		if item.IsRead {
			continue
		}
		result.Total++
		switch notifyCategory(item.Type) {
		case "social":
			result.ByType.Social++
		case "interaction":
			result.ByType.Interaction++
		case "group":
			result.ByType.Group++
		case "report":
			result.ByType.Report++
		case "nearby":
			result.ByType.Nearby++
		case "system":
			result.ByType.System++
		}
	}
	return result, nil
}

// MarkAsRead 把指定通知标记已读（外部/不存在 id 静默忽略）。
func (r *Repository) MarkAsRead(_ context.Context, userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for i := range r.items[userID] {
		if _, ok := wanted[r.items[userID][i].NotifyID]; ok {
			r.items[userID][i].IsRead = true
		}
	}
	return nil
}

// MarkAllRead 把用户全部通知标记已读。
func (r *Repository) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items[userID] {
		r.items[userID][i].IsRead = true
	}
	return nil
}

// PruneReadBefore 物理删除已读且创建早于 before 的通知。
func (r *Repository) PruneReadBefore(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var pruned int64
	for userID, items := range r.items {
		kept := items[:0]
		for _, item := range items {
			createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
			if err == nil && item.IsRead && createdAt.Before(before) {
				pruned++
				continue
			}
			kept = append(kept, item)
		}
		if len(kept) == 0 {
			delete(r.items, userID)
			continue
		}
		r.items[userID] = kept
	}
	return pruned, nil
}

// notifyCategory 映射通知类型到分类桶（与 notify 生产实现同口径）。
func notifyCategory(notifyType string) string {
	switch notifyType {
	case "new_follower", "mutual_follow":
		return "social"
	case "moment_commented", "comment_reply", "mentioned", "reaction_added":
		return "interaction"
	case "group_invite_received", "group_application_received", "group_application_accepted", "group_application_rejected":
		return "group"
	case "report_resolved", "report_dismissed":
		return "report"
	case "nearby_new_moment":
		return "nearby"
	default:
		return "system"
	}
}
