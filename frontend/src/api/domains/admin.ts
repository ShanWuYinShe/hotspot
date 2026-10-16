// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	AdminPermissions,
	AdminPermissionDictionary,
	AdminRoleList,
	AdminRoleItem,
	AdminOperatorItem,
	AuditAction,
	AuditTargetType,
	AuditLogItem,
	AdminGeoStatsResponse,
	AdminGeoDrilldownPage,
	StatsTimeRange,
	AppConfig,
	UpdateAppConfigRequest,
	UserAccountStatus,
	AdminUserItem,
	AdminGroupItem,
	AdminReportItem,
	AdminTagItem,
	AdminAnnouncementItem,
	ReportAction,
	ReportStatus,
	ReportTargetType,
	AnnounceSeverity,
	UserType,
	PaginatedData,
	PaginationParams,
} from '@/api/types';

// ── 平台管理（spec: api_spec_admin.md，/admin/api/v1；仅平台管理员）──

// RBAC
export async function adminRbacMe()                      { return request<AdminPermissions>(Endpoints.Admin.Rbac.GetMe); }
export async function adminListPermissions()             { return request<AdminPermissionDictionary>(Endpoints.Admin.Rbac.PermissionsFetch); }
export async function adminListRoles()                   { return request<AdminRoleList>(Endpoints.Admin.Rbac.RolesFetch); }
export async function adminCreateRole(params: { name: string; description: string | null })          { return request<AdminRoleItem>(Endpoints.Admin.Rbac.RolesCreate, params); }
export async function adminUpdateRole(params: { role_id: string; name: string | null; description: string | null; clear_description: boolean | null }) { return request<void>(Endpoints.Admin.Rbac.RolesUpdate, params); }
export async function adminDeleteRole(params: { role_id: string })                                   { return request<void>(Endpoints.Admin.Rbac.RolesDelete, params); }
export async function adminSetRolePermissions(params: { role_id: string; permission_keys: string[] }) { return request<void>(Endpoints.Admin.Rbac.RolesSetPermissions, params); }
export async function adminListAdmins(params?: { pagination?: PaginationParams })                    { return request<PaginatedData<AdminOperatorItem>>(Endpoints.Admin.Rbac.AdminsFetch, params ?? {}); }
export async function adminAwardAdmin(params: { user_id: string; award_all_permissions: boolean | null; role_ids: string[] }) { return request<void>(Endpoints.Admin.Rbac.AdminsAward, params); }
export async function adminRevokeAdmin(params: { user_id: string })                                  { return request<void>(Endpoints.Admin.Rbac.AdminsRevoke, params); }

// 审计日志
export async function adminListAuditLogs(params?: {
	action?: AuditAction | null; operator_id?: string | null; target_type?: AuditTargetType | null;
	target_id?: string | null; start_time?: string | null; end_time?: string | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AuditLogItem>>(Endpoints.Admin.Audit.Fetch, params ?? {}); }

// 地理统计（区域排名 + 热门地点 + 全站存量 + 每日新增趋势）
export async function adminStatsGeo(params?: {
	level?: number | null;
	time_range?: StatsTimeRange | null;
	limit?: number | null;
	trend_days?: number | null;
}) { return request<AdminGeoStatsResponse>(Endpoints.Admin.Stats.Geo, params ?? {}); }

// 区域下钻（网格桶内实体明细，keyset 分页）
export async function adminStatsGeoDrilldown(params: {
	cell_token: string;
	level?: number | null;
	entity_type: 'user' | 'group' | 'moment';
	time_range?: StatsTimeRange | null;
	pagination?: { limit?: number; cursor?: string };
}) { return request<AdminGeoDrilldownPage>(Endpoints.Admin.Stats.Drilldown, params); }

// 应用配置
export async function adminGetAppConfig()                { return request<AppConfig>(Endpoints.Admin.AppConfig.Get); }
export async function adminUpdateAppConfig(params: UpdateAppConfigRequest) { return request<void>(Endpoints.Admin.AppConfig.Update, params); }

// 用户管理
export async function adminSearchUsers(params?: {
	user_id?: string | null; nickname?: string | null; login_id?: string | null;
	status?: UserAccountStatus | null; user_type?: UserType | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AdminUserItem>>(Endpoints.Admin.Users.Search, params ?? {}); }
export async function adminOpenUser(params: { user_id: string })              { return request<AdminUserItem>(Endpoints.Admin.Users.OpenTheDoor, params); }
export async function adminSetUserBan(params: { user_id: string; active: boolean; reason: string | null; duration_days: number | null }) { return request<void>(Endpoints.Admin.Users.SetBan, params); }

// 群组管理
export async function adminSearchGroups(params?: {
	group_id?: string | null; name?: string | null; owner_id?: string | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AdminGroupItem>>(Endpoints.Admin.Groups.Search, params ?? {}); }
export async function adminOpenGroup(params: { group_id: string })            { return request<AdminGroupItem>(Endpoints.Admin.Groups.OpenTheDoor, params); }
export async function adminDismissGroup(params: { group_id: string; reason: string | null }) { return request<void>(Endpoints.Admin.Groups.Dismiss, params); }

// 举报审核
export async function adminListReports(params?: {
	status?: ReportStatus | null; target_type?: ReportTargetType | null; pagination?: PaginationParams;
}) {
	// total：带 status 过滤时后端返回该状态总数（不受分页影响），供「待处理」角标；否则 null
	return request<PaginatedData<AdminReportItem> & { total: number | null }>(Endpoints.Admin.Reports.Fetch, params ?? {});
}
export async function adminJudgeReport(params: { report_id: string; action: ReportAction; note: string | null; ban_duration_days: number | null }) { return request<void>(Endpoints.Admin.Reports.Judge, params); }

// 标签管理
export async function adminSearchTags(params?: { keyword?: string | null; pagination?: PaginationParams }) { return request<PaginatedData<AdminTagItem>>(Endpoints.Admin.Tags.Search, params ?? {}); }
export async function adminSetTagBan(params: { name: string; active: boolean; hard_delete: boolean | null }) { return request<void>(Endpoints.Admin.Tags.SetBan, params); }

// 系统公告（create 为契约幂等端点：必须携带 Idempotency-Key）
export async function adminListAnnouncements(params?: { severity?: AnnounceSeverity | null; pagination?: PaginationParams }) { return request<PaginatedData<AdminAnnouncementItem>>(Endpoints.Admin.Announcements.Fetch, params ?? {}); }
export async function adminCreateAnnouncement(params: { title: string; content: string; severity: AnnounceSeverity; action_url: string | null; send_push: boolean | null }) {
	return request<AdminAnnouncementItem>(Endpoints.Admin.Announcements.Create, params, { 'Idempotency-Key': newIdempotencyKey() });
}
export async function adminUpdateAnnouncement(params: { announcement_id: string; title: string | null; content: string | null; severity: AnnounceSeverity | null; action_url: string | null }) { return request<AdminAnnouncementItem>(Endpoints.Admin.Announcements.Update, params); }
export async function adminDeleteAnnouncement(params: { announcement_id: string }) { return request<void>(Endpoints.Admin.Announcements.Delete, params); }
