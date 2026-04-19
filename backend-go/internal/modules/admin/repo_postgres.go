package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
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
		return AdminActor{}, false, persistence.ErrNotImplemented
	}
	return r.getActorFromDB(ctx, queryable, userID)
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
		return persistence.ErrNotImplemented
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
		return persistence.ErrNotImplemented
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
		return persistence.ErrNotImplemented
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
		return persistence.ErrNotImplemented
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
		return persistence.ErrNotImplemented
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

func (r *PostgresRepository) ListAuditLogs(ctx context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return auditLogPage{}, persistence.ErrNotImplemented
	}
	cursor, apiErr := decodeAuditCursor(filter.Pagination.Cursor)
	if apiErr != nil {
		return auditLogPage{}, apiErr
	}
	rows, err := queryable.Query(ctx, queryAuditList)
	if err != nil {
		return auditLogPage{}, err
	}
	defer rows.Close()
	limit := normalizeLimit(filter.Pagination.Limit)
	page := auditLogPage{Items: make([]AuditLogItem, 0, limit)}
	seenCursor := filter.Pagination.Cursor == ""
	for rows.Next() {
		entry, err := scanAuditEntry(rows)
		if err != nil {
			return auditLogPage{}, err
		}
		if !matchAuditFilter(entry, filter) {
			continue
		}
		if !seenCursor {
			if entry.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if entry.CreatedAt.Equal(cursor.CreatedAt) && entry.LogID >= cursor.LogID {
				continue
			}
			seenCursor = true
		}
		if len(page.Items) >= limit {
			page.Next = &auditCursor{CreatedAt: entry.CreatedAt, LogID: entry.LogID}
			break
		}
		page.Items = append(page.Items, auditLogItemFromEntry(entry, includeIPAddress))
	}
	if err := rows.Err(); err != nil {
		return auditLogPage{}, err
	}
	return page, nil
}

func (r *PostgresRepository) GetAuditLog(ctx context.Context, logID string, includeIPAddress bool) (AuditLogItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AuditLogItem{}, false, persistence.ErrNotImplemented
	}
	entry, found, err := r.getAuditEntryByID(ctx, queryable, logID)
	if err != nil || !found {
		return AuditLogItem{}, found, err
	}
	return auditLogItemFromEntry(entry, includeIPAddress), true, nil
}

func (r *PostgresRepository) AppendAuditLog(ctx context.Context, entry AuditLogEntry) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if entry.LogID == "" {
		entry.LogID = fmt.Sprintf("audit_%d", time.Now().UTC().UnixNano())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	detailsJSON := "{}"
	if len(entry.Details) > 0 {
		blob, err := json.Marshal(entry.Details)
		if err != nil {
			return err
		}
		detailsJSON = string(blob)
	}
	return queryable.Exec(ctx, queryAuditInsert,
		entry.LogID,
		string(entry.Action),
		entry.Operator.UserID,
		entry.Operator.Nickname,
		nullableString(entry.Operator.AvatarURL),
		string(entry.TargetType),
		entry.TargetID,
		nullableString(entry.TargetName),
		detailsJSON,
		nullableString(entry.IPLocation),
		nullableString(entry.IPAddress),
		nullableString(entry.DeviceSummary),
		entry.CreatedAt.UTC(),
	)
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

func (r *PostgresRepository) SearchUsers(ctx context.Context, req AdminUserSearchRequest) (adminUserPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return adminUserPage{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminSearchUsers, req.UserID, req.Nickname, req.LoginID, string(req.Status), string(req.UserType))
	if err != nil {
		return adminUserPage{}, err
	}
	defer rows.Close()
	items := make([]AdminUserItem, 0)
	for rows.Next() {
		item, err := scanAdminUserRow(rows)
		if err != nil {
			return adminUserPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminUserPage{}, err
	}
	cursorRaw := paginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminUserCursor(cursorRaw)
	if apiErr != nil {
		return adminUserPage{}, apiErr
	}
	filtered := make([]AdminUserItem, 0, len(items))
	for _, item := range items {
		if cursorRaw != "" {
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
	page := adminUserPage{Items: make([]AdminUserItem, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			page.Next = &adminUserCursor{CreatedAt: timeFromRFC3339(item.CreatedAt), UserID: item.User.UserID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (AdminUserItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminUserItem{}, false, persistence.ErrNotImplemented
	}
	item, err := scanAdminUserRow(queryable.QueryRow(ctx, queryAdminGetUser, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminUserItem{}, false, nil
		}
		return AdminUserItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) GetUserBanState(ctx context.Context, userID string) (UserBanState, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return UserBanState{}, false, persistence.ErrNotImplemented
	}
	var state UserBanState
	var bannedAt *time.Time
	if err := queryable.QueryRow(ctx, queryAdminGetUserBanState, userID).Scan(&state.Status, &state.Reason, &state.DurationDays, &bannedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserBanState{}, false, nil
		}
		return UserBanState{}, false, err
	}
	if bannedAt != nil {
		state.BannedAt = *bannedAt
	}
	return state, true, nil
}

func (r *PostgresRepository) SetUserBanState(ctx context.Context, userID string, state UserBanState) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	status := state.Status
	if status == "" {
		status = UserAccountStatusNormal
	}
	var reason any
	var duration any
	var bannedAt any
	if status == UserAccountStatusBanned {
		reason = nullableString(strings.TrimSpace(state.Reason))
		duration = state.DurationDays
		if state.BannedAt.IsZero() {
			bannedAt = time.Now().UTC()
		} else {
			bannedAt = state.BannedAt
		}
	} else {
		reason = nil
		duration = nil
		bannedAt = nil
	}
	return queryable.Exec(ctx, queryAdminSetUserBanState, userID, string(status), reason, duration, bannedAt)
}

func (r *PostgresRepository) ListUserSessions(ctx context.Context, userID string) ([]string, error) {
	queryable, ok := r.queryable()
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminListUserSessions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, err
		}
		items = append(items, sessionID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, sessionID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryAdminRevokeSession, sessionID)
}

func (r *PostgresRepository) SearchGroups(ctx context.Context, req AdminGroupSearchRequest) (adminGroupPage, error) {
	queryable, ok := r.queryable()
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
	cursorRaw := paginationCursor(req.Pagination)
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
	limit := normalizeLimit(paginationLimit(req.Pagination))
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
	queryable, ok := r.queryable()
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
	queryable, ok := r.queryable()
	if !ok {
		return AdminGroupItem{}, false, persistence.ErrNotImplemented
	}
	item, found, err := r.GetGroup(ctx, groupID)
	if err != nil || !found {
		return AdminGroupItem{}, found, err
	}
	if err := queryable.Exec(ctx, queryAdminDismissGroup, groupID); err != nil {
		return AdminGroupItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) ListGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	queryable, ok := r.queryable()
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

func (r *PostgresRepository) SearchReports(ctx context.Context, req AdminReportSearchRequest) (adminReportPage, error) {
	queryable, ok := r.queryable()
	if !ok {
		return adminReportPage{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryAdminSearchReports, string(req.Status), string(req.TargetType))
	if err != nil {
		return adminReportPage{}, err
	}
	defer rows.Close()
	items := make([]AdminReportItem, 0)
	for rows.Next() {
		record, err := scanAdminReportRow(rows, false)
		if err != nil {
			return adminReportPage{}, err
		}
		items = append(items, record.Item)
	}
	if err := rows.Err(); err != nil {
		return adminReportPage{}, err
	}
	cursorRaw := paginationCursor(req.Pagination)
	cursor, apiErr := decodeAdminReportCursor(cursorRaw)
	if apiErr != nil {
		return adminReportPage{}, apiErr
	}
	filtered := make([]AdminReportItem, 0, len(items))
	for _, item := range items {
		if cursorRaw != "" {
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
	page := adminReportPage{Items: make([]AdminReportItem, 0, limit)}
	for idx, item := range filtered {
		if idx >= limit {
			page.Next = &adminReportCursor{CreatedAt: timeFromRFC3339(item.CreatedAt), ReportID: item.ReportID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *PostgresRepository) GetReportHandleRecord(ctx context.Context, reportID string) (reportHandleRecord, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return reportHandleRecord{}, false, persistence.ErrNotImplemented
	}
	record, err := scanAdminReportRow(queryable.QueryRow(ctx, queryAdminGetReportHandleRecord, reportID), true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reportHandleRecord{}, false, nil
		}
		return reportHandleRecord{}, false, err
	}
	return record, true, nil
}

func (r *PostgresRepository) HandleReport(ctx context.Context, reportID string, action ReportHandleAction, note string, handledBy *UserInfo) (AdminReportItem, bool, error) {
	queryable, ok := r.queryable()
	if !ok {
		return AdminReportItem{}, false, persistence.ErrNotImplemented
	}
	record, found, err := r.GetReportHandleRecord(ctx, reportID)
	if err != nil || !found {
		return AdminReportItem{}, found, err
	}
	status := ReportStatusReviewing
	switch action {
	case ReportHandleResolve:
		status = ReportStatusResolved
	case ReportHandleDismiss:
		status = ReportStatusDismissed
	case ReportHandleEscalate:
		status = ReportStatusReviewing
	}
	now := time.Now().UTC()
	var handledByID any
	if handledBy != nil {
		handledByID = handledBy.UserID
	}
	if err := queryable.Exec(ctx, queryAdminHandleReport, reportID, string(status), now, handledByID, nullableString(strings.TrimSpace(note))); err != nil {
		return AdminReportItem{}, false, err
	}
	record.Item.Status = status
	record.Item.HandledAt = now.Format(time.RFC3339)
	record.Item.HandleNote = strings.TrimSpace(note)
	if handledBy != nil {
		hb := *handledBy
		record.Item.HandledBy = &hb
	}
	return record.Item, true, nil
}

func (r *PostgresRepository) RecallGroupMessage(ctx context.Context, messageID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, "UPDATE group_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE", messageID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) RecallPrivateMessage(ctx context.Context, messageID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, "UPDATE private_messages SET is_recalled = TRUE, recalled_at = NOW() WHERE message_id = $1 AND is_recalled = FALSE", messageID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) DeleteMoment(ctx context.Context, momentID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, "DELETE FROM moments WHERE moment_id = $1", momentID)
}

func (r *PostgresRepository) DeleteComment(ctx context.Context, commentID string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, "DELETE FROM comments WHERE comment_id = $1", commentID)
}

func (r *PostgresRepository) IsBanned(ctx context.Context, userID string) (bool, error) {
	state, found, err := r.GetUserBanState(ctx, userID)
	if err != nil || !found {
		return false, err
	}
	return state.Status == UserAccountStatusBanned, nil
}
