/**
 * 管理员服务
 * 严格对齐 spec/api_spec_admin_core.md + spec/api_spec_admin_moderation.md
 */

import { post, paginate, generateIdempotencyKey } from '../client';
import {
	ADMIN_RBAC,
	ADMIN_USERS,
	ADMIN_GROUPS,
	ADMIN_ANNOUNCEMENTS,
	ADMIN_AUDIT,
	ADMIN_REPORTS,
	ADMIN_TAGS,
	ADMIN_APP_CONFIG
} from '../endpoints';
import type {
	AdminPermissionItem,
	AdminPermissions,
	AdminRoleItem,
	AdminOperatorItem,
	AdminUserItem,
	AdminGroupItem,
	AdminReportItem,
	AdminAnnouncementItem,
	AdminTagItem,
	AdminTagSortBy,
	AuditLogItem,
	AuditAction,
	AuditTargetType,
	AnnouncementCreated,
	AnnouncementSeverity,
	ReportHandleAction,
	ReportStatus,
	ReportTargetType,
	UserAccountStatus,
	UserType,
	AppConfig,
	UploadConfig,
	RateLimitConfig,
	ResponseDefaults,
	GroupSettings,
	PaginationParams,
	Location
} from '$lib/types/api';

// ==================== RBAC ====================

export const adminRbacService = {
	/**
	 * 获取我的管理员权限
	 * spec: Response = AdminPermissions
	 */
	getMyPermissions() {
		return post<AdminPermissions>(ADMIN_RBAC.GET_ME);
	},

	/**
	 * 获取权限字典
	 * spec: Response = { items: AdminPermissionItem[] }
	 */
	getPermissions() {
		return post<{ items: AdminPermissionItem[] }>(ADMIN_RBAC.LIST_PERMISSIONS);
	},

	/**
	 * 获取角色列表（非分页，直接返回全部）
	 * spec: Response = { items: AdminRoleItem[] }
	 */
	getRoles() {
		return post<{ items: AdminRoleItem[] }>(ADMIN_RBAC.ROLES.LIST);
	},

	/**
	 * 创建角色
	 * spec: Request = { name, description? }
	 */
	createRole(data: { name: string; description?: string }) {
		return post<AdminRoleItem>(ADMIN_RBAC.ROLES.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 更新角色
	 * spec: Response = ApiResponse (void)
	 */
	updateRole(data: { role_id: string; name?: string; description?: string; clear_description?: boolean }) {
		return post<void>(ADMIN_RBAC.ROLES.UPDATE, data);
	},

	/**
	 * 删除角色
	 */
	deleteRole(data: { role_id: string }) {
		return post<void>(ADMIN_RBAC.ROLES.DELETE, data);
	},

	/**
	 * 设置角色权限
	 * spec: Request = { role_id, permission_keys }
	 */
	setRolePermissions(data: { role_id: string; permission_keys: string[] }) {
		return post<void>(ADMIN_RBAC.ROLES.SET_PERMISSIONS, data);
	},

	/**
	 * 获取管理员列表
	 * spec: Response = PaginatedData<AdminOperatorItem>
	 */
	getAdmins(params?: { pagination?: PaginationParams }) {
		return paginate<AdminOperatorItem>(ADMIN_RBAC.ADMINS.LIST, params);
	},

	/**
	 * 授予管理员权限
	 * spec: Request = { user_id, grant_all_permissions?, role_ids }
	 */
	grantAdmin(data: { user_id: string; grant_all_permissions?: boolean; role_ids: string[] }) {
		return post<void>(ADMIN_RBAC.ADMINS.GRANT, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 撤销管理员权限
	 */
	revokeAdmin(data: { user_id: string }) {
		return post<void>(ADMIN_RBAC.ADMINS.REVOKE, data);
	}
};

// ==================== 用户管理 ====================

export const adminUsersService = {
	/**
	 * 搜索用户
	 * spec: Response = PaginatedData<AdminUserItem>
	 */
	search(params: {
		user_id?: string;
		nickname?: string;
		login_id?: string;
		status?: UserAccountStatus;
		user_type?: UserType;
		pagination?: PaginationParams;
	}) {
		return paginate<AdminUserItem>(ADMIN_USERS.SEARCH, params);
	},

	/**
	 * 获取用户详情
	 * spec: Response = AdminUserItem
	 */
	getDetail(data: { user_id: string }) {
		return post<AdminUserItem>(ADMIN_USERS.OPEN_THE_DOOR, data);
	},

	/**
	 * 封禁用户
	 * spec: Request = { user_id, reason?, duration_days? }
	 */
	ban(data: { user_id: string; reason?: string; duration_days?: number }) {
		return post<void>(ADMIN_USERS.BAN, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 解封用户
	 * spec: Request = { user_id, reason? }
	 */
	unban(data: { user_id: string; reason?: string }) {
		return post<void>(ADMIN_USERS.UNBAN, data);
	}
};

// ==================== 群组管理 ====================

export const adminGroupsService = {
	/**
	 * 搜索群组
	 * spec: Response = PaginatedData<AdminGroupItem>
	 */
	search(params: {
		group_id?: string;
		name?: string;
		owner_id?: string;
		pagination?: PaginationParams;
	}) {
		return paginate<AdminGroupItem>(ADMIN_GROUPS.SEARCH, params);
	},

	/**
	 * 获取群组详情
	 * spec: Response = AdminGroupItem
	 */
	getDetail(data: { group_id: string }) {
		return post<AdminGroupItem>(ADMIN_GROUPS.OPEN_THE_DOOR, data);
	},

	/**
	 * 解散群组
	 * spec: Request = { group_id, reason? }
	 */
	dismiss(data: { group_id: string; reason?: string }) {
		return post<void>(ADMIN_GROUPS.DISMISS, data, { idempotencyKey: generateIdempotencyKey() });
	}
};

// ==================== 系统公告 ====================

export const adminAnnouncementsService = {
	/**
	 * 搜索公告列表
	 * spec: Request = { pagination?, severity? }
	 * spec: Response = PaginatedData<AdminAnnouncementItem>
	 */
	search(params?: { pagination?: PaginationParams; severity?: AnnouncementSeverity }) {
		return paginate<AdminAnnouncementItem>(ADMIN_ANNOUNCEMENTS.SEARCH, params);
	},

	/**
	 * 创建公告
	 */
	create(data: {
		title: string;
		content: string;
		severity: AnnouncementSeverity;
		action_url?: string;
		send_push?: boolean;
	}) {
		return post<AnnouncementCreated>(ADMIN_ANNOUNCEMENTS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 更新公告
	 * spec: Response = ApiResponse (void)
	 */
	update(data: {
		announcement_id: string;
		title?: string;
		content?: string;
		severity?: AnnouncementSeverity;
		action_url?: string;
	}) {
		return post<void>(ADMIN_ANNOUNCEMENTS.UPDATE, data);
	},

	/**
	 * 删除公告
	 */
	delete(data: { announcement_id: string }) {
		return post<void>(ADMIN_ANNOUNCEMENTS.DELETE, data);
	}
};

// ==================== 审计日志 ====================

export const adminAuditService = {
	/**
	 * 获取审计日志列表
	 * spec: Request includes target_id
	 */
	getList(params?: {
		action?: AuditAction;
		operator_id?: string;
		target_type?: AuditTargetType;
		target_id?: string;
		start_time?: string;
		end_time?: string;
		pagination?: PaginationParams;
	}) {
		return paginate<AuditLogItem>(ADMIN_AUDIT.LIST, params);
	},

	/**
	 * 获取审计日志详情
	 * spec: Request = { log_id }
	 */
	getDetail(data: { log_id: string }) {
		return post<AuditLogItem>(ADMIN_AUDIT.OPEN_THE_DOOR, data);
	}
};

// ==================== 举报审核 ====================

export const adminReportsService = {
	/**
	 * 搜索举报列表
	 * spec: Request = { pagination?, status?, target_type? }
	 * spec: Response = PaginatedData<AdminReportItem>
	 */
	search(params?: {
		status?: ReportStatus;
		target_type?: ReportTargetType;
		pagination?: PaginationParams;
	}) {
		return paginate<AdminReportItem>(ADMIN_REPORTS.SEARCH, params);
	},

	/**
	 * 处理举报
	 * spec: Request = { report_id, action, note? }
	 */
	handle(data: {
		report_id: string;
		action: ReportHandleAction;
		note?: string;
	}) {
		return post<void>(ADMIN_REPORTS.HANDLE, data, { idempotencyKey: generateIdempotencyKey() });
	}
};

// ==================== 标签管理 ====================

export const adminTagsService = {
	/**
	 * 搜索标签
	 * spec: Request = { pagination?, keyword?, sort_by? }
	 * spec: Response = PaginatedData<AdminTagItem>
	 */
	search(params?: {
		keyword?: string;
		sort_by?: AdminTagSortBy;
		pagination?: PaginationParams;
	}) {
		return paginate<AdminTagItem>(ADMIN_TAGS.SEARCH, params);
	},

	/**
	 * 删除标签
	 * spec: Request = { name }
	 */
	delete(data: { name: string }) {
		return post<void>(ADMIN_TAGS.DELETE, data);
	}
};

// ==================== 应用配置（统一 app_config） ====================

export const adminAppConfigService = {
	/**
	 * 获取完整应用配置（AppConfig）
	 * spec: AppConfig 包含 ClientConfig 的所有字段 + 仅服务端字段 + 元数据
	 */
	get() {
		return post<AppConfig>(ADMIN_APP_CONFIG.GET);
	},

	/**
	 * 更新上传配置
	 * spec: Request = { upload: Partial<UploadConfig> }
	 */
	updateUpload(data: { upload: Partial<UploadConfig> }) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_UPLOAD, data);
	},

	/**
	 * 更新默认配置
	 * spec: Request = UpdateDefaultsRequest
	 * default_location 为 Partial<Location>，即 { point?: GeoPoint; location_name?: string }
	 */
	updateDefaults(data: {
		default_location?: Partial<Location>;
		default_group_avatar?: string;
		default_user_avatar?: string;
		default_group_settings?: Partial<GroupSettings>;
	}) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_DEFAULTS, data);
	},

	/**
	 * 更新频率限制
	 * spec: Request = { rate_limits: Partial<RateLimitConfig> }
	 */
	updateRateLimits(data: { rate_limits: Partial<RateLimitConfig> }) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_RATE_LIMITS, data);
	},

	/**
	 * 更新响应默认值
	 * spec: Request = { response_defaults: Partial<ResponseDefaults> }
	 */
	updateResponseDefaults(data: { response_defaults: Partial<ResponseDefaults> }) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_RESPONSE_DEFAULTS, data);
	},

	/**
	 * 更新功能开关
	 */
	updateFeatures(data: {
		enable_public_registration?: boolean;
		enable_guest_login?: boolean;
		enable_moments?: boolean;
		enable_nearby_people?: boolean;
	}) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_FEATURES, data);
	},

	/**
	 * 更新限制配置
	 * spec: Request = { limits: Partial<AppLimits> }
	 */
	updateLimits(data: {
		limits: Partial<{
			max_group_members: number;
			max_moment_tags: number;
			max_moment_images: number;
			moment_description_max_length: number;
			comment_max_length: number;
			search_radius_max: number;
		}>;
	}) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_LIMITS, data);
	},

	/**
	 * 更新联系信息
	 */
	updateContact(data: {
		support_email?: string;
		privacy_policy_url?: string;
		terms_of_service_url?: string;
		official_website?: string;
	}) {
		return post<void>(ADMIN_APP_CONFIG.UPDATE_CONTACT, data);
	}
};
