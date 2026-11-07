package admin

import (
	"context"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceModerationStore struct {
	momentTags momentTagStore
	factory    *persistence.Factory
	// announcements 由 system 模块实现（只暴露公告管理最小接口）：公告数据
	// 归 system 所有，admin 不再自建 system.Repository 副本。
	announcements systemmodule.AnnouncementAdminStore
}

// NewPersistenceModerationStore 生产仓储：标签/公告全部经 DB 与 system 模块
// 注入的公告适配器读写。bootstrap 必须注入 announcements；缺失时公告方法返回
// ErrNotImplemented（不再回退内存实现——生产 fail-fast，内存 store 仅测试使用）。
func NewPersistenceModerationStore(factory *persistence.Factory, announcements systemmodule.AnnouncementAdminStore, momentTags momentTagStore) *PersistenceModerationStore {
	return &PersistenceModerationStore{
		factory:       factory,
		announcements: announcements,
		momentTags:    momentTags,
	}
}

func (s *PersistenceModerationStore) queryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, s.factory)
}

func (s *PersistenceModerationStore) SearchTags(ctx context.Context, filter TagSearchFilter) (TagPage, error) {
	queryable, ok := s.queryable(ctx)
	if !ok {
		return TagPage{}, persistence.ErrNotImplemented
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
	page := TagPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		// 下一页游标取本页最后一条（排他 keyset）；此前取第 limit+1 条（下一页首条），
		// 下一页会把它自身排除 → 每个分页边界丢一个标签。
		last := page.Items[limit-1]
		page.Next = &tagCursor{Name: last.Name, UsageCount: last.UsageCount, CreatedAt: last.CreatedAt}
	}
	return page, nil
}

func (s *PersistenceModerationStore) GetTagDeleteState(ctx context.Context, name string) (bool, bool, error) {
	queryable, ok := s.queryable(ctx)
	if !ok {
		return false, false, persistence.ErrNotImplemented
	}
	tagName := normalizeTagName(name)
	var isBlocked bool
	if err := queryable.QueryRow(ctx, queryBlockedTagExists, tagName).Scan(&isBlocked); err != nil {
		return false, false, err
	}
	if s.momentTags == nil {
		return false, false, errMomentTagStoreNotWired
	}
	hasMomentRefs, err := s.momentTags.HasTagAssignments(ctx, tagName)
	if err != nil {
		return false, false, err
	}
	return isBlocked, hasMomentRefs, nil
}

// SetTagBan Postgres 版两态屏蔽：active=true 屏蔽（hardDelete 的删关联 +
// 屏蔽两步包进同一事务）；active=false 解屏蔽（DELETE 未命中行 no-op 成功，
// 解屏蔽后同名标签恢复可创建/命中）。
func (s *PersistenceModerationStore) SetTagBan(ctx context.Context, name string, active bool, hardDelete bool) error {
	if _, ok := s.queryable(ctx); !ok {
		return persistence.ErrNotImplemented
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
			if s.momentTags == nil {
				return errMomentTagStoreNotWired
			}
			if err := s.momentTags.DeleteTagAssignments(txCtx, tagName); err != nil {
				return err
			}
		}
		return txQueryable.Exec(txCtx, queryBlockTag, tagName, time.Now().UTC())
	})
}

func (s *PersistenceModerationStore) SearchAnnouncements(ctx context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	if s.announcements == nil {
		return systemmodule.AnnouncementPage{}, persistence.ErrNotImplemented
	}
	return s.announcements.SearchAnnouncements(ctx, filter)
}

func (s *PersistenceModerationStore) GetAnnouncement(ctx context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	if s.announcements == nil {
		return systemmodule.AnnouncementRecord{}, false, persistence.ErrNotImplemented
	}
	return s.announcements.GetAnnouncement(ctx, announcementID)
}

func (s *PersistenceModerationStore) CreateAnnouncement(ctx context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	if s.announcements == nil {
		return systemmodule.AnnouncementRecord{}, persistence.ErrNotImplemented
	}
	return s.announcements.CreateAnnouncement(ctx, input)
}

func (s *PersistenceModerationStore) UpdateAnnouncement(ctx context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	if s.announcements == nil {
		return systemmodule.AnnouncementRecord{}, persistence.ErrNotImplemented
	}
	return s.announcements.UpdateAnnouncement(ctx, input)
}

func (s *PersistenceModerationStore) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	if s.announcements == nil {
		return persistence.ErrNotImplemented
	}
	return s.announcements.DeleteAnnouncement(ctx, announcementID)
}

func normalizeTagSearchKeyword(keyword string) string {
	return strings.TrimSpace(keyword)
}
