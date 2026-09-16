package admin

import (
	"time"

	reportsmodule "hotspot/backend-go/internal/modules/reports"
	systemmodule "hotspot/backend-go/internal/modules/system"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/types"
)

type UserInfo = usersmodule.UserInfo

type GeoPoint = systemmodule.GeoPoint

type Location = systemmodule.Location

type FeatureSwitches = systemmodule.FeatureSwitches

type AppLimits = systemmodule.AppLimits

type ContactInfo = systemmodule.ContactInfo
type AnnounceSeverity = systemmodule.AnnounceSeverity

const (
	AnnounceLevelInfo    = systemmodule.AnnounceLevelInfo
	AnnounceLevelWarning = systemmodule.AnnounceLevelWarning
	AnnounceLevelError   = systemmodule.AnnounceLevelError
)

type PaginationParams = types.PaginationParams

type AdminRoleBasic struct {
	RoleID string `json:"role_id"`
	Name   string `json:"name"`
}

type AdminPermissions struct {
	Permissions []string         `json:"permissions"`
	Roles       []AdminRoleBasic `json:"roles"`
}

type AdminPermissionItem struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type AdminPermissionDictionary struct {
	Items []AdminPermissionItem `json:"items"`
}

type AdminRoleItem struct {
	RoleID      string   `json:"role_id"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

type AdminRoleList struct {
	Items []AdminRoleItem `json:"items"`
}

type AdminOperatorItem struct {
	User        UserInfo         `json:"user"`
	GrantedAt   string           `json:"granted_at"` // 最近一次授权时间（排序主序，RFC3339）
	Permissions []string         `json:"permissions"`
	Roles       []AdminRoleBasic `json:"roles"`
}

type PaginatedAdminOperators struct {
	Items      []AdminOperatorItem `json:"items"`
	NextCursor *string             `json:"next_cursor"`
}

type AdminRbacRoleCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type AdminRbacRoleUpdateRequest struct {
	RoleID           string  `json:"role_id"`
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ClearDescription bool    `json:"clear_description,omitempty"`
}

type AdminRbacRoleDeleteRequest struct {
	RoleID string `json:"role_id"`
}

type AdminRbacRoleSetPermissionsRequest struct {
	RoleID         string   `json:"role_id"`
	PermissionKeys []string `json:"permission_keys"`
}

type AdminRbacAdminsListRequest struct {
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminRbacGrantAdminRequest struct {
	UserID              string   `json:"user_id"`
	AwardAllPermissions bool     `json:"award_all_permissions,omitempty"`
	RoleIDs             []string `json:"role_ids,omitempty"`
}

type AdminRbacRevokeAdminRequest struct {
	UserID string `json:"user_id"`
}

type AdminActor struct {
	UserID string
	// Nickname 供审计日志 operator 展示；repo 未能取到时留空，
	// newAuditEntry 会兜底回填 UserID。
	Nickname            string
	Permissions         []string
	Roles               []AdminRoleBasic
	GrantAllPermissions bool
}

func (a AdminActor) IsSuperAdmin() bool {
	return a.GrantAllPermissions || containsString(a.Permissions, wildcardPermission)
}

func (a AdminActor) HasPermission(key string) bool {
	return a.IsSuperAdmin() || containsString(a.Permissions, key)
}

func (a AdminActor) Response() AdminPermissions {
	permissions := append([]string(nil), a.Permissions...)
	roles := append([]AdminRoleBasic(nil), a.Roles...)
	return AdminPermissions{Permissions: permissions, Roles: roles}
}

type adminCursor struct {
	GrantedAt time.Time `json:"granted_at"`
	UserID    string    `json:"user_id"`
}

type adminListPage struct {
	Items []AdminOperatorItem
	Next  *adminCursor
}

type auditCursor struct {
	CreatedAt time.Time `json:"created_at"`
	LogID     string    `json:"log_id"`
}

type AuditLogFilter struct {
	Action     AuditAction
	OperatorID string
	TargetType AuditTargetType
	TargetID   string
	StartTime  *time.Time
	EndTime    *time.Time
	Pagination PaginationParams
}

type AuditLogEntry struct {
	LogID         string
	Action        AuditAction
	Operator      UserInfo
	TargetType    AuditTargetType
	TargetID      string
	TargetName    string
	Details       AuditDetails
	IPLocation    string
	IPAddress     string
	DeviceSummary string
	CreatedAt     time.Time
}

type auditLogPage struct {
	Items []AuditLogItem
	Next  *auditCursor
}

type adminUserCursor struct {
	CreatedAt time.Time `json:"created_at"`
	UserID    string    `json:"user_id"`
}

type adminUserPage struct {
	Items []AdminUserItem
	Next  *adminUserCursor
}

type adminGroupCursor struct {
	CreatedAt time.Time `json:"created_at"`
	GroupID   string    `json:"group_id"`
}

type adminGroupPage struct {
	Items []AdminGroupItem
	Next  *adminGroupCursor
}

type adminReportCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ReportID  string    `json:"report_id"`
}

type adminReportPage struct {
	Items []AdminReportItem
	Next  *adminReportCursor
}

type UserBanState struct {
	Status       UserAccountStatus
	Reason       string
	DurationDays int
	BannedAt     time.Time
}

// Expired 限期封禁是否已到期（spec api_spec_admin.md users/set_ban 区分永久
// 与限期）：DurationDays<=0 为永久封禁，需手动解封；BannedAt 零值（历史行缺
// 时间戳）按永久处理，避免误解封（fail-closed）。调用方一律经此判定，
// 不得只看 Status 位。
func (s UserBanState) Expired(now time.Time) bool {
	if s.Status != UserAccountStatusBanned {
		return false
	}
	if s.DurationDays <= 0 || s.BannedAt.IsZero() {
		return false
	}
	return !now.Before(s.BannedAt.Add(time.Duration(s.DurationDays) * 24 * time.Hour))
}

type reportHandleRecord struct {
	Item       AdminReportItem
	TargetUser usersmodule.UserInfo
}

func reportItemFromModule(item reportsmodule.ReportItem, reporter usersmodule.UserInfo, handledBy *usersmodule.UserInfo, handleNote string) AdminReportItem {
	return AdminReportItem{
		ReportID:        item.ReportID,
		TargetType:      ReportTargetType(item.TargetType),
		TargetID:        item.TargetID,
		Reason:          string(item.Reason),
		Description:     item.Description,
		Status:          ReportStatus(item.Status),
		CreatedAt:       item.CreatedAt,
		Reporter:        reporter,
		HandledBy:       handledBy,
		HandleNote:      types.StrPtr(handleNote),
		MessageSnapshot: item.MessageSnapshot,
		Resolution:      item.Resolution,
	}
}
