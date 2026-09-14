package notify

import (
	"context"
	"errors"
	"sort"
	"sync"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]NotifyItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[string][]NotifyItem)}
}

func (r *InMemoryRepository) Append(_ context.Context, userID string, item NotifyItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[userID] = append(r.items[userID], normalizeNotifyItem(item))
	return nil
}

func (r *InMemoryRepository) AppendMany(ctx context.Context, userIDs []string, item NotifyItem) error {
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

// AppendBatchPerUser 逐条落库（内存实现无批量收益），userIDs 与 items 按位
// 对应、长度必须相等，语义与 Postgres 实现一致。
func (r *InMemoryRepository) AppendBatchPerUser(ctx context.Context, userIDs []string, items []NotifyItem) error {
	if len(userIDs) != len(items) {
		return errors.New("notify.AppendBatchPerUser: userIDs/items 长度不一致")
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

func (r *InMemoryRepository) List(_ context.Context, userID string, req NotifyListRequest) (PaginatedNotify, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query, err := normalizeNotifyListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	items := make([]NotifyItem, 0)
	for _, item := range r.items[userID] {
		if query.Type != "" && item.Type != query.Type {
			continue
		}
		if query.IsRead != nil && item.IsRead != *query.IsRead {
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
	if query.Cursor != nil {
		filtered := items[:0]
		for _, item := range items {
			if item.CreatedAt < query.Cursor.CreatedAt || (item.CreatedAt == query.Cursor.CreatedAt && item.NotifyID < query.Cursor.NotifyID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	page := PaginatedNotify{}
	if len(items) > query.Limit {
		last := items[query.Limit-1]
		c := encodeNotifyCursor(notifyCursor{CreatedAt: last.CreatedAt, NotifyID: last.NotifyID})
		page.NextCursor = &c
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

func (r *InMemoryRepository) UnreadCount(_ context.Context, userID string) (UnreadSummaryResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := UnreadSummaryResult{ByType: UnreadCountByType{}}
	for _, item := range r.items[userID] {
		if item.IsRead {
			continue
		}
		result.Total++
		switch bucket(item.Type) {
		case "social":
			result.ByType.Social++
		case "interaction":
			result.ByType.Interaction++
		case "group":
			result.ByType.Group++
		case "system":
			result.ByType.System++
		}
	}
	return result, nil
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items[userID] {
		for _, id := range ids {
			if r.items[userID][i].NotifyID == id {
				r.items[userID][i].IsRead = true
			}
		}
	}
	return nil
}

func (r *InMemoryRepository) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items[userID] {
		r.items[userID][i].IsRead = true
	}
	return nil
}

func bucket(notifyType string) string {
	switch notifyType {
	case "new_follower", "mutual_follow":
		return "social"
	case "moment_commented", "comment_reply", "mentioned", "reaction_added":
		return "interaction"
	case "group_invite_received", "group_invite_revoked", "group_invite_rejected", "group_invite_accepted", "group_application_received", "group_application_cancelled", "group_application_accepted", "group_application_rejected":
		return "group"
	case "report_resolved", "report_dismissed":
		return "report"
	case "nearby_new_moment":
		return "nearby"
	default:
		return "system"
	}
}
