package admin

import "context"

type Repository interface {
	GetActor(context.Context, string) (AdminActor, bool, error)
	IsAdmin(context.Context, string) (bool, error)
	ListPermissions(context.Context) ([]AdminPermissionItem, error)
	ListRoles(context.Context) ([]AdminRoleItem, error)
	CreateRole(context.Context, AdminRbacRoleCreateRequest) (AdminRoleItem, error)
	UpdateRole(context.Context, AdminRbacRoleUpdateRequest) error
	DeleteRole(context.Context, string) error
	SetRolePermissions(context.Context, string, []string) error
	ListAdmins(context.Context, PaginationParams) (adminListPage, error)
	GrantAdmin(context.Context, AdminGrantUpsert) error
	RevokeAdmin(context.Context, string) error
	ListAuditLogs(context.Context, AuditLogFilter, bool) (auditLogPage, error)
	SearchUsers(context.Context, AdminUserSearchRequest) (adminUserPage, error)
	// ListUserIDsAfter 按 user_id 升序 keyset 分批遍历全量用户 id（系统公告
	// fanout 用）：SearchUsers 的 limit 会被 NormalizeLimit 钳到 100，复用它
	// 只能覆盖前 100 个用户。
	ListUserIDsAfter(ctx context.Context, afterUserID string, limit int) ([]string, error)
	GetUser(context.Context, string) (AdminUserItem, bool, error)
	GetUserBanState(context.Context, string) (UserBanState, bool, error)
	SetUserBanState(context.Context, string, UserBanState) error
	IsBanned(context.Context, string) (bool, error)
	ListUserSessions(context.Context, string) ([]string, error)
	RevokeSession(context.Context, string) error
	SearchGroups(context.Context, AdminGroupSearchRequest) (adminGroupPage, error)
	GetGroup(context.Context, string) (AdminGroupItem, bool, error)
	DismissGroup(context.Context, string) (AdminGroupItem, bool, error)
	ListGroupMemberIDs(context.Context, string) ([]string, error)
	SearchReports(context.Context, AdminReportListRequest) (adminReportPage, error)
	GetReportHandleRecord(context.Context, string) (reportHandleRecord, bool, error)
	HandleReport(context.Context, string, ReportAction, string, *UserInfo) (AdminReportItem, bool, error)
	RecallGroupMessage(context.Context, string) error
	RecallPrivateMessage(context.Context, string) error
	DeleteMoment(context.Context, string) error
	DeleteComment(context.Context, string) error
	AppendAuditLog(context.Context, AuditLogEntry) error
	// StatsGeo 地理统计聚合（区域/地点排名 + 全站存量），见 service_stats.go。
	StatsGeo(context.Context, geoStatsQuery) (AdminGeoStatsResponse, error)
}
