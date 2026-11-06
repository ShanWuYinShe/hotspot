package admin

import (
	"context"
	"time"
)

type Repository interface {
	GetActor(context.Context, string) (AdminActor, bool, error)
	IsAdmin(context.Context, string) (bool, error)
	ListPermissions(context.Context) ([]AdminPermissionItem, error)
	ListRoles(context.Context) ([]AdminRoleItem, error)
	// GetRole 按 role_id 直查（审查 P2-23）：角色写操作的审计快照用它，避免
	// ListRoles 全量扫描 + 线性查找。
	GetRole(ctx context.Context, roleID string) (AdminRoleItem, bool, error)
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
	AppendAuditLog(context.Context, AuditLogEntry) error
	// DeleteAuditLogsBefore / DeleteHandledReportsBefore 物理删除超保留期数据，
	// 供后台 retention worker 调用（幂等；返回删除行数用于日志）。举报只删
	// resolved/dismissed 终态，pending 不受影响。
	DeleteAuditLogsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteHandledReportsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	// StatsGeo 地理统计聚合（区域/地点排名 + 全站存量 + 可选趋势），见 service_stats.go。
	StatsGeo(context.Context, geoStatsQuery) (AdminGeoStatsResponse, error)
	// StatsGeoDrilldown 区域下钻明细（keyset 分页），见 service_stats.go。
	StatsGeoDrilldown(context.Context, geoDrilldownQuery) (geoDrilldownPage, error)
}
