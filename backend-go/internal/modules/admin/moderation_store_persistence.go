package admin

import (
	"context"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceModerationStore struct {
	factory *persistence.Factory
	system  systemmodule.Repository
	memory  *InMemoryModerationStore
}

func NewPersistenceModerationStore(factory *persistence.Factory, systemRepo systemmodule.Repository) *PersistenceModerationStore {
	if systemRepo == nil {
		systemRepo = NewInMemoryModerationStoreSystemAdapter()
	}
	return &PersistenceModerationStore{
		factory: factory,
		system:  systemRepo,
		memory:  NewInMemoryModerationStore(),
	}
}

func (s *PersistenceModerationStore) queryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, s.factory)
}

func (s *PersistenceModerationStore) SearchTags(ctx context.Context, filter TagSearchFilter) (TagPage, error) {
	queryable, ok := s.queryable(ctx)
	if !ok {
		return s.memory.SearchTags(ctx, filter)
	}
	limit := normalizeCursorLimit(filter.Limit)
	cursorUsageCount := any(nil)
	cursorName := ""
	if filter.Cursor != nil {
		cursorUsageCount = filter.Cursor.UsageCount
		cursorName = filter.Cursor.Name
	}
	rows, err := queryable.Query(ctx, querySearchTags, normalizeTagSearchKeyword(filter.Keyword), cursorUsageCount, cursorName, limit+1)
	if err != nil {
		return TagPage{}, err
	}
	defer rows.Close()
	items := make([]TagRecord, 0, limit+1)
	for rows.Next() {
		var item TagRecord
		if err := rows.Scan(&item.Name, &item.UsageCount, &item.CreatedAt, &item.IsBlocked); err != nil {
			return TagPage{}, err
		}
		item.CreatedAt = item.CreatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return TagPage{}, err
	}
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

func (s *PersistenceModerationStore) GetTagDeleteState(ctx context.Context, name string) (bool, bool, error) {
	queryable, ok := s.queryable(ctx)
	if !ok {
		return s.memory.GetTagDeleteState(ctx, name)
	}
	tagName := normalizeTagName(name)
	var isBlocked bool
	var hasMomentRefs bool
	if err := queryable.QueryRow(ctx, queryTagDeleteState, tagName).Scan(&isBlocked, &hasMomentRefs); err != nil {
		return false, false, err
	}
	return isBlocked, hasMomentRefs, nil
}

// SetTagBan Postgres 版两态屏蔽：active=true 屏蔽（hardDelete 的删关联 +
// 屏蔽两步包进同一事务）；active=false 解屏蔽（DELETE 未命中行 no-op 成功，
// 解屏蔽后同名标签恢复可创建/命中）。
func (s *PersistenceModerationStore) SetTagBan(ctx context.Context, name string, active bool, hardDelete bool) error {
	if _, ok := s.queryable(ctx); !ok {
		return s.memory.SetTagBan(ctx, name, active, hardDelete)
	}
	tagName := normalizeTagName(name)
	if !active {
		queryable, ok := s.queryable(ctx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(ctx, queryUnblockTag, tagName); err != nil {
			return err
		}
		return nil
	}
	isBlocked, hasMomentRefs, err := s.GetTagDeleteState(ctx, tagName)
	if err != nil {
		return err
	}
	if !isBlocked && !hasMomentRefs {
		return errTagNotFound
	}
	return persistence.RunInTransaction(ctx, s.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, s.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if hardDelete {
			if err := txQueryable.Exec(txCtx, queryDeleteMomentTagAssignments, tagName); err != nil {
				return err
			}
		}
		return txQueryable.Exec(txCtx, queryBlockTag, tagName, time.Now().UTC())
	})
}

func (s *PersistenceModerationStore) SearchAnnouncements(ctx context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	return s.system.SearchAnnouncements(ctx, filter)
}

func (s *PersistenceModerationStore) GetAnnouncement(ctx context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	return s.system.GetAnnouncement(ctx, announcementID)
}

func (s *PersistenceModerationStore) CreateAnnouncement(ctx context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	return s.system.CreateAnnouncement(ctx, input)
}

func (s *PersistenceModerationStore) UpdateAnnouncement(ctx context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	return s.system.UpdateAnnouncement(ctx, input)
}

func (s *PersistenceModerationStore) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	return s.system.DeleteAnnouncement(ctx, announcementID)
}

type inMemoryModerationStoreSystemAdapter struct{ repo *InMemoryModerationStore }

func NewInMemoryModerationStoreSystemAdapter() systemmodule.Repository {
	store := NewInMemoryModerationStore()
	return &inMemoryModerationStoreSystemAdapter{repo: store}
}

func (a *inMemoryModerationStoreSystemAdapter) GetMeta(ctx context.Context) systemmodule.ApiVersionMeta {
	return systemmodule.ApiVersionMeta{Version: "v1", Status: systemmodule.ApiVersionStatusActive}
}

func (a *inMemoryModerationStoreSystemAdapter) Health(ctx context.Context) (int, string, *systemmodule.HealthStatus) {
	status := &systemmodule.HealthStatus{Status: systemmodule.SystemHealthHealthy, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	return 0, "", status
}

func (a *inMemoryModerationStoreSystemAdapter) ListAnnouncements(ctx context.Context, limit int) ([]systemmodule.AnnouncementRecord, error) {
	page, err := a.repo.SearchAnnouncements(ctx, systemmodule.AnnouncementSearchFilter{Limit: limit})
	return page.Items, err
}

func (a *inMemoryModerationStoreSystemAdapter) SearchAnnouncements(ctx context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	return a.repo.SearchAnnouncements(ctx, filter)
}

func (a *inMemoryModerationStoreSystemAdapter) GetAnnouncement(ctx context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	return a.repo.GetAnnouncement(ctx, announcementID)
}

func (a *inMemoryModerationStoreSystemAdapter) CreateAnnouncement(ctx context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	return a.repo.CreateAnnouncement(ctx, input)
}

func (a *inMemoryModerationStoreSystemAdapter) UpdateAnnouncement(ctx context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	return a.repo.UpdateAnnouncement(ctx, input)
}

func (a *inMemoryModerationStoreSystemAdapter) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	return a.repo.DeleteAnnouncement(ctx, announcementID)
}

func normalizeTagSearchKeyword(keyword string) string {
	return strings.TrimSpace(keyword)
}
