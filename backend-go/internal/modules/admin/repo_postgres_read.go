package admin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"hotspot/backend-go/internal/platform/persistence"
)

type postgresRoleRecord struct {
	item      AdminRoleItem
	createdAt time.Time
	updatedAt time.Time
	deleted   bool
}

type adminListRow struct {
	User                UserInfo
	GrantAllPermissions bool
	GrantedAt           time.Time
	RoleID              string
	RoleName            string
	Permissions         []string
}

type adminListRecord struct {
	User                UserInfo
	GrantAllPermissions bool
	GrantedAt           time.Time
	Roles               []AdminRoleBasic
	Permissions         []string
}

func (r adminListRecord) rolesForResponse() []AdminRoleBasic {
	return append([]AdminRoleBasic(nil), r.Roles...)
}

func (r adminListRecord) permissionsForResponse() []string {
	if r.GrantAllPermissions {
		return []string{wildcardPermission}
	}
	return cloneStrings(r.Permissions)
}

func (r *PostgresRepository) getRoleByID(ctx context.Context, queryable persistence.QueryableDB, roleID string) (postgresRoleRecord, bool, error) {
	var record postgresRoleRecord
	var description string
	if err := queryable.QueryRow(ctx, queryRoleByID, roleID).Scan(&record.item.RoleID, &record.item.Name, &description, &record.createdAt, &record.updatedAt, &record.deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return postgresRoleRecord{}, false, nil
		}
		return postgresRoleRecord{}, false, err
	}
	record.item.Description = description
	permissions, err := r.getRolePermissions(ctx, queryable, roleID)
	if err != nil {
		return postgresRoleRecord{}, false, err
	}
	record.item.Permissions = permissions
	return record, true, nil
}

func (r *PostgresRepository) getRoleByName(ctx context.Context, queryable persistence.QueryableDB, name string) (postgresRoleRecord, bool, error) {
	var record postgresRoleRecord
	var description string
	if err := queryable.QueryRow(ctx, queryRoleByName, name).Scan(&record.item.RoleID, &record.item.Name, &description, &record.createdAt, &record.updatedAt, &record.deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return postgresRoleRecord{}, false, nil
		}
		return postgresRoleRecord{}, false, err
	}
	record.item.Description = description
	permissions, err := r.getRolePermissions(ctx, queryable, record.item.RoleID)
	if err != nil {
		return postgresRoleRecord{}, false, err
	}
	record.item.Permissions = permissions
	return record, true, nil
}

func (r *PostgresRepository) getRolePermissions(ctx context.Context, queryable persistence.QueryableDB, roleID string) ([]string, error) {
	rows, err := queryable.Query(ctx, queryRolePermissions, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sortAndUniqueStrings(permissions), nil
}

func (r *PostgresRepository) lookupUser(ctx context.Context, queryable persistence.QueryableDB, userID string) (UserInfo, bool, error) {
	var user UserInfo
	if err := queryable.QueryRow(ctx, queryUserExists, userID).Scan(&user.UserID, &user.Nickname, &user.AvatarURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, false, nil
		}
		return UserInfo{}, false, err
	}
	return user, true, nil
}

func (r *PostgresRepository) getActorFromDB(ctx context.Context, queryable persistence.QueryableDB, userID string) (AdminActor, bool, error) {
	record, found, err := r.getAdminRecordByUserID(ctx, queryable, userID)
	if err != nil || !found {
		return AdminActor{}, found, err
	}
	return AdminActor{
		UserID:              record.User.UserID,
		Permissions:         record.permissionsForResponse(),
		Roles:               record.rolesForResponse(),
		GrantAllPermissions: record.GrantAllPermissions,
	}, true, nil
}

func (r *PostgresRepository) getAdminRecordByUserID(ctx context.Context, queryable persistence.QueryableDB, userID string) (adminListRecord, bool, error) {
	records, err := r.listAdminRecords(ctx, queryable, queryAdminActorDetail, userID)
	if err != nil {
		return adminListRecord{}, false, err
	}
	if len(records) == 0 {
		return adminListRecord{}, false, nil
	}
	return records[0], true, nil
}

func (r *PostgresRepository) listAdminsFromDB(ctx context.Context, queryable persistence.QueryableDB) ([]adminListRecord, error) {
	return r.listAdminRecords(ctx, queryable, queryAdminList)
}

func (r *PostgresRepository) listAdminRecords(ctx context.Context, queryable persistence.QueryableDB, query string, args ...any) ([]adminListRecord, error) {
	rows, err := queryable.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]adminListRecord, 0)
	indexByUserID := make(map[string]int)
	for rows.Next() {
		var row adminListRow
		if err := rows.Scan(&row.User.UserID, &row.User.Nickname, &row.User.AvatarURL, &row.GrantAllPermissions, &row.GrantedAt, &row.RoleID, &row.RoleName, &row.Permissions); err != nil {
			return nil, err
		}
		row.Permissions = sortAndUniqueStrings(row.Permissions)
		idx, ok := indexByUserID[row.User.UserID]
		if !ok {
			items = append(items, adminListRecord{
				User:                row.User,
				GrantAllPermissions: row.GrantAllPermissions,
				GrantedAt:           row.GrantedAt,
				Roles:               make([]AdminRoleBasic, 0),
				Permissions:         make([]string, 0),
			})
			idx = len(items) - 1
			indexByUserID[row.User.UserID] = idx
		}
		if row.RoleID != "" {
			items[idx].Roles = append(items[idx].Roles, AdminRoleBasic{RoleID: row.RoleID, Name: row.RoleName})
			items[idx].Permissions = append(items[idx].Permissions, row.Permissions...)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Permissions = sortAndUniqueStrings(items[i].Permissions)
		if items[i].GrantAllPermissions {
			items[i].Permissions = []string{wildcardPermission}
		}
	}
	return items, nil
}

func getAuditEntryByRow(row persistence.Row) (AuditLogEntry, error) {
	var (
		entry       AuditLogEntry
		action      string
		targetType  string
		detailsJSON string
	)
	if err := row.Scan(&entry.LogID, &action, &entry.Operator.UserID, &entry.Operator.Nickname, &entry.Operator.AvatarURL, &targetType, &entry.TargetID, &entry.TargetName, &detailsJSON, &entry.IPLocation, &entry.IPAddress, &entry.DeviceSummary, &entry.CreatedAt); err != nil {
		return AuditLogEntry{}, err
	}
	entry.Action = AuditAction(action)
	entry.TargetType = AuditTargetType(targetType)
	if err := json.Unmarshal([]byte(detailsJSON), &entry.Details); err != nil {
		return AuditLogEntry{}, err
	}
	return entry, nil
}

func (r *PostgresRepository) getAuditEntryByID(ctx context.Context, queryable persistence.QueryableDB, logID string) (AuditLogEntry, bool, error) {
	entry, err := getAuditEntryByRow(queryable.QueryRow(ctx, queryAuditByID, logID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuditLogEntry{}, false, nil
		}
		return AuditLogEntry{}, false, err
	}
	return entry, true, nil
}

func scanAuditEntry(rows persistence.Rows) (AuditLogEntry, error) {
	return getAuditEntryByRow(rows)
}

func matchAuditFilter(entry AuditLogEntry, filter AuditLogFilter) bool {
	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}
	if filter.OperatorID != "" && entry.Operator.UserID != filter.OperatorID {
		return false
	}
	if filter.TargetType != "" && entry.TargetType != filter.TargetType {
		return false
	}
	if filter.TargetID != "" && entry.TargetID != filter.TargetID {
		return false
	}
	if filter.StartTime != nil && entry.CreatedAt.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.CreatedAt.After(*filter.EndTime) {
		return false
	}
	return true
}
