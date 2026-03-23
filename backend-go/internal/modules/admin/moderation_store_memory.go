package admin

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

type InMemoryModerationStore struct {
	mu            sync.RWMutex
	tags          map[string]TagRecord
	announcements map[string]systemmodule.AnnouncementRecord
}

func NewInMemoryModerationStore() *InMemoryModerationStore {
	now := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	return &InMemoryModerationStore{
		tags: map[string]TagRecord{
			"hotspot": {Name: "hotspot", UsageCount: 3, CreatedAt: now},
			"travel":  {Name: "travel", UsageCount: 2, CreatedAt: now.Add(-time.Hour)},
		},
		announcements: make(map[string]systemmodule.AnnouncementRecord),
	}
}

func (s *InMemoryModerationStore) SearchTags(_ context.Context, filter TagSearchFilter) (TagPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]TagRecord, 0, len(s.tags))
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	for _, item := range s.tags {
		if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) {
			continue
		}
		if filter.Cursor != nil {
			if item.UsageCount > filter.Cursor.UsageCount {
				continue
			}
			if item.UsageCount == filter.Cursor.UsageCount && item.Name <= filter.Cursor.Name {
				continue
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UsageCount == items[j].UsageCount {
			return items[i].Name < items[j].Name
		}
		return items[i].UsageCount > items[j].UsageCount
	})
	limit := normalizeCursorLimit(filter.Limit)
	page := TagPage{Items: make([]TagRecord, 0, limit)}
	for idx, item := range items {
		if idx >= limit {
			cursor := tagCursor{Name: item.Name, UsageCount: item.UsageCount, CreatedAt: item.CreatedAt}
			page.Next = &cursor
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (s *InMemoryModerationStore) DeleteTag(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tags, normalizeTagName(name))
	return nil
}

func (s *InMemoryModerationStore) SearchAnnouncements(_ context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]systemmodule.AnnouncementRecord, 0, len(s.announcements))
	for _, item := range s.announcements {
		if item.IsDeleted() {
			continue
		}
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
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].AnnouncementID > items[j].AnnouncementID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	limit := normalizeCursorLimit(filter.Limit)
	page := systemmodule.AnnouncementPage{Items: make([]systemmodule.AnnouncementRecord, 0, limit)}
	for idx, item := range items {
		if idx >= limit {
			cursor := systemmodule.AnnouncementCursor{CreatedAt: item.CreatedAt, AnnouncementID: item.AnnouncementID}
			page.Next = &cursor
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (s *InMemoryModerationStore) CreateAnnouncement(_ context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item := systemmodule.AnnouncementRecord{
		AnnouncementID: "ann_" + now.Format("20060102150405.000000000"),
		Title:          input.Title,
		Content:        input.Content,
		Severity:       input.Severity,
		ActionURL:      input.ActionURL,
		CreatedAt:      now,
		CreatedBy:      input.Actor,
		SendPush:       input.SendPush,
		ViewCount:      0,
	}
	if input.SendPush {
		pushSentAt := now
		item.PushSentAt = &pushSentAt
	}
	s.announcements[item.AnnouncementID] = item
	return item, nil
}

func (s *InMemoryModerationStore) UpdateAnnouncement(_ context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.announcements[input.AnnouncementID]
	if !ok || item.IsDeleted() {
		return systemmodule.AnnouncementRecord{}, errors.New("announcement not found")
	}
	if input.Title != nil {
		item.Title = *input.Title
	}
	if input.Content != nil {
		item.Content = *input.Content
	}
	if input.Severity != nil {
		item.Severity = *input.Severity
	}
	if input.ActionURL != nil {
		item.ActionURL = *input.ActionURL
	}
	s.announcements[input.AnnouncementID] = item
	return item, nil
}

func (s *InMemoryModerationStore) DeleteAnnouncement(_ context.Context, announcementID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.announcements[announcementID]
	if !ok {
		return errors.New("announcement not found")
	}
	if item.IsDeleted() {
		return nil
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	s.announcements[announcementID] = item
	return nil
}
