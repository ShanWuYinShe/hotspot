package admin

import (
	"context"
	"fmt"
	groupsmodule "hotspot/backend-go/internal/modules/groups"
	usersmodule "hotspot/backend-go/internal/modules/users"
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
	mu           sync.RWMutex
	roles        map[string]roleRecord
	admins       map[string]adminGrantRecord
	users        map[string]UserInfo
	adminUsers   map[string]AdminUserItem
	groups       map[string]AdminGroupItem
	groupMembers map[string][]string
	reports      map[string]reportHandleRecord
	// seq 保证同纳秒内连续创建的 ID 不碰撞（UnixNano 时间戳在快速循环下会重复），
	// 同时服务 role 与 audit ID 生成。
	seq             uint64
	banStates       map[string]UserBanState
	userSessions    map[string][]string
	auditLogs       map[string]AuditLogEntry
	auditOrder      []string
	deletedRoleIDs  map[string]struct{}
	revokedAdmins   map[string]struct{}
	moments         map[string]momentRecord
	comments        map[string]commentRecord
	groupMessages   map[string]messageRecord
	privateMessages map[string]messageRecord
}

type momentRecord struct {
	ID        string
	Content   string
	AuthorID  string
	CreatedAt time.Time
	Deleted   bool
}

type commentRecord struct {
	ID        string
	Content   string
	AuthorID  string
	MomentID  string
	CreatedAt time.Time
	Deleted   bool
}

type messageRecord struct {
	ID        string
	Content   string
	SenderID  string
	CreatedAt time.Time
	Recalled  bool
}

func NewInMemoryRepository() *InMemoryRepository {
	now := time.Now().UTC()
	aliceCreatedAt := now.Add(-72 * time.Hour).Format(time.RFC3339)
	bobCreatedAt := now.Add(-48 * time.Hour).Format(time.RFC3339)
	cindyCreatedAt := now.Add(-24 * time.Hour).Format(time.RFC3339)
	adminCreatedAt := now.Add(-96 * time.Hour).Format(time.RFC3339)

	adminUsers := map[string]AdminUserItem{
		"admin": {
			User:              UserInfo{UserID: "admin", Nickname: "admin"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         adminCreatedAt,
			LastActiveAt:      now.Add(-time.Hour).Format(time.RFC3339),
			LoginID:           "admin",
			PresenceStatus:    usersmodule.UserPresenceOnline,
			LastOnline:        now.Add(-time.Hour).Format(time.RFC3339),
			RegisterIP:        "127.0.0.1",
			RegisterLocation:  "Local",
			LastLoginIP:       "127.0.0.1",
			LastLoginLocation: "Local",
		},
		"user_alice": {
			User:              UserInfo{UserID: "user_alice", Nickname: "Alice"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         aliceCreatedAt,
			LastActiveAt:      now.Add(-2 * time.Hour).Format(time.RFC3339),
			LoginID:           "alice",
			PresenceStatus:    usersmodule.UserPresenceOnline,
			LastOnline:        now.Add(-2 * time.Hour).Format(time.RFC3339),
			RegisterIP:        "10.0.0.2",
			RegisterLocation:  "Shanghai",
			LastLoginIP:       "10.0.0.3",
			LastLoginLocation: "Shanghai",
		},
		"user_bob": {
			User:              UserInfo{UserID: "user_bob", Nickname: "Bob"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         bobCreatedAt,
			LastActiveAt:      now.Add(-3 * time.Hour).Format(time.RFC3339),
			LoginID:           "bob",
			PresenceStatus:    usersmodule.UserPresenceOffline,
			LastOnline:        now.Add(-3 * time.Hour).Format(time.RFC3339),
			RegisterIP:        "10.0.0.4",
			RegisterLocation:  "Hangzhou",
			LastLoginIP:       "10.0.0.4",
			LastLoginLocation: "Hangzhou",
		},
		"user_cindy": {
			User:              UserInfo{UserID: "user_cindy", Nickname: "Cindy"},
			Status:            UserAccountStatusNormal,
			CreatedAt:         cindyCreatedAt,
			LastActiveAt:      now.Add(-4 * time.Hour).Format(time.RFC3339),
			LoginID:           "cindy",
			PresenceStatus:    usersmodule.UserPresenceOffline,
			LastOnline:        now.Add(-4 * time.Hour).Format(time.RFC3339),
			RegisterIP:        "10.0.0.5",
			RegisterLocation:  "Beijing",
			LastLoginIP:       "10.0.0.5",
			LastLoginLocation: "Beijing",
		},
	}

	groups := map[string]AdminGroupItem{
		"group_hotspot": {
			Group: groupsmodule.GroupInfo{GroupID: "group_hotspot", Name: "HotSpot", MembersCount: 2},
			Location: groupsmodule.Location{Point: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{Latitude: 31.2304, Longitude: 121.4737}, LocationName: "Shanghai"},
			CreatedAt:      now.Add(-36 * time.Hour).Format(time.RFC3339),
			Owner:          UserInfo{UserID: "user_alice", Nickname: "Alice"},
			JoinMethod:     groupsmodule.GroupJoinMethod("direct"),
			IsSearchable:   true,
			MaxMembers:     200,
			LastActivityAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
		},
	}

	reports := map[string]reportHandleRecord{
		"report_1": {
			Item: AdminReportItem{
				ReportID:   "report_1",
				TargetType: ReportTargetTypeUser,
				TargetID:   "user_bob",
				Reason:     "spam",
				Status:     ReportStatusPending,
				CreatedAt:  now.Add(-6 * time.Hour).Format(time.RFC3339),
				Reporter:   UserInfo{UserID: "user_alice", Nickname: "Alice"},
			},
			TargetUser: UserInfo{UserID: "user_bob", Nickname: "Bob"},
		},
	}

	users := map[string]UserInfo{}
	for userID, item := range adminUsers {
		users[userID] = item.User
	}

	repo := &InMemoryRepository{
		roles:           make(map[string]roleRecord),
		admins:          make(map[string]adminGrantRecord),
		users:           users,
		adminUsers:      adminUsers,
		groups:          groups,
		groupMembers:    map[string][]string{"group_hotspot": {"user_alice", "user_bob"}},
		reports:         reports,
		banStates:       map[string]UserBanState{"admin": {Status: UserAccountStatusNormal}, "user_alice": {Status: UserAccountStatusNormal}, "user_bob": {Status: UserAccountStatusNormal}, "user_cindy": {Status: UserAccountStatusNormal}},
		userSessions:    map[string][]string{"user_alice": {"sess_user_alice_1"}, "user_bob": {"sess_user_bob_1"}},
		auditLogs:       make(map[string]AuditLogEntry),
		deletedRoleIDs:  make(map[string]struct{}),
		revokedAdmins:   make(map[string]struct{}),
		moments:         make(map[string]momentRecord),
		comments:        make(map[string]commentRecord),
		groupMessages:   make(map[string]messageRecord),
		privateMessages: make(map[string]messageRecord),
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
			return AdminRoleItem{}, &httpx.APIError{Code: httpx.CodeConflict, Message: "角色已存在"}
		}
	}
	now := time.Now().UTC()
	r.seq++
	item := AdminRoleItem{
		RoleID:      fmt.Sprintf("role_%d_%d", now.UnixNano(), r.seq),
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
		r.seq++
		entry.LogID = fmt.Sprintf("audit_%d_%d", time.Now().UTC().UnixNano(), r.seq)
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

func paginationLimit(pagination *PaginationParams) int {
	if pagination == nil {
		return 0
	}
	return pagination.Limit
}

func paginationCursor(pagination *PaginationParams) string {
	if pagination == nil {
		return ""
	}
	return pagination.Cursor
}

func timeFromRFC3339(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func cloneAdminUserItem(item AdminUserItem) AdminUserItem {
	cloned := item
	cloned.Tags = cloneStrings(item.Tags)
	if item.Location != nil {
		loc := *item.Location
		cloned.Location = &loc
	}
	if item.Stats != nil {
		stats := *item.Stats
		cloned.Stats = &stats
	}
	return cloned
}

func cloneAdminGroupItem(item AdminGroupItem) AdminGroupItem {
	cloned := item
	cloned.Tags = cloneStrings(item.Tags)
	if item.MyState != nil {
		state := *item.MyState
		cloned.MyState = &state
	}
	return cloned
}

func cloneAdminReportItem(item AdminReportItem) AdminReportItem {
	cloned := item
	if item.HandledBy != nil {
		handledBy := *item.HandledBy
		cloned.HandledBy = &handledBy
	}
	return cloned
}

func cloneReportHandleRecord(record reportHandleRecord) reportHandleRecord {
	cloned := record
	cloned.Item = cloneAdminReportItem(record.Item)
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
				"type":                  string(AuditActionAdminGrant),
				"target_user_id":        "user_alice",
				"role_ids":              []string{"role_ops"},
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

func (r *InMemoryRepository) SearchUsers(_ context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminUserCursor(paginationCursor(req.Pagination))
	if apiErr != nil {
		return adminUserPage{}, apiErr
	}
	items := make([]AdminUserItem, 0, len(r.adminUsers))
	for _, item := range r.adminUsers {
		if req.UserID != "" && item.User.UserID != req.UserID {
			continue
		}
		if req.Nickname != "" && !strings.Contains(strings.ToLower(item.User.Nickname), strings.ToLower(req.Nickname)) {
			continue
		}
		if req.LoginID != "" && !strings.Contains(strings.ToLower(item.LoginID), strings.ToLower(req.LoginID)) {
			continue
		}
		if req.Status != "" && item.Status != req.Status {
			continue
		}
		if req.UserType != "" && item.User.UserID != "admin" && req.UserType != usersmodule.UserTypeNormal {
			continue
		}
		items = append(items, cloneAdminUserItem(item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].User.UserID < items[j].User.UserID
		}
		return it.After(jt)
	})
	filtered := make([]AdminUserItem, 0, len(items))
	for _, item := range items {
		if paginationCursor(req.Pagination) != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.User.UserID <= cursor.UserID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := normalizeLimit(paginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminUserPage{Items: append(make([]AdminUserItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminUserCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), UserID: last.User.UserID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetUser(_ context.Context, userID string) (AdminUserItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.adminUsers[userID]
	if !ok {
		return AdminUserItem{}, false, nil
	}
	return cloneAdminUserItem(item), true, nil
}

func (r *InMemoryRepository) GetUserBanState(_ context.Context, userID string) (UserBanState, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.adminUsers[userID]; !ok {
		return UserBanState{}, false, nil
	}
	state, ok := r.banStates[userID]
	if !ok {
		return UserBanState{Status: UserAccountStatusNormal}, true, nil
	}
	return state, true, nil
}

func (r *InMemoryRepository) SetUserBanState(_ context.Context, userID string, state UserBanState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.adminUsers[userID]
	if !ok {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}
	}
	if state.Status == "" {
		state.Status = UserAccountStatusNormal
	}
	if state.Status != UserAccountStatusBanned {
		state = UserBanState{Status: UserAccountStatusNormal}
	}
	r.banStates[userID] = state
	item.Status = state.Status
	r.adminUsers[userID] = item
	return nil
}

func (r *InMemoryRepository) ListUserSessions(_ context.Context, userID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneStrings(r.userSessions[userID]), nil
}

func (r *InMemoryRepository) RevokeSession(_ context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for userID, sessions := range r.userSessions {
		filtered := make([]string, 0, len(sessions))
		for _, candidate := range sessions {
			if candidate != sessionID {
				filtered = append(filtered, candidate)
			}
		}
		if len(filtered) == 0 {
			delete(r.userSessions, userID)
			continue
		}
		r.userSessions[userID] = filtered
	}
	return nil
}

func (r *InMemoryRepository) SearchGroups(_ context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminGroupCursor(paginationCursor(req.Pagination))
	if apiErr != nil {
		return adminGroupPage{}, apiErr
	}
	items := make([]AdminGroupItem, 0, len(r.groups))
	for _, item := range r.groups {
		if req.GroupID != "" && item.Group.GroupID != req.GroupID {
			continue
		}
		if req.Name != "" && !strings.Contains(strings.ToLower(item.Group.Name), strings.ToLower(req.Name)) {
			continue
		}
		if req.OwnerID != "" && item.Owner.UserID != req.OwnerID {
			continue
		}
		items = append(items, cloneAdminGroupItem(item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].Group.GroupID < items[j].Group.GroupID
		}
		return it.After(jt)
	})
	filtered := make([]AdminGroupItem, 0, len(items))
	for _, item := range items {
		if paginationCursor(req.Pagination) != "" {
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
	limit := normalizeLimit(paginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminGroupPage{Items: append(make([]AdminGroupItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminGroupCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), GroupID: last.Group.GroupID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetGroup(_ context.Context, groupID string) (AdminGroupItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.groups[groupID]
	if !ok {
		return AdminGroupItem{}, false, nil
	}
	return cloneAdminGroupItem(item), true, nil
}

func (r *InMemoryRepository) DismissGroup(_ context.Context, groupID string) (AdminGroupItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.groups[groupID]
	if !ok {
		return AdminGroupItem{}, false, nil
	}
	delete(r.groups, groupID)
	delete(r.groupMembers, groupID)
	return cloneAdminGroupItem(item), true, nil
}

func (r *InMemoryRepository) ListGroupMemberIDs(_ context.Context, groupID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneStrings(r.groupMembers[groupID]), nil
}

func (r *InMemoryRepository) SearchReports(_ context.Context, req AdminReportListRequest) (adminReportPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminReportCursor(paginationCursor(req.Pagination))
	if apiErr != nil {
		return adminReportPage{}, apiErr
	}
	items := make([]AdminReportItem, 0, len(r.reports))
	for _, record := range r.reports {
		if req.Status != "" && record.Item.Status != req.Status {
			continue
		}
		if req.TargetType != "" && record.Item.TargetType != req.TargetType {
			continue
		}
		items = append(items, cloneAdminReportItem(record.Item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].ReportID > items[j].ReportID
		}
		return it.After(jt)
	})
	filtered := make([]AdminReportItem, 0, len(items))
	for _, item := range items {
		if paginationCursor(req.Pagination) != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.ReportID >= cursor.ReportID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := normalizeLimit(paginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminReportPage{Items: append(make([]AdminReportItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminReportCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), ReportID: last.ReportID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetReportHandleRecord(_ context.Context, reportID string) (reportHandleRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.reports[reportID]
	if !ok {
		return reportHandleRecord{}, false, nil
	}
	return cloneReportHandleRecord(record), true, nil
}

func (r *InMemoryRepository) HandleReport(_ context.Context, reportID string, action ReportAction, note string, handledBy *UserInfo) (AdminReportItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.reports[reportID]
	if !ok {
		return AdminReportItem{}, false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	switch action {
	case ReportActionResolve:
		record.Item.Status = ReportStatusResolved
	case ReportActionDismiss:
		record.Item.Status = ReportStatusDismissed
	}
	record.Item.HandledAt = now
	record.Item.HandleNote = note
	if handledBy != nil {
		h := *handledBy
		record.Item.HandledBy = &h
	}
	r.reports[reportID] = record
	return cloneAdminReportItem(record.Item), true, nil
}

func (r *InMemoryRepository) RecallGroupMessage(_ context.Context, messageID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg, ok := r.groupMessages[messageID]
	if !ok {
		return nil
	}
	msg.Recalled = true
	r.groupMessages[messageID] = msg
	return nil
}

func (r *InMemoryRepository) RecallPrivateMessage(_ context.Context, messageID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg, ok := r.privateMessages[messageID]
	if !ok {
		return nil
	}
	msg.Recalled = true
	r.privateMessages[messageID] = msg
	return nil
}

func (r *InMemoryRepository) DeleteMoment(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.moments, momentID)
	return nil
}

func (r *InMemoryRepository) DeleteComment(_ context.Context, commentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.comments, commentID)
	return nil
}

func (r *InMemoryRepository) CreateMoment(_ context.Context, id, content, authorID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.moments[id] = momentRecord{
		ID:        id,
		Content:   content,
		AuthorID:  authorID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) CreateComment(_ context.Context, id, content, authorID, momentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comments[id] = commentRecord{
		ID:        id,
		Content:   content,
		AuthorID:  authorID,
		MomentID:  momentID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) SendGroupMessage(_ context.Context, id, content, senderID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groupMessages[id] = messageRecord{
		ID:        id,
		Content:   content,
		SenderID:  senderID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) HasMoment(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.moments[id]
	return ok
}

func (r *InMemoryRepository) HasComment(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.comments[id]
	return ok
}

func (r *InMemoryRepository) IsGroupMessageRecalled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	msg, ok := r.groupMessages[id]
	if !ok {
		return false
	}
	return msg.Recalled
}

func (r *InMemoryRepository) IsBanned(_ context.Context, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.banStates[userID]
	if !ok {
		return false, nil
	}
	return state.Status == UserAccountStatusBanned, nil
}
