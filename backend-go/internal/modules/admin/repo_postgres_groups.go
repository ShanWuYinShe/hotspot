package admin

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/types"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) SearchGroups(ctx context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return adminGroupPage{}, persistence.ErrNotImplemented
	}
	cursorRaw := types.PaginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminGroupCursor(cursorRaw)
	if apiErr != nil {
		return adminGroupPage{}, apiErr
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	// keyset 谓词与 LIMIT 下推 SQL（审查 P1-1，同 SearchUsers）。
	var cursorAt, cursorID any
	if cursorRaw != "" {
		cursorAt, cursorID = cursor.CreatedAt.UTC(), cursor.GroupID
	}
	rows, err := queryable.Query(ctx, queryAdminSearchGroups, req.GroupID, req.Name, req.OwnerID, cursorAt, cursorID, limit+1)
	if err != nil {
		return adminGroupPage{}, err
	}
	defer rows.Close()
	items := make([]AdminGroupItem, 0, limit+1)
	rawCreatedAt := make([]time.Time, 0, limit+1)
	for rows.Next() {
		item, createdAt, err := scanAdminGroupRowWithCreatedAt(rows)
		if err != nil {
			return adminGroupPage{}, err
		}
		items = append(items, item)
		rawCreatedAt = append(rawCreatedAt, createdAt)
	}
	if err := rows.Err(); err != nil {
		return adminGroupPage{}, err
	}
	page := adminGroupPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &adminGroupCursor{CreatedAt: rawCreatedAt[limit-1], GroupID: last.Group.GroupID}
	}
	return page, nil
}

func (r *PostgresRepository) GetGroup(ctx context.Context, groupID string) (AdminGroupItem, bool, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return AdminGroupItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanAdminGroupRow(queryable.QueryRow(ctx, queryAdminGetGroup, groupID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminGroupItem{}, false, nil
		}
		return AdminGroupItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) ListGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminListGroupMemberIDs, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		items = append(items, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
