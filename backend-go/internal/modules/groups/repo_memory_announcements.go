package groups

import (
	"context"
	"fmt"

	"time"
)

func (r *InMemoryRepository) CreateAnnouncement(_ context.Context, creatorID string, req CreateGroupAnnouncementRequest) (GroupAnnouncementItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.seq++
	announcementID := fmt.Sprintf("announce_%d_%d", now.UnixNano(), r.seq)
	isPinned := false
	if req.IsPinned != nil {
		isPinned = *req.IsPinned
	}
	item := GroupAnnouncementItem{
		AnnouncementID: announcementID,
		GroupID:        req.GroupID,
		Title:          req.Title,
		Content:        req.Content,
		CreatedBy:      UserInfo{UserID: creatorID, Nickname: creatorID},
		IsPinned:       isPinned,
		CreatedAt:      now.Format(time.RFC3339),
	}
	r.announcements[announcementID] = item
	return item, nil
}

func (r *InMemoryRepository) ListAnnouncements(_ context.Context, groupID string) ([]GroupAnnouncementItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.groups[groupID]; !ok {
		return nil, ErrGroupNotFound
	}
	items := make([]GroupAnnouncementItem, 0)
	for _, item := range r.announcements {
		if item.GroupID == groupID {
			items = append(items, item)
		}
	}
	return items, nil
}

// GetAnnouncement 按归属双键直查（repo.go 接口；审查 P2-2，与 PG 实现对齐：
// 跨群归属按不存在处理，不泄漏其它群公告存在性）。
func (r *InMemoryRepository) GetAnnouncement(_ context.Context, groupID, announcementID string) (GroupAnnouncementItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.announcements[announcementID]
	if !ok || item.GroupID != groupID {
		return GroupAnnouncementItem{}, false, nil
	}
	return item, true, nil
}

func (r *InMemoryRepository) UpdateAnnouncement(_ context.Context, groupID string, req UpdateGroupAnnouncementRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	item, ok := r.announcements[req.AnnouncementID]
	if !ok || item.GroupID != groupID {
		return ErrAnnouncementNotFound
	}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Content != nil {
		item.Content = *req.Content
	}
	if req.IsPinned != nil {
		item.IsPinned = *req.IsPinned
	}
	r.announcements[req.AnnouncementID] = item
	return nil
}

func (r *InMemoryRepository) DeleteAnnouncement(_ context.Context, groupID string, announcementID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return ErrGroupNotFound
	}
	item, ok := r.announcements[announcementID]
	if !ok || item.GroupID != groupID {
		return ErrAnnouncementNotFound
	}
	delete(r.announcements, announcementID)
	return nil
}
