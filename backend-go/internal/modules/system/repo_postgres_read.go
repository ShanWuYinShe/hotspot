package system

import (
	"context"
	"hotspot/backend-go/internal/platform/types"
)

func (r *PostgresRepository) ListAnnouncements(ctx context.Context, limit int) ([]AnnouncementRecord, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := queryable.Query(ctx, queryListPublicAnnouncements, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AnnouncementRecord, 0)
	for rows.Next() {
		item, err := scanAnnouncement(rows, false)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) SearchAnnouncements(ctx context.Context, filter AnnouncementSearchFilter) (AnnouncementPage, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return AnnouncementPage{}, err
	}
	limit := types.NormalizeLimit(filter.Limit)
	cursorCreatedAt := any(nil)
	cursorAnnouncementID := ""
	if filter.Cursor != nil {
		cursorCreatedAt = filter.Cursor.CreatedAt.UTC()
		cursorAnnouncementID = filter.Cursor.AnnouncementID
	}
	rows, err := queryable.Query(ctx, querySearchAnnouncements, string(filter.Severity), cursorCreatedAt, cursorAnnouncementID, limit+1)
	if err != nil {
		return AnnouncementPage{}, err
	}
	defer rows.Close()
	items := make([]AnnouncementRecord, 0, limit+1)
	for rows.Next() {
		item, err := scanAnnouncement(rows, true)
		if err != nil {
			return AnnouncementPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AnnouncementPage{}, err
	}
	page := AnnouncementPage{Items: make([]AnnouncementRecord, 0, limit)}
	for idx, item := range items {
		if idx >= limit {
			// 下一页游标取本页最后一条（排他 keyset：(created_at, announcement_id) <
			// 游标）。此前取第 limit+1 条（下一页首条），下一页会把它自身排除 →
			// 每个分页边界静默丢一条公告（生产后台检索翻页少数据）。
			last := items[limit-1]
			cursor := AnnouncementCursor{CreatedAt: last.CreatedAt, AnnouncementID: last.AnnouncementID}
			page.Next = &cursor
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}
