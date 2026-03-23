package system

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu            sync.RWMutex
	announcements map[string]AnnouncementRecord
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{announcements: make(map[string]AnnouncementRecord)}
}

func (r *InMemoryRepository) GetMeta(context.Context) ApiVersionMeta {
	return ApiVersionMeta{Version: "v1", Status: ApiVersionStatusActive}
}

func (r *InMemoryRepository) Health(context.Context) (int, string, *HealthStatus) {
	return 0, "", &HealthStatus{Status: SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func (r *InMemoryRepository) ListAnnouncements(_ context.Context, limit int) ([]AnnouncementRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.sortedAnnouncementsLocked(false)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return cloneAnnouncementRecords(items), nil
}

func (r *InMemoryRepository) SearchAnnouncements(_ context.Context, filter AnnouncementSearchFilter) (AnnouncementPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.sortedAnnouncementsLocked(true)
	filtered := make([]AnnouncementRecord, 0, len(items))
	for _, item := range items {
		if filter.Severity != "" && item.Severity != filter.Severity {
			continue
		}
		if filter.Cursor != nil {
			if item.CreatedAt.After(filter.Cursor.CreatedAt) {
				continue
			}
			if item.CreatedAt.Equal(filter.Cursor.CreatedAt) && item.AnnouncementID >= filter.Cursor.AnnouncementID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := normalizeLimit(filter.Limit)
	page := AnnouncementPage{Items: make([]AnnouncementRecord, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			cursor := AnnouncementCursor{CreatedAt: item.CreatedAt, AnnouncementID: item.AnnouncementID}
			page.Next = &cursor
			break
		}
		page.Items = append(page.Items, cloneAnnouncementRecord(item))
	}
	return page, nil
}

func (r *InMemoryRepository) CreateAnnouncement(_ context.Context, input AnnouncementCreateInput) (AnnouncementRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	item := AnnouncementRecord{
		AnnouncementID: fmt.Sprintf("ann_%d", now.UnixNano()),
		Title:          strings.TrimSpace(input.Title),
		Content:        strings.TrimSpace(input.Content),
		Severity:       input.Severity,
		ActionURL:      strings.TrimSpace(input.ActionURL),
		CreatedAt:      now,
		CreatedBy:      input.Actor,
		SendPush:       input.SendPush,
		ViewCount:      0,
	}
	if input.SendPush {
		pushSentAt := now
		item.PushSentAt = &pushSentAt
	}
	r.announcements[item.AnnouncementID] = item
	return cloneAnnouncementRecord(item), nil
}

func (r *InMemoryRepository) UpdateAnnouncement(_ context.Context, input AnnouncementUpdateInput) (AnnouncementRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.announcements[input.AnnouncementID]
	if !ok || item.IsDeleted() {
		return AnnouncementRecord{}, errAnnouncementNotFound
	}
	if input.Title != nil {
		item.Title = strings.TrimSpace(*input.Title)
	}
	if input.Content != nil {
		item.Content = strings.TrimSpace(*input.Content)
	}
	if input.Severity != nil {
		item.Severity = *input.Severity
	}
	if input.ActionURL != nil {
		item.ActionURL = strings.TrimSpace(*input.ActionURL)
	}
	r.announcements[input.AnnouncementID] = item
	return cloneAnnouncementRecord(item), nil
}

func (r *InMemoryRepository) DeleteAnnouncement(_ context.Context, announcementID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.announcements[announcementID]
	if !ok {
		return errAnnouncementNotFound
	}
	if item.IsDeleted() {
		return nil
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	r.announcements[announcementID] = item
	return nil
}

func (r *InMemoryRepository) sortedAnnouncementsLocked(includeDeleted bool) []AnnouncementRecord {
	items := make([]AnnouncementRecord, 0, len(r.announcements))
	for _, item := range r.announcements {
		if !includeDeleted && item.IsDeleted() {
			continue
		}
		if includeDeleted || !item.IsDeleted() {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].AnnouncementID > items[j].AnnouncementID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if includeDeleted {
		return items
	}
	published := items[:0]
	for _, item := range items {
		if item.IsDeleted() {
			continue
		}
		published = append(published, item)
	}
	return published
}

func cloneAnnouncementRecord(item AnnouncementRecord) AnnouncementRecord {
	cloned := item
	if item.PushSentAt != nil {
		pushSentAt := item.PushSentAt.UTC()
		cloned.PushSentAt = &pushSentAt
	}
	if item.DeletedAt != nil {
		deletedAt := item.DeletedAt.UTC()
		cloned.DeletedAt = &deletedAt
	}
	return cloned
}

func cloneAnnouncementRecords(items []AnnouncementRecord) []AnnouncementRecord {
	cloned := make([]AnnouncementRecord, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneAnnouncementRecord(item))
	}
	return cloned
}
