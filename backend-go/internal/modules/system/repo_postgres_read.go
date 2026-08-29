package system

import (
	"context"
	"hotspot/backend-go/internal/platform/types"

	"hotspot/backend-go/internal/platform/persistence"
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
			cursor := AnnouncementCursor{CreatedAt: item.CreatedAt, AnnouncementID: item.AnnouncementID}
			page.Next = &cursor
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

var _ persistence.QueryableDB
