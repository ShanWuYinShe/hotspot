package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, bool) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}

func (r *PostgresRepository) GetActor(ctx context.Context, userID string) (AdminActor, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.GetActor(ctx, userID)
	}
	return r.getActorFromDB(ctx, queryable, userID)
}

func (r *PostgresRepository) ListPermissions(ctx context.Context) ([]AdminPermissionItem, error) {
	if _, ok := r.queryable(); !ok {
		return r.fallback.ListPermissions(ctx)
	}
	items := make([]AdminPermissionItem, len(permissionDictionary))
	copy(items, permissionDictionary)
	return items, nil
}

func (r *PostgresRepository) ListRoles(ctx context.Context) ([]AdminRoleItem, error) {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.ListRoles(ctx)
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
		return r.fallback.CreateRole(ctx, req)
	}
	name := normalizeRoleName(req.Name)
	existing, found, err := r.getRoleByName(ctx, queryable, name)
	if err != nil {
		return AdminRoleItem{}, err
	}
	if found && !existing.deleted {
		return AdminRoleItem{}, &httpx.APIError{Code: 10400, Message: "角色已存在"}
	}
	now := time.Now().UTC()
	item := AdminRoleItem{
		RoleID:      fmt.Sprintf("role_%d", now.UnixNano()),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Permissions: []string{},
	}
	if err := queryable.Exec(ctx, queryRoleInsert, item.RoleID, item.Name, nullableString(item.Description), now, now); err != nil {
		if isUniqueViolation(err) {
			return AdminRoleItem{}, &httpx.APIError{Code: 10400, Message: "角色已存在"}
		}
		return AdminRoleItem{}, err
	}
	return item, nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, req AdminRbacRoleUpdateRequest) error {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.UpdateRole(ctx, req)
	}
	item, found, err := r.getRoleByID(ctx, queryable, req.RoleID)
	if err != nil {
		return err
	}
	if !found || item.deleted {
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
	}
	name := item.item.Name
	if req.Name != nil {
		name = normalizeRoleName(*req.Name)
		existing, found, err := r.getRoleByName(ctx, queryable, name)
		if err != nil {
			return err
		}
		if found && !existing.deleted && existing.item.RoleID != req.RoleID {
			return &httpx.APIError{Code: 10400, Message: "角色已存在"}
		}
	}
	description := item.item.Description
	if req.ClearDescription {
		description = ""
	} else if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	if err := queryable.Exec(ctx, queryRoleUpdate, req.RoleID, name, nullableString(description), time.Now().UTC()); err != nil {
		if isUniqueViolation(err) {
			return &httpx.APIError{Code: 10400, Message: "角色已存在"}
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteRole(ctx context.Context, roleID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.DeleteRole(ctx, roleID)
	}
	item, found, err := r.getRoleByID(ctx, queryable, roleID)
	if err != nil {
		return err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
	}
	if item.deleted {
		return nil
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryRoleDeleteAssignments, roleID); err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryRoleDelete, roleID, now); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) SetRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.SetRolePermissions(ctx, roleID, permissionKeys)
	}
	item, found, err := r.getRoleByID(ctx, queryable, roleID)
	if err != nil {
		return err
	}
	if !found || item.deleted {
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
	}
	if err := queryable.Exec(ctx, queryRoleReplacePermissions, roleID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, permissionKey := range permissionKeys {
		if err := queryable.Exec(ctx, queryRoleInsertPermission, roleID, permissionKey, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) ListAdmins(ctx context.Context, pagination PaginationParams) (adminListPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.ListAdmins(ctx, pagination)
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
	limit := normalizeLimit(pagination.Limit)
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
		return r.fallback.GrantAdmin(ctx, grant)
	}
	if _, found, err := r.lookupUser(ctx, queryable, grant.UserID); err != nil {
		return err
	} else if !found {
		return &httpx.APIError{Code: 10300, Message: "用户不存在"}
	}
	for _, roleID := range grant.RoleIDs {
		item, found, err := r.getRoleByID(ctx, queryable, roleID)
		if err != nil {
			return err
		}
		if !found || item.deleted {
			return &httpx.APIError{Code: 10300, Message: "角色不存在"}
		}
	}
	grantedAt := grant.GrantedAt
	if grantedAt.IsZero() {
		grantedAt = time.Now().UTC()
	}
	if err := queryable.Exec(ctx, queryAdminUpsert, grant.UserID, grant.GrantAllPermissions, grantedAt); err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryAdminReplaceRoles, grant.UserID); err != nil {
		return err
	}
	for _, roleID := range grant.RoleIDs {
		if err := queryable.Exec(ctx, queryAdminInsertRole, grant.UserID, roleID, grantedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) RevokeAdmin(ctx context.Context, userID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return r.fallback.RevokeAdmin(ctx, userID)
	}
	_, found, err := r.getActorFromDB(ctx, queryable, userID)
	if err != nil {
		return err
	}
	if !found {
		return &httpx.APIError{Code: 10300, Message: "管理员不存在"}
	}
	if err := queryable.Exec(ctx, queryAdminReplaceRoles, userID); err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryAdminRevoke, userID, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
