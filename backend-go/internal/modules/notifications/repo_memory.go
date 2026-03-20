package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	items map[string][]NotificationItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{items: make(map[string][]NotificationItem)}
}

func (r *InMemoryRepository) seed(userID string) {
	if len(r.items[userID]) > 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.items[userID] = []NotificationItem{{NotificationID: fmt.Sprintf("notif_%s_1", userID), Type: "system_announcement", IsRead: false, CreatedAt: now, Payload: map[string]any{"type": "system_announcement", "announcement_id": "ann_1", "severity": "info"}}}
}

func (r *InMemoryRepository) List(_ context.Context, userID string, req NotificationListRequest) PaginatedNotifications {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seed(userID)
	items := make([]NotificationItem, 0)
	for _, item := range r.items[userID] {
		if req.Type == "" || item.Type == req.Type {
			items = append(items, item)
		}
	}
	return PaginatedNotifications{Items: items}
}

func (r *InMemoryRepository) UnreadCount(_ context.Context, userID string) UnreadCountResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seed(userID)
	result := UnreadCountResult{ByType: map[string]int{"social": 0, "interaction": 0, "group": 0, "system": 0}}
	for _, item := range r.items[userID] {
		if item.IsRead {
			continue
		}
		result.Total++
		result.ByType[bucket(item.Type)]++
	}
	return result
}

func (r *InMemoryRepository) MarkAsRead(_ context.Context, userID string, ids []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seed(userID)
	for i := range r.items[userID] {
		for _, id := range ids {
			if r.items[userID][i].NotificationID == id {
				r.items[userID][i].IsRead = true
			}
		}
	}
}

func (r *InMemoryRepository) MarkAllRead(_ context.Context, userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seed(userID)
	for i := range r.items[userID] {
		r.items[userID][i].IsRead = true
	}
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
