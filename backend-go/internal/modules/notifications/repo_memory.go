package notifications

import (
	"context"
	"sort"
	"sync"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]NotificationItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[string][]NotificationItem)}
}

func (r *InMemoryRepository) Append(_ context.Context, userID string, item NotificationItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[userID] = append(r.items[userID], normalizeNotificationItem(item))
	return nil
}

func (r *InMemoryRepository) AppendMany(ctx context.Context, userIDs []string, item NotificationItem) error {
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

func (r *InMemoryRepository) List(_ context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotifications{}, err
	}
	items := make([]NotificationItem, 0)
	for _, item := range r.items[userID] {
		if query.Type == "" || item.Type == query.Type {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].NotificationID > items[j].NotificationID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if query.Cursor != nil {
		filtered := items[:0]
		for _, item := range items {
			if item.CreatedAt < query.Cursor.CreatedAt || (item.CreatedAt == query.Cursor.CreatedAt && item.NotificationID < query.Cursor.NotificationID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	page := PaginatedNotifications{}
	if len(items) > query.Limit {
		last := items[query.Limit-1]
		page.NextCursor = encodeNotificationCursor(notificationCursor{CreatedAt: last.CreatedAt, NotificationID: last.NotificationID})
		items = items[:query.Limit]
	}
	page.Items = items
	return page, nil
}

func (r *InMemoryRepository) UnreadCount(_ context.Context, userID string) (UnreadCountResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := UnreadCountResult{ByType: map[string]int{"social": 0, "interaction": 0, "group": 0, "system": 0}}
	for _, item := range r.items[userID] {
		if item.IsRead {
			continue
		}
		result.Total++
		result.ByType[bucket(item.Type)]++
	}
	return result, nil
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items[userID] {
		for _, id := range ids {
			if r.items[userID][i].NotificationID == id {
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

func bucket(notificationType string) string {
	switch notificationType {
	case "new_follower", "mutual_follow":
		return "social"
	case "moment_liked", "moment_commented", "comment_liked", "comment_reply", "mentioned":
		return "interaction"
	case "group_invite_received", "group_invite_accepted", "group_application_received", "group_application_accepted":
		return "group"
	default:
		return "system"
	}
}
