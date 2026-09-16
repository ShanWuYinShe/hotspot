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
	rows, err := queryable.Query(ctx, queryAdminSearchGroups, req.GroupID, req.Name, req.OwnerID)
	if err != nil {
		return adminGroupPage{}, err
	}
	defer rows.Close()
	items := make([]AdminGroupItem, 0)
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
	cursorRaw := types.PaginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminGroupCursor(cursorRaw)
	if apiErr != nil {
		return adminGroupPage{}, apiErr
	}
	filtered := make([]AdminGroupItem, 0, len(items))
	for _, item := range items {
		if cursorRaw != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.Group.GroupID <= cursor.GroupID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	page := adminGroupPage{Items: make([]AdminGroupItem, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			page.Next = &adminGroupCursor{CreatedAt: timeFromRFC3339(item.CreatedAt), GroupID: item.Group.GroupID}
			break
		}
		page.Items = append(page.Items, item)
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
