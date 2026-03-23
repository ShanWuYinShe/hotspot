package admin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

type roleRecord struct {
	item      AdminRoleItem
	deleted   bool
	updatedAt time.Time
}

type adminGrantRecord struct {
	userID              string
	grantAllPermissions bool
	roleIDs             []string
	grantedAt           time.Time
	revoked             bool
}

type InMemoryRepository struct {
	mu             sync.RWMutex
	roles          map[string]roleRecord
	admins         map[string]adminGrantRecord
	users          map[string]UserInfo
	auditLogs      map[string]AuditLogEntry
	auditOrder     []string
	deletedRoleIDs map[string]struct{}
	revokedAdmins  map[string]struct{}
}

func NewInMemoryRepository() *InMemoryRepository {
	repo := &InMemoryRepository{
		roles:  make(map[string]roleRecord),
		admins: make(map[string]adminGrantRecord),
		users: map[string]UserInfo{
			"admin":      {UserID: "admin", Nickname: "admin"},
			"user_alice": {UserID: "user_alice", Nickname: "Alice"},
			"user_bob":   {UserID: "user_bob", Nickname: "Bob"},
			"user_cindy": {UserID: "user_cindy", Nickname: "Cindy"},
		},
		auditLogs:      make(map[string]AuditLogEntry),
		deletedRoleIDs: make(map[string]struct{}),
		revokedAdmins:  make(map[string]struct{}),
	}
	repo.seedAuditLogs()
	return repo
}

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
			return AdminRoleItem{}, &httpx.APIError{Code: 10400, Message: "角色已存在"}
		}
	}
	now := time.Now().UTC()
	item := AdminRoleItem{
		RoleID:      fmt.Sprintf("role_%d", now.UnixNano()),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
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
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
	}
	if req.Name != nil {
		name := normalizeRoleName(*req.Name)
		for _, other := range r.roles {
			if other.deleted || other.item.RoleID == req.RoleID {
				continue
			}
			if strings.EqualFold(other.item.Name, name) {
				return &httpx.APIError{Code: 10400, Message: "角色已存在"}
			}
		}
		record.item.Name = name
	}
	if req.ClearDescription {
		record.item.Description = ""
	} else if req.Description != nil {
		record.item.Description = strings.TrimSpace(*req.Description)
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
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
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
		return &httpx.APIError{Code: 10300, Message: "角色不存在"}
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
	limit := normalizeLimit(pagination.Limit)
	page := adminListPage{Items: make([]AdminOperatorItem, 0, limit)}
	for idx, grant := range filtered {
		if idx >= limit {
			page.Next = &adminCursor{GrantedAt: grant.grantedAt, UserID: grant.userID}
			break
		}
		actor, _, _ := r.actorFromRecordLocked(grant)
		user, _ := r.lookupUserLocked(grant.userID)
		page.Items = append(page.Items, AdminOperatorItem{User: user, Permissions: cloneStrings(actor.Permissions), Roles: append([]AdminRoleBasic(nil), actor.Roles...)})
	}
	return page, nil
}

func (r *InMemoryRepository) GrantAdmin(_ context.Context, grant AdminGrantUpsert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookupUserLocked(grant.UserID); !ok {
		return &httpx.APIError{Code: 10300, Message: "用户不存在"}
	}
	for _, roleID := range grant.RoleIDs {
		record, ok := r.roles[roleID]
		if !ok || record.deleted {
			return &httpx.APIError{Code: 10300, Message: "角色不存在"}
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
		return &httpx.APIError{Code: 10300, Message: "管理员不存在"}
	}
	if record.revoked {
		return nil
	}
	record.revoked = true
	r.admins[userID] = record
	r.revokedAdmins[userID] = struct{}{}
	return nil
}

func (r *InMemoryRepository) ListAuditLogs(_ context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAuditCursor(filter.Pagination.Cursor)
	if apiErr != nil {
		return auditLogPage{}, apiErr
	}
	logs := r.filteredAuditLogsLocked(filter)
	limit := normalizeLimit(filter.Pagination.Limit)
	page := auditLogPage{Items: make([]AuditLogItem, 0, limit)}
	seenCursor := filter.Pagination.Cursor == ""
	for _, entry := range logs {
		if !seenCursor {
			if entry.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if entry.CreatedAt.Equal(cursor.CreatedAt) && entry.LogID >= cursor.LogID {
				continue
			}
			seenCursor = true
		}
		item := auditLogItemFromEntry(entry, includeIPAddress)
		if len(page.Items) >= limit {
			page.Next = &auditCursor{CreatedAt: entry.CreatedAt, LogID: entry.LogID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *InMemoryRepository) GetAuditLog(_ context.Context, logID string, includeIPAddress bool) (AuditLogItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.auditLogs[logID]
	if !ok {
		return AuditLogItem{}, false, nil
	}
	return auditLogItemFromEntry(entry, includeIPAddress), true, nil
}

func (r *InMemoryRepository) AppendAuditLog(_ context.Context, entry AuditLogEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry.LogID == "" {
		entry.LogID = fmt.Sprintf("audit_%d", time.Now().UTC().UnixNano())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	r.auditLogs[entry.LogID] = cloneAuditEntry(entry)
	r.auditOrder = append(r.auditOrder, entry.LogID)
	sort.SliceStable(r.auditOrder, func(i, j int) bool {
		left := r.auditLogs[r.auditOrder[i]]
		right := r.auditLogs[r.auditOrder[j]]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.LogID > right.LogID
		}
		return left.CreatedAt.After(right.CreatedAt)
	})
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

func (r *InMemoryRepository) filteredAuditLogsLocked(filter AuditLogFilter) []AuditLogEntry {
	items := make([]AuditLogEntry, 0, len(r.auditOrder))
	for _, logID := range r.auditOrder {
		entry := r.auditLogs[logID]
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.OperatorID != "" && entry.Operator.UserID != filter.OperatorID {
			continue
		}
		if filter.TargetType != "" && entry.TargetType != filter.TargetType {
			continue
		}
		if filter.TargetID != "" && entry.TargetID != filter.TargetID {
			continue
		}
		if filter.StartTime != nil && entry.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && entry.CreatedAt.After(*filter.EndTime) {
			continue
		}
		items = append(items, cloneAuditEntry(entry))
	}
	return items
}

func auditLogItemFromEntry(entry AuditLogEntry, includeIPAddress bool) AuditLogItem {
	item := AuditLogItem{
		LogID:         entry.LogID,
		Action:        entry.Action,
		Operator:      entry.Operator,
		TargetType:    entry.TargetType,
		TargetID:      entry.TargetID,
		TargetName:    entry.TargetName,
		Details:       cloneAuditDetails(entry.Details),
		IPLocation:    entry.IPLocation,
		DeviceSummary: entry.DeviceSummary,
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	if includeIPAddress && entry.IPAddress != "" {
		ip := entry.IPAddress
		item.IPAddress = &ip
	}
	return item
}

func cloneAuditEntry(entry AuditLogEntry) AuditLogEntry {
	entry.Details = cloneAuditDetails(entry.Details)
	return entry
}

func cloneAuditDetails(details AuditDetails) AuditDetails {
	if len(details) == 0 {
		return nil
	}
	cloned := make(AuditDetails, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}

func (r *InMemoryRepository) seedAuditLogs() {
	now := time.Now().UTC()
	logs := []AuditLogEntry{
		{
			LogID:      "audit_seed_002",
			Action:     AuditActionConfigUpdate,
			Operator:   UserInfo{UserID: "admin", Nickname: "admin"},
			TargetType: AuditTargetTypeConfig,
			TargetID:   "response_defaults.chat_recent_messages_count",
			TargetName: "chat_recent_messages_count",
			Details: AuditDetails{
				"type":       string(AuditActionConfigUpdate),
				"config_key": "response_defaults.chat_recent_messages_count",
				"old_value":  20,
				"new_value":  30,
			},
			IPLocation:    "CN-ZJ-Hangzhou",
			IPAddress:     "203.0.113.10",
			DeviceSummary: "Chrome on macOS",
			CreatedAt:     now.Add(-1 * time.Hour),
		},
		{
			LogID:      "audit_seed_001",
			Action:     AuditActionAdminGrant,
			Operator:   UserInfo{UserID: "admin", Nickname: "admin"},
			TargetType: AuditTargetTypeAdmin,
			TargetID:   "user_alice",
			TargetName: "Alice",
			Details: AuditDetails{
				"type":            string(AuditActionAdminGrant),
				"target_user_id":  "user_alice",
				"role_ids":        []string{"role_ops"},
				"grant_all_permissions": false,
			},
			IPLocation:    "CN-BJ-Beijing",
			IPAddress:     "198.51.100.20",
			DeviceSummary: "Safari on iPhone",
			CreatedAt:     now.Add(-2 * time.Hour),
		},
	}
	for _, log := range logs {
		r.auditLogs[log.LogID] = cloneAuditEntry(log)
		r.auditOrder = append(r.auditOrder, log.LogID)
	}
}
