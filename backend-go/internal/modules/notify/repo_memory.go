package notify

import (
	"context"
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

func (r *InMemoryRepository) List(_ context.Context, userID string, req NotificationListRequest) (PaginatedNotify, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	query, err := normalizeNotificationListQuery(req)
	if err != nil {
		return PaginatedNotify{}, err
	}
	items := make([]NotifyItem, 0)
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
	page := PaginatedNotify{}
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
	result := UnreadCountResult{ByType: UnreadCountByType{}}
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

func (r *InMemoryRepository) MarkAllReadByType(_ context.Context, userID string, notificationType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items[userID] {
		if r.items[userID][i].Type == notificationType {
			r.items[userID][i].IsRead = true
		}
	}
	return nil
}

func bucket(notificationType string) string {
	switch notificationType {
	case "new_follower", "mutual_follow", "nearby_new_moment":
		return "social"
	case "moment_liked", "moment_commented", "comment_liked", "comment_reply", "mentioned", "reaction_added":
		return "interaction"
	case "group_invite_received", "group_invite_accepted", "group_application_received", "group_application_accepted":
		return "group"
	default:
		return "system"
	}
}
