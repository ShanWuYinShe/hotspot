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
	for rows.Next() {
		item, err := scanAdminGroupRow(rows)
		if err != nil {
			return adminGroupPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminGroupPage{}, err
	}
	page := adminGroupPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		page.Next = &adminGroupCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), GroupID: last.Group.GroupID}
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

func (r *PostgresRepository) DismissGroup(ctx context.Context, groupID string) (AdminGroupItem, bool, error) {
	if _, ok := r.queryable(ctx); !ok {
		return AdminGroupItem{}, false, persistence.ErrNotImplemented
	}
	item, found, err := r.GetGroup(ctx, groupID)
	if err != nil || !found {
		return AdminGroupItem{}, found, err
	}
	// 解散 + 级联作废 pending 申请/邀请包进同一事务（spec api_spec_admin.md
	// groups/dismiss 契约级）：避免「群已解散、申请仍可被同意」的中间态。
	err = persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryAdminDismissGroup, groupID); err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := txQueryable.Exec(txCtx, queryAdminRevokePendingGroupApplications, groupID, now); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryAdminRevokePendingGroupInvites, groupID, now)
	})
	if err != nil {
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
