package admin

import (
	"context"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/types"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

func (r *PostgresRepository) GetActor(ctx context.Context, userID string) (AdminActor, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminActor{}, false, persistence.ErrNotImplemented
	}
	return r.getActorFromDB(ctx, queryable, userID)
}

// IsAdmin 真实验证 userID 是否为管理员（查 admin_users 表）。
// 供平台 AdminAccess 中间件与业务侧管理员判断使用。
func (r *PostgresRepository) IsAdmin(ctx context.Context, userID string) (bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	_, found, err := r.getAdminRecordByUserID(ctx, queryable, userID)
	return found, err
}

func (r *PostgresRepository) ListPermissions(ctx context.Context) ([]AdminPermissionItem, error) {
	if _, ok := r.queryable(); !ok {
		return nil, persistence.ErrNotImplemented
	}
	items := make([]AdminPermissionItem, len(permissionDictionary))
	copy(items, permissionDictionary)
	return items, nil
}

func (r *PostgresRepository) ListRoles(ctx context.Context) ([]AdminRoleItem, error) {
	queryable, ok := r.queryable()
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryRoleList)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminRoleItem, 0)
	for rows.Next() {
		var item AdminRoleItem
		var permissions []string
		if err := rows.Scan(&item.RoleID, &item.Name, &item.Description, &permissions); err != nil {
			return nil, err
		}
		item.Permissions = sortAndUniqueStrings(permissions)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) CreateRole(ctx context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminRoleItem{}, persistence.ErrNotImplemented
	}
	name := normalizeRoleName(req.Name)
	existing, found, err := r.getRoleByName(ctx, queryable, name)
	if err != nil {
		return AdminRoleItem{}, err
	}
	if found && !existing.deleted {
		return AdminRoleItem{}, &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
	}
	now := time.Now().UTC()
	item := AdminRoleItem{
		RoleID:      idgen.MustWithPrefix("role_"),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Permissions: []string{},
	}
	if err := queryable.Exec(ctx, queryRoleInsert, item.RoleID, item.Name, nullableString(item.Description), now, now); err != nil {
		if persistence.IsUniqueViolation(err) {
			return AdminRoleItem{}, &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
		}
		return AdminRoleItem{}, err
	}
	return item, nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	item, found, err := r.getRoleByID(ctx, queryable, req.RoleID)
	if err != nil {
		return err
	}
	if !found || item.deleted {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	name := item.item.Name
	if req.Name != nil {
		name = normalizeRoleName(*req.Name)
		existing, found, err := r.getRoleByName(ctx, queryable, name)
		if err != nil {
			return err
		}
		if found && !existing.deleted && existing.item.RoleID != req.RoleID {
			return &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
		}
	}
	description := item.item.Description
	if req.ClearDescription {
		description = ""
	} else if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	if err := queryable.Exec(ctx, queryRoleUpdate, req.RoleID, name, nullableString(description), time.Now().UTC()); err != nil {
		if persistence.IsUniqueViolation(err) {
			return &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteRole(ctx context.Context, roleID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	item, found, err := r.getRoleByID(ctx, queryable, roleID)
	if err != nil {
		return err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	if item.deleted {
		return nil
	}
	// Wrap the assignment clear and the role soft-delete in a single tx so a
	// failure cannot leave assignments pointing at a soft-deleted role.
	now := time.Now().UTC()
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryRoleDeleteAssignments, roleID); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryRoleDelete, roleID, now)
	})
}

func (r *PostgresRepository) SetRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	item, found, err := r.getRoleByID(ctx, queryable, roleID)
	if err != nil {
		return err
	}
	if !found || item.deleted {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	// Wrap the permission delete + batch insert in a single tx so a failure
	// cannot wipe a role's permissions with nothing replacing them.
	now := time.Now().UTC()
	keys := permissionKeys
	if keys == nil {
		keys = []string{}
	}
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryRoleReplacePermissions, roleID); err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		return txQueryable.Exec(txCtx, queryRoleInsertPermissionsBatch, roleID, now, keys)
	})
}

func (r *PostgresRepository) ListAdmins(ctx context.Context, pagination PaginationParams) (adminListPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return adminListPage{}, persistence.ErrNotImplemented
	}
	cursor, apiErr := decodeAdminCursor(pagination.Cursor)
	if apiErr != nil {
		return adminListPage{}, apiErr
	}
	records, err := r.listAdminsFromDB(ctx, queryable)
	if err != nil {
		return adminListPage{}, err
	}
	filtered := make([]adminListRecord, 0, len(records))
	for _, record := range records {
		if pagination.Cursor != "" {
			if record.GrantedAt.After(cursor.GrantedAt) {
				continue
			}
			if record.GrantedAt.Equal(cursor.GrantedAt) && record.User.UserID <= cursor.UserID {
				continue
			}
		}
		filtered = append(filtered, record)
	}
	limit := types.NormalizeLimit(pagination.Limit)
	page := adminListPage{Items: make([]AdminOperatorItem, 0, limit)}
	for idx, record := range filtered {
		if idx >= limit {
			page.Next = &adminCursor{GrantedAt: record.GrantedAt, UserID: record.User.UserID}
			break
		}
		page.Items = append(page.Items, AdminOperatorItem{User: record.User, Permissions: record.permissionsForResponse(), Roles: record.rolesForResponse()})
	}
	return page, nil
}

func (r *PostgresRepository) GrantAdmin(ctx context.Context, grant AdminGrantUpsert) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if _, found, err := r.lookupUser(ctx, queryable, grant.UserID); err != nil {
		return err
	} else if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	}
	for _, roleID := range grant.RoleIDs {
		item, found, err := r.getRoleByID(ctx, queryable, roleID)
		if err != nil {
			return err
		}
		if !found || item.deleted {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
		}
	}
	grantedAt := grant.GrantedAt
	if grantedAt.IsZero() {
		grantedAt = time.Now().UTC()
	}
	// upsert + 清旧角色 + 逐角色插入三步包进同一事务：中途失败不会留下
	// 半套角色授权。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryAdminUpsert, grant.UserID, grant.GrantAllPermissions, grantedAt); err != nil {
			return err
		}
		if err := txQueryable.Exec(txCtx, queryAdminReplaceRoles, grant.UserID); err != nil {
			return err
		}
		for _, roleID := range grant.RoleIDs {
			if err := txQueryable.Exec(txCtx, queryAdminInsertRole, grant.UserID, roleID, grantedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepository) RevokeAdmin(ctx context.Context, userID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, found, err := r.getActorFromDB(ctx, queryable, userID)
	if err != nil {
		return err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "管理员不存在"}
	}
	// 清角色 + 标记撤销两步包进同一事务。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryAdminReplaceRoles, userID); err != nil {
			return err
		}
		return txQueryable.Exec(txCtx, queryAdminRevoke, userID, time.Now().UTC())
	})
}
