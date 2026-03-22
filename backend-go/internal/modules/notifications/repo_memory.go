package notifications

import (
	"context"
	"sync"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]NotificationItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[string][]NotificationItem)}
}

func (r *InMemoryRepository) List(_ context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]NotificationItem, 0)
	for _, item := range r.items[userID] {
		if req.Type == "" || item.Type == req.Type {
			items = append(items, item)
		}
	}
	return PaginatedNotifications{Items: items}, nil
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
