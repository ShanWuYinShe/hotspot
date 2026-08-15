package admin

import (
	"context"
	"fmt"
	"hotspot/backend-go/internal/platform/types"
	"sort"
	"strings"

	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

func (r *InMemoryRepository) GetActor(_ context.Context, userID string) (AdminActor, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if userID == "admin" {
		return AdminActor{
			UserID:              "admin",
			Permissions:         []string{wildcardPermission},
			Roles:               []AdminRoleBasic{{RoleID: "role_super", Name: "Super Admin"}},
			GrantAllPermissions: true,
		}, true, nil
	}
	record, ok := r.admins[userID]
	if !ok || record.revoked {
		return AdminActor{}, false, nil
	}
	return r.actorFromRecordLocked(record)
}
func (r *InMemoryRepository) IsAdmin(_ context.Context, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.admins[userID]
	return ok, nil
}

func (r *InMemoryRepository) ListPermissions(context.Context) ([]AdminPermissionItem, error) {
	items := make([]AdminPermissionItem, len(permissionDictionary))
	copy(items, permissionDictionary)
	return items, nil
}

func (r *InMemoryRepository) ListRoles(context.Context) ([]AdminRoleItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]AdminRoleItem, 0, len(r.roles))
	for _, record := range r.roles {
		if record.deleted {
			continue
		}
		item := record.item
		item.Permissions = cloneStrings(item.Permissions)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].RoleID < items[j].RoleID
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (r *InMemoryRepository) CreateRole(_ context.Context, req AdminRbacRoleCreateRequest) (AdminRoleItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := normalizeRoleName(req.Name)
	for _, record := range r.roles {
		if record.deleted {
			continue
		}
		if strings.EqualFold(record.item.Name, name) {
			return AdminRoleItem{}, &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
		}
	}
	now := time.Now().UTC()
	r.seq++
	item := AdminRoleItem{
		RoleID:      fmt.Sprintf("role_%d_%d", now.UnixNano(), r.seq),
		Name:        name,
		Description: types.StrPtr(strings.TrimSpace(req.Description)),
		Permissions: []string{},
	}
	r.roles[item.RoleID] = roleRecord{item: item, updatedAt: now}
	delete(r.deletedRoleIDs, item.RoleID)
	return item, nil
}

func (r *InMemoryRepository) UpdateRole(_ context.Context, req AdminRbacRoleUpdateRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.roles[req.RoleID]
	if !ok || record.deleted {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	if req.Name != nil {
		name := normalizeRoleName(*req.Name)
		for _, other := range r.roles {
			if other.deleted || other.item.RoleID == req.RoleID {
				continue
			}
			if strings.EqualFold(other.item.Name, name) {
				return &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
			}
		}
		record.item.Name = name
	}
	if req.ClearDescription {
		record.item.Description = nil
	} else if req.Description != nil {
		record.item.Description = types.StrPtr(strings.TrimSpace(*req.Description))
	}
	record.updatedAt = time.Now().UTC()
	r.roles[req.RoleID] = record
	return nil
}

func (r *InMemoryRepository) DeleteRole(_ context.Context, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.roles[roleID]
	if !ok {
		if _, deleted := r.deletedRoleIDs[roleID]; deleted {
			return nil
		}
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	if record.deleted {
		return nil
	}
	record.deleted = true
	r.roles[roleID] = record
	r.deletedRoleIDs[roleID] = struct{}{}
	for userID, grant := range r.admins {
		filtered := make([]string, 0, len(grant.roleIDs))
		for _, assigned := range grant.roleIDs {
			if assigned != roleID {
				filtered = append(filtered, assigned)
			}
		}
		grant.roleIDs = filtered
		r.admins[userID] = grant
	}
	return nil
}

func (r *InMemoryRepository) SetRolePermissions(_ context.Context, roleID string, permissionKeys []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.roles[roleID]
	if !ok || record.deleted {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
	}
	record.item.Permissions = cloneStrings(permissionKeys)
	record.updatedAt = time.Now().UTC()
	r.roles[roleID] = record
	return nil
}

func (r *InMemoryRepository) ListAdmins(_ context.Context, pagination PaginationParams) (adminListPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminCursor(pagination.Cursor)
	if apiErr != nil {
		return adminListPage{}, apiErr
	}
	items := make([]adminGrantRecord, 0, len(r.admins))
	for _, grant := range r.admins {
		if grant.revoked {
			continue
		}
		items = append(items, grant)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].grantedAt.Equal(items[j].grantedAt) {
			return items[i].userID < items[j].userID
		}
		return items[i].grantedAt.After(items[j].grantedAt)
	})
	filtered := make([]adminGrantRecord, 0, len(items))
	for _, item := range items {
		if pagination.Cursor != "" {
			if item.grantedAt.After(cursor.GrantedAt) {
				continue
			}
			if item.grantedAt.Equal(cursor.GrantedAt) && item.userID <= cursor.UserID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(pagination.Limit)
	page := adminListPage{Items: make([]AdminOperatorItem, 0, limit)}
	for idx, grant := range filtered {
		if idx >= limit {
			page.Next = &adminCursor{GrantedAt: grant.grantedAt, UserID: grant.userID}
			break
		}
		actor, _, _ := r.actorFromRecordLocked(grant)
		user, _ := r.lookupUserLocked(grant.userID)
		page.Items = append(page.Items, AdminOperatorItem{User: user, GrantedAt: grant.grantedAt.UTC().Format(time.RFC3339), Permissions: cloneStrings(actor.Permissions), Roles: append([]AdminRoleBasic(nil), actor.Roles...)})
	}
	return page, nil
}

func (r *InMemoryRepository) GrantAdmin(_ context.Context, grant AdminGrantUpsert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookupUserLocked(grant.UserID); !ok {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	}
	for _, roleID := range grant.RoleIDs {
		record, ok := r.roles[roleID]
		if !ok || record.deleted {
			return &httpx.APIError{Code: httpx.CodeNotFound, Message: "角色不存在"}
		}
	}
	grantedAt := grant.GrantedAt
	if grantedAt.IsZero() {
		grantedAt = time.Now().UTC()
	}
	r.admins[grant.UserID] = adminGrantRecord{
		userID:              grant.UserID,
		grantAllPermissions: grant.GrantAllPermissions,
		roleIDs:             cloneStrings(grant.RoleIDs),
		grantedAt:           grantedAt,
		revoked:             false,
	}
	delete(r.revokedAdmins, grant.UserID)
	return nil
}

func (r *InMemoryRepository) RevokeAdmin(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.admins[userID]
	if !ok {
		if _, revoked := r.revokedAdmins[userID]; revoked {
			return nil
		}
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "管理员不存在"}
	}
	if record.revoked {
		return nil
	}
	record.revoked = true
	r.admins[userID] = record
	r.revokedAdmins[userID] = struct{}{}
	return nil
}

func (r *InMemoryRepository) actorFromRecordLocked(record adminGrantRecord) (AdminActor, bool, error) {
	roles := make([]AdminRoleBasic, 0, len(record.roleIDs))
	permissions := make([]string, 0)
	if record.grantAllPermissions {
		permissions = []string{wildcardPermission}
	} else {
		for _, roleID := range record.roleIDs {
			role, ok := r.roles[roleID]
			if !ok || role.deleted {
				continue
			}
			roles = append(roles, AdminRoleBasic{RoleID: role.item.RoleID, Name: role.item.Name})
			permissions = append(permissions, role.item.Permissions...)
		}
		permissions = sortAndUniqueStrings(permissions)
	}
	return AdminActor{UserID: record.userID, Permissions: permissions, Roles: roles, GrantAllPermissions: record.grantAllPermissions}, true, nil
}

func (r *InMemoryRepository) lookupUserLocked(userID string) (UserInfo, bool) {
	if user, ok := r.users[userID]; ok {
		return user, true
	}
	if strings.HasPrefix(userID, "missing") {
		return UserInfo{}, false
	}
	user := UserInfo{UserID: userID, Nickname: userID}
	r.users[userID] = user
	return user, true
}
