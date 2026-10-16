package admin

import (
	groupsmodule "hotspot/backend-go/internal/modules/groups"
	reportsmodule "hotspot/backend-go/internal/modules/reports"
	usersmodule "hotspot/backend-go/internal/modules/users"
)

type UserAccountStatus string

type ReportStatus string

type ReportTargetType string

type ReportAction string

const (
	UserAccountStatusNormal UserAccountStatus = "normal"
	UserAccountStatusBanned UserAccountStatus = "banned"

	// ReportStatus/ReportAction 收敛（spec api_spec_types.md）：reviewing/escalated/escalate 已删除
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"

	ReportTargetTypeUser    ReportTargetType = "user"
	ReportTargetTypeMoment  ReportTargetType = "moment"
	ReportTargetTypeComment ReportTargetType = "comment"
	ReportTargetTypeMessage ReportTargetType = "message"
	ReportTargetTypeGroup   ReportTargetType = "group"

	ReportActionResolve ReportAction = "resolve"
	ReportActionDismiss ReportAction = "dismiss"
)

type AdminUserItem struct {
	User              usersmodule.UserInfo            `json:"user"`
	Bio               *string                         `json:"bio"`
	Tags              []string                        `json:"tags"`
	PresenceStatus    *usersmodule.UserPresenceStatus `json:"presence_status"`
	LastOnline        *string                         `json:"last_online_at"`
	Location          *usersmodule.RelativeLocation   `json:"location"`
	Stats             *usersmodule.UserStats          `json:"stats"`
	Relation          *usersmodule.Relationship       `json:"relation"`
	Status            UserAccountStatus               `json:"status"`
	BannedUntil       *string                         `json:"banned_until"`
	PrivacyLevel      string                          `json:"privacy_level"`
	CreatedAt         string                          `json:"created_at"`
	LastActiveAt      *string                         `json:"last_active_at"`
	LoginID           string                          `json:"login_id"`
	RegisterIP        *string                         `json:"register_ip"`
	RegisterLocation  *string                         `json:"register_location"`
	LastLoginIP       *string                         `json:"last_login_ip"`
	LastLoginLocation *string                         `json:"last_login_location"`
	Email             *string                         `json:"email"`
	Phone             *string                         `json:"phone"`
}

type PaginatedAdminUsers struct {
	Items      []AdminUserItem `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}

type AdminUserSearchRequest struct {
	UserID     string               `json:"user_id,omitempty"`
	Nickname   string               `json:"nickname,omitempty"`
	LoginID    string               `json:"login_id,omitempty"`
	Status     UserAccountStatus    `json:"status,omitempty"`
	UserType   usersmodule.UserType `json:"user_type,omitempty"`
	Pagination *PaginationParams    `json:"pagination,omitempty"`
}

type AdminUserOpenTheDoorRequest struct {
	UserID string `json:"user_id"`
}

// AdminSetUserBanRequest 是两态统一端点 admin/users/set_ban 的请求类型（契约级）。
type AdminSetUserBanRequest struct {
	UserID       string `json:"user_id"`
	Active       bool   `json:"active"`
	Reason       string `json:"reason,omitempty"`
	DurationDays *int   `json:"duration_days,omitempty"`
}

type AdminGroupItem struct {
	Group          groupsmodule.GroupInfo       `json:"group"`
	Description    *string                      `json:"description"`
	Tags           []string                     `json:"tags"`
	HasPassword    bool                         `json:"has_password"`
	Location       groupsmodule.Location        `json:"location"`
	JoinMethod     groupsmodule.GroupJoinMethod `json:"join_method"`
	IsSearchable   bool                         `json:"is_searchable"`
	MaxMembers     int                          `json:"max_members"`
	CreatedAt      string                       `json:"created_at"`
	Owner          usersmodule.UserInfo         `json:"owner"`
	LastActivityAt *string                      `json:"last_activity_at"`
}

type PaginatedAdminGroups struct {
	Items      []AdminGroupItem `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

type AdminGroupSearchRequest struct {
	GroupID    string            `json:"group_id,omitempty"`
	Name       string            `json:"name,omitempty"`
	OwnerID    string            `json:"owner_id,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type AdminGroupOpenTheDoorRequest struct {
	GroupID string `json:"group_id"`
}

type AdminDismissGroupRequest struct {
	GroupID string `json:"group_id"`
	Reason  string `json:"reason,omitempty"`
}

type AdminReportItem struct {
	ReportID    string                `json:"report_id"`
	TargetType  ReportTargetType      `json:"target_type"`
	TargetID    string                `json:"target_id"`
	Reason      string                `json:"reason"`
	Description *string               `json:"description"`
	Status      ReportStatus          `json:"status"`
	CreatedAt   string                `json:"created_at"`
	HandledAt   *string               `json:"handled_at"`
	Reporter    usersmodule.UserInfo  `json:"reporter"`
	HandledBy   *usersmodule.UserInfo `json:"handled_by"`
	HandleNote  *string               `json:"handle_note"`
	// MessageSnapshot 仅 target_type='message' 时返回：管理员基于快照审核，
	// 不直接查询私聊/群聊消息库（spec api_spec_types.md ReportItem）。
	MessageSnapshot *reportsmodule.ReportSnapshot `json:"message_snapshot"`
	// Resolution 处理结果摘要（恒定存在，null = 尚未处理）：AdminReportItem
	// extends ReportItem（spec api_spec_admin.md），ReportResolution 脱敏不含
	// 处理者身份（spec api_spec_types.md）。
	Resolution *reportsmodule.ReportResolution `json:"resolution"`
}

type PaginatedAdminReports struct {
	Items      []AdminReportItem `json:"items"`
	NextCursor *string           `json:"next_cursor"`
	// Total 带 status 过滤时返回该状态总数（不受分页窗口影响）；status 为 null 时
	// 恒为 null（spec api_spec_admin.md AdminReportsPage.total，向后兼容新增字段）。
	Total *int64 `json:"total"`
}

type AdminReportListRequest struct {
	Status     ReportStatus      `json:"status,omitempty"`
	TargetType ReportTargetType  `json:"target_type,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}

type JudgeAdminReportRequest struct {
	ReportID        string       `json:"report_id"`
	Action          ReportAction `json:"action"`
	Note            string       `json:"note,omitempty"`
	BanDurationDays *int         `json:"ban_duration_days,omitempty"`
}

type AuditAction string

type AuditTargetType string

type AuditDetails map[string]any

const (
	AuditActionAnnouncementCreate AuditAction = "announcement.create"
	AuditActionAnnouncementUpdate AuditAction = "announcement.update"
	AuditActionAnnouncementDelete AuditAction = "announcement.delete"
	AuditActionRoleCreate         AuditAction = "role.create"
	AuditActionRoleUpdate         AuditAction = "role.update"
	AuditActionRoleDelete         AuditAction = "role.delete"
	AuditActionRoleSetPermissions AuditAction = "role.set_permissions"
	AuditActionAdminAward         AuditAction = "admin.award"
	AuditActionAdminRevoke        AuditAction = "admin.revoke"
	AuditActionTagSetBan          AuditAction = "tag.set_ban"
	AuditActionConfigUpdate       AuditAction = "config.update"
	AuditActionGroupDismiss       AuditAction = "group.dismiss"
)

const (
	AuditTargetTypeAnnouncement AuditTargetType = "announcement"
	AuditTargetTypeRole         AuditTargetType = "role"
	AuditTargetTypeAdmin        AuditTargetType = "admin"
	AuditTargetTypeTag          AuditTargetType = "tag"
	AuditTargetTypeConfig       AuditTargetType = "config"
	AuditTargetTypeGroup        AuditTargetType = "group"
)

type AuditLogItem struct {
	LogID         string          `json:"log_id"`
	Action        AuditAction     `json:"action"`
	Operator      UserInfo        `json:"operator"`
	TargetType    AuditTargetType `json:"target_type"`
	TargetID      string          `json:"target_id"`
	TargetName    *string         `json:"target_name"`
	Details       AuditDetails    `json:"details"`
	IPLocation    *string         `json:"ip_location"`
	IPAddress     *string         `json:"ip_address"`
	DeviceSummary *string         `json:"device_summary"`
	CreatedAt     string          `json:"created_at"`
}

type PaginatedAuditLogs struct {
	Items      []AuditLogItem `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

type AuditLogListRequest struct {
	Action     AuditAction       `json:"action,omitempty"`
	OperatorID string            `json:"operator_id,omitempty"`
	TargetType AuditTargetType   `json:"target_type,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	StartTime  string            `json:"start_time,omitempty"`
	EndTime    string            `json:"end_time,omitempty"`
	Pagination *PaginationParams `json:"pagination,omitempty"`
}
