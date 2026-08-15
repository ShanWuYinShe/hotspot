package admin

import (
	"context"
	"encoding/json"
	"errors"
	groupsmodule "hotspot/backend-go/internal/modules/groups"
	reportsmodule "hotspot/backend-go/internal/modules/reports"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
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
	record.item.Description = types.StrPtr(description)
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
	record.item.Description = types.StrPtr(description)
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

type adminUserRow struct {
	UserID            string
	Nickname          string
	AvatarURL         string
	Bio               string
	PresenceStatus    string
	LastOnline        time.Time
	LocationName      string
	Latitude          float64
	Longitude         float64
	UserType          string
	CreatedAt         time.Time
	Status            string
	LoginID           string
	RegisterIP        string
	RegisterLocation  string
	LastLoginIP       string
	LastLoginLocation string
	Email             string
	Phone             string
}

type adminGroupRow struct {
	GroupID        string
	Name           string
	AvatarURL      string
	Description    string
	HasPassword    bool
	LocationName   string
	Latitude       float64
	Longitude      float64
	JoinMethod     string
	IsSearchable   bool
	MaxMembers     int
	CreatedAt      time.Time
	OwnerID        string
	OwnerNickname  string
	OwnerAvatarURL string
	MembersCount   int
	LastActivityAt time.Time
}

type adminReportRow struct {
	ReportID           string
	TargetType         string
	TargetID           string
	Reason             string
	Description        string
	Status             string
	CreatedAt          time.Time
	HandledAt          *time.Time
	ReporterUserID     string
	ReporterNickname   string
	ReporterAvatarURL  string
	HandledByUserID    string
	HandledByNickname  string
	HandledByAvatar    string
	HandleNote         string
	MessageSnapshot    *string
	TargetUserID       string
	TargetUserNickname string
	TargetUserAvatar   string
}

func scanAdminUserRow(row persistence.Row) (AdminUserItem, error) {
	var record adminUserRow
	if err := row.Scan(
		&record.UserID,
		&record.Nickname,
		&record.AvatarURL,
		&record.Bio,
		&record.PresenceStatus,
		&record.LastOnline,
		&record.LocationName,
		&record.Latitude,
		&record.Longitude,
		&record.UserType,
		&record.CreatedAt,
		&record.Status,
		&record.LoginID,
		&record.RegisterIP,
		&record.RegisterLocation,
		&record.LastLoginIP,
		&record.LastLoginLocation,
		&record.Email,
		&record.Phone,
	); err != nil {
		return AdminUserItem{}, err
	}
	presenceStatus := usersmodule.UserPresenceStatus(record.PresenceStatus)
	var presenceStatusPtr *usersmodule.UserPresenceStatus
	if presenceStatus != "" {
		presenceStatusPtr = &presenceStatus
	}
	lastOnline := record.LastOnline.UTC().Format(time.RFC3339)
	lastActiveAt := record.LastOnline.UTC().Format(time.RFC3339)
	return AdminUserItem{
		User:              usersmodule.UserInfo{UserID: record.UserID, Nickname: record.Nickname, AvatarURL: types.StrPtr(record.AvatarURL)},
		Bio:               types.StrPtr(record.Bio),
		Tags:              []string{},
		PresenceStatus:    presenceStatusPtr,
		LastOnline:        types.StrPtr(lastOnline),
		Location:          &usersmodule.RelativeLocation{Point: usersmodule.GeoPoint{Latitude: record.Latitude, Longitude: record.Longitude}, LocationName: record.LocationName},
		Stats:             nil,
		Relation:          nil,
		Status:            UserAccountStatus(record.Status),
		CreatedAt:         record.CreatedAt.UTC().Format(time.RFC3339),
		LastActiveAt:      types.StrPtr(lastActiveAt),
		LoginID:           record.LoginID,
		RegisterIP:        strPtr(record.RegisterIP),
		RegisterLocation:  types.StrPtr(record.RegisterLocation),
		LastLoginIP:       strPtr(record.LastLoginIP),
		LastLoginLocation: types.StrPtr(record.LastLoginLocation),
		Email:             types.StrPtr(record.Email),
		Phone:             types.StrPtr(record.Phone),
	}, nil
}

func scanAdminGroupRow(row persistence.Row) (AdminGroupItem, error) {
	var record adminGroupRow
	if err := row.Scan(
		&record.GroupID,
		&record.Name,
		&record.AvatarURL,
		&record.Description,
		&record.HasPassword,
		&record.LocationName,
		&record.Latitude,
		&record.Longitude,
		&record.JoinMethod,
		&record.IsSearchable,
		&record.MaxMembers,
		&record.CreatedAt,
		&record.OwnerID,
		&record.OwnerNickname,
		&record.OwnerAvatarURL,
		&record.MembersCount,
		&record.LastActivityAt,
	); err != nil {
		return AdminGroupItem{}, err
	}
	return AdminGroupItem{
		Group:       groupsmodule.GroupInfo{GroupID: record.GroupID, Name: record.Name, AvatarURL: types.StrPtr(record.AvatarURL), MembersCount: record.MembersCount},
		Description: types.StrPtr(record.Description),
		HasPassword: record.HasPassword,
		Location: groupsmodule.Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: record.Latitude, Longitude: record.Longitude}, LocationName: record.LocationName},
		JoinMethod:     groupsmodule.GroupJoinMethod(record.JoinMethod),
		IsSearchable:   record.IsSearchable,
		MaxMembers:     record.MaxMembers,
		CreatedAt:      record.CreatedAt.UTC().Format(time.RFC3339),
		Owner:          usersmodule.UserInfo{UserID: record.OwnerID, Nickname: record.OwnerNickname, AvatarURL: types.StrPtr(record.OwnerAvatarURL)},
		LastActivityAt: types.StrPtr(record.LastActivityAt.UTC().Format(time.RFC3339)),
	}, nil
}

func scanAdminReportRow(row persistence.Row, withTargetUser bool) (reportHandleRecord, error) {
	var record adminReportRow
	if withTargetUser {
		if err := row.Scan(
			&record.ReportID,
			&record.TargetType,
			&record.TargetID,
			&record.Reason,
			&record.Description,
			&record.Status,
			&record.CreatedAt,
			&record.HandledAt,
			&record.ReporterUserID,
			&record.ReporterNickname,
			&record.ReporterAvatarURL,
			&record.HandledByUserID,
			&record.HandledByNickname,
			&record.HandledByAvatar,
			&record.HandleNote,
			&record.MessageSnapshot,
			&record.TargetUserID,
			&record.TargetUserNickname,
			&record.TargetUserAvatar,
		); err != nil {
			return reportHandleRecord{}, err
		}
	} else {
		if err := row.Scan(
			&record.ReportID,
			&record.TargetType,
			&record.TargetID,
			&record.Reason,
			&record.Description,
			&record.Status,
			&record.CreatedAt,
			&record.HandledAt,
			&record.ReporterUserID,
			&record.ReporterNickname,
			&record.ReporterAvatarURL,
			&record.HandledByUserID,
			&record.HandledByNickname,
			&record.HandledByAvatar,
			&record.HandleNote,
			&record.MessageSnapshot,
		); err != nil {
			return reportHandleRecord{}, err
		}
	}
	item := AdminReportItem{
		ReportID:    record.ReportID,
		TargetType:  ReportTargetType(record.TargetType),
		TargetID:    record.TargetID,
		Reason:      record.Reason,
		Description: types.StrPtr(record.Description),
		Status:      ReportStatus(record.Status),
		CreatedAt:   record.CreatedAt.UTC().Format(time.RFC3339),
		Reporter:    usersmodule.UserInfo{UserID: record.ReporterUserID, Nickname: record.ReporterNickname, AvatarURL: types.StrPtr(record.ReporterAvatarURL)},
		HandleNote:  types.StrPtr(record.HandleNote),
	}
	if record.MessageSnapshot != nil && *record.MessageSnapshot != "" {
		var snapshot reportsmodule.ReportSnapshot
		if err := json.Unmarshal([]byte(*record.MessageSnapshot), &snapshot); err != nil {
			return reportHandleRecord{}, err
		}
		item.MessageSnapshot = &snapshot
	}
	if record.HandledAt != nil {
		handledAt := record.HandledAt.UTC().Format(time.RFC3339)
		item.HandledAt = &handledAt
	}
	if record.HandledByUserID != "" {
		handledBy := usersmodule.UserInfo{UserID: record.HandledByUserID, Nickname: record.HandledByNickname, AvatarURL: types.StrPtr(record.HandledByAvatar)}
		item.HandledBy = &handledBy
	}
	result := reportHandleRecord{Item: item}
	if withTargetUser && record.TargetUserID != "" {
		result.TargetUser = usersmodule.UserInfo{UserID: record.TargetUserID, Nickname: record.TargetUserNickname, AvatarURL: types.StrPtr(record.TargetUserAvatar)}
	}
	return result, nil
}
