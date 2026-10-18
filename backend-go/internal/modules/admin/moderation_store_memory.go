package admin

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
)

type InMemoryModerationStore struct {
	mu            sync.RWMutex
	tags          map[string]TagRecord
	blockedTags   map[string]struct{}
	announcements map[string]systemmodule.AnnouncementRecord
}

func NewInMemoryModerationStore() *InMemoryModerationStore {
	now := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	return &InMemoryModerationStore{
		tags: map[string]TagRecord{
			"hotspot": {Name: "hotspot", UsageCount: 3, CreatedAt: now},
			"travel":  {Name: "travel", UsageCount: 2, CreatedAt: now.Add(-time.Hour)},
		},
		blockedTags:   make(map[string]struct{}),
		announcements: make(map[string]systemmodule.AnnouncementRecord),
	}
}

func (s *InMemoryModerationStore) SearchTags(_ context.Context, filter TagSearchFilter) (TagPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]TagRecord, 0, len(s.tags))
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	for _, item := range s.tags {
		// 已屏蔽标签仍返回，但带 is_blocked 标记（与 Postgres 实现同口径）
		if _, blocked := s.blockedTags[item.Name]; blocked {
			item.IsBlocked = true
		}
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

func (s *InMemoryModerationStore) GetTagDeleteState(_ context.Context, name string) (bool, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tagName := normalizeTagName(name)
	_, isBlocked := s.blockedTags[tagName]
	_, hasRefs := s.tags[tagName]
	return isBlocked, hasRefs, nil
}

// SetTagBan 内存版两态屏蔽：active=true 屏蔽（hardDelete 一并删 tag 记录）；
// active=false 解屏蔽（未屏蔽时 no-op 成功）。屏蔽不存在的标签返回 errTagNotFound。
func (s *InMemoryModerationStore) SetTagBan(_ context.Context, name string, active bool, hardDelete bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tagName := normalizeTagName(name)
	_, hasRefs := s.tags[tagName]
	_, isBlocked := s.blockedTags[tagName]
	if active {
		if !hasRefs && !isBlocked {
			return errTagNotFound
		}
		if hardDelete {
			delete(s.tags, tagName)
		}
		s.blockedTags[tagName] = struct{}{}
		return nil
	}
	// 解屏蔽：未屏蔽标签 no-op 成功；解屏蔽后同名标签恢复可创建/命中。
	delete(s.blockedTags, tagName)
	return nil
}

func (s *InMemoryModerationStore) SearchAnnouncements(_ context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]systemmodule.AnnouncementRecord, 0, len(s.announcements))
	for _, item := range s.announcements {
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

func (s *InMemoryModerationStore) GetAnnouncement(_ context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.announcements[announcementID]
	if !ok {
		return systemmodule.AnnouncementRecord{}, false, nil
	}
	return item, true, nil
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
		return systemmodule.AnnouncementRecord{}, systemmodule.ErrAnnouncementNotFound()
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
		return systemmodule.ErrAnnouncementNotFound()
	}
	if item.IsDeleted() {
		return nil
	}
	now := time.Now().UTC()
	item.DeletedAt = &now
	s.announcements[announcementID] = item
	return nil
}
