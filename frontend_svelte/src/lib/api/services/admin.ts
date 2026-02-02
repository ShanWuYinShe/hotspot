/**
 * 管理员服务
 */

import { post, paginate } from '../client';
import {
	ADMIN_RBAC,
	ADMIN_USERS,
	ADMIN_GROUPS,
	ADMIN_ANNOUNCEMENTS,
	ADMIN_AUDIT,
	ADMIN_REPORTS,
	ADMIN_TAGS,
	ADMIN_GLOBAL_SETTINGS
} from '../endpoints';
import type {
	AdminPermission,
	AdminRole,
	AdminUser,
	UserCard,
	GroupCard,
	GroupSettings,
	SystemAnnouncementItem,
	AnnouncementCreated,
	AuditLogItem,
	ReportItem,
	GlobalSettings,
	AnnouncementSeverity
} from '$lib/types/api';

// ==================== RBAC ====================

export const adminRbacService = {
	/**
	 * 获取我的管理员权限
	 */
	getMyPermissions() {
		return post<{
			is_admin: boolean;
			permissions: string[];
			roles: AdminRole[];
		}>(ADMIN_RBAC.ME);
	},

	/**
	 * 获取权限字典
	 */
	getPermissions() {
		return post<AdminPermission[]>(ADMIN_RBAC.PERMISSIONS);
	},

	/**
	 * 获取角色列表
	 */
	getRoles(params?: { limit?: number; cursor?: string }) {
		return paginate<AdminRole>(ADMIN_RBAC.ROLES.LIST, params);
	},

	/**
	 * 创建角色
	 */
	createRole(data: { name: string; description?: string; permissions?: string[] }) {
		return post<AdminRole>(ADMIN_RBAC.ROLES.CREATE, data);
	},

	/**
	 * 更新角色
	 */
	updateRole(data: { role_id: string; name?: string; description?: string }) {
		return post<AdminRole>(ADMIN_RBAC.ROLES.UPDATE, data);
	},

	/**
	 * 删除角色
	 */
	deleteRole(data: { role_id: string }) {
		return post<void>(ADMIN_RBAC.ROLES.DELETE, data);
	},

	/**
	 * 设置角色权限
	 */
	setRolePermissions(data: { role_id: string; permissions: string[] }) {
		return post<void>(ADMIN_RBAC.ROLES.SET_PERMISSIONS, data);
	},

	/**
	 * 获取管理员列表
	 */
	getAdmins(params?: { limit?: number; cursor?: string }) {
		return paginate<AdminUser>(ADMIN_RBAC.ADMINS.LIST, params);
	},

	/**
	 * 授予管理员权限
	 */
	grantAdmin(data: { user_id: number; role_ids: string[] }) {
		return post<void>(ADMIN_RBAC.ADMINS.GRANT, data);
	},

	/**
	 * 撤销管理员权限
	 */
	revokeAdmin(data: { user_id: number }) {
		return post<void>(ADMIN_RBAC.ADMINS.REVOKE, data);
	}
};

// ==================== 用户管理 ====================

export const adminUsersService = {
	/**
	 * 搜索用户
	 */
	search(params: {
		user_id?: number;
		nickname?: string;
		login_id?: string;
		status?: 'normal' | 'banned';
		user_type?: 'temp' | 'normal' | 'admin';
		limit?: number;
		cursor?: string;
	}) {
		return paginate<UserCard & { status: 'normal' | 'banned' }>(ADMIN_USERS.SEARCH, params);
	},

	/**
	 * 获取用户详情
	 */
	getDetail(data: { user_id: number }) {
		return post<
			UserCard & {
				status: 'normal' | 'banned';
				email?: string;
				phone?: string;
				created_at: string;
				banned_at?: string;
				ban_reason?: string;
			}
		>(ADMIN_USERS.DETAIL, data);
	},

	/**
	 * 封禁用户
	 */
	ban(data: { user_id: number; reason: string; duration?: number }) {
		return post<void>(ADMIN_USERS.BAN, data);
	},

	/**
	 * 解封用户
	 */
	unban(data: { user_id: number }) {
		return post<void>(ADMIN_USERS.UNBAN, data);
	}
};

// ==================== 群组管理 ====================

export const adminGroupsService = {
	/**
	 * 搜索群组
	 */
	search(params: { group_chat_id?: string; name?: string; owner_id?: number; limit?: number; cursor?: string }) {
		return paginate<GroupCard>(ADMIN_GROUPS.SEARCH, params);
	},

	/**
	 * 获取群组详情
	 */
	getDetail(data: { group_chat_id: string }) {
		return post<
			GroupCard & {
				members_count: number;
				created_at: string;
			}
		>(ADMIN_GROUPS.DETAIL, data);
	},

	/**
	 * 解散群组
	 */
	dismiss(data: { group_chat_id: string; reason: string }) {
		return post<void>(ADMIN_GROUPS.DISMISS, data);
	}
};

// ==================== 系统公告 ====================

export const adminAnnouncementsService = {
	/**
	 * 获取公告列表
	 */
	getList(params?: { limit?: number; cursor?: string }) {
		return paginate<SystemAnnouncementItem>(ADMIN_ANNOUNCEMENTS.LIST, params);
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
		return post<AnnouncementCreated>(ADMIN_ANNOUNCEMENTS.CREATE, data);
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
	 */
	getList(params?: {
		action?: string;
		operator_id?: number;
		target_type?: string;
		start_time?: string;
		end_time?: string;
		limit?: number;
		cursor?: string;
	}) {
		return paginate<AuditLogItem>(ADMIN_AUDIT.LIST, params);
	},

	/**
	 * 获取审计日志详情
	 */
	getDetail(data: { audit_id: string }) {
		return post<AuditLogItem>(ADMIN_AUDIT.DETAIL, data);
	},

	/**
	 * 导出审计日志
	 */
	export(data: {
		start_time: string;
		end_time: string;
		action?: string;
		format?: 'csv' | 'json';
	}) {
		return post<{ task_id: string }>(ADMIN_AUDIT.EXPORT, data);
	},

	/**
	 * 获取导出状态
	 */
	getExportStatus(data: { task_id: string }) {
		return post<{
			status: 'pending' | 'processing' | 'completed' | 'failed';
			download_url?: string;
			error?: string;
		}>(ADMIN_AUDIT.EXPORT_STATUS, data);
	}
};

// ==================== 举报审核 ====================

export const adminReportsService = {
	/**
	 * 获取举报列表
	 */
	getList(params?: {
		status?: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
		limit?: number;
		cursor?: string;
	}) {
		return paginate<ReportItem>(ADMIN_REPORTS.LIST, params);
	},

	/**
	 * 处理举报
	 */
	handle(data: {
		report_id: string;
		action: 'resolve' | 'dismiss';
		resolution?: string;
		punish_target?: boolean;
		punishment?: {
			type: 'warning' | 'mute' | 'ban';
			duration?: number;
			reason: string;
		};
	}) {
		return post<void>(ADMIN_REPORTS.HANDLE, data);
	}
};

// ==================== 标签管理 ====================

export const adminTagsService = {
	/**
	 * 删除标签
	 */
	delete(data: { tag: string }) {
		return post<void>(ADMIN_TAGS.DELETE, data);
	}
};

// ==================== 全局配置 ====================

export const adminGlobalSettingsService = {
	/**
	 * 获取全局配置
	 */
	fetch() {
		return post<GlobalSettings>(ADMIN_GLOBAL_SETTINGS.FETCH);
	},

	/**
	 * 更新上传配置
	 */
	updateUpload(data: {
		image_max_size?: number;
		video_max_size?: number;
		audio_max_size?: number;
		file_max_size?: number;
		allowed_image_types?: string[];
		allowed_video_types?: string[];
	}) {
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_UPLOAD, data);
	},

	/**
	 * 更新默认配置
	 */
	updateDefaults(data: {
		default_location_name?: string;
		default_group_avatar?: string;
		default_user_avatar?: string;
		default_group_settings?: GroupSettings;
	}) {
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_DEFAULTS, data);
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
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_FEATURES, data);
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
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_CONTACT, data);
	},

	/**
	 * 更新限制配置
	 */
	updateLimits(data: {
		max_group_members?: number;
		max_moment_tags?: number;
		max_moment_images?: number;
		moment_description_max_length?: number;
		comment_max_length?: number;
		search_radius_max?: number;
	}) {
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_LIMITS, data);
	},

	/**
	 * 更新频率限制
	 */
	updateRateLimits(data: {
		global_per_hour?: number;
		user_per_hour?: number;
		auth_send_code_per_minute?: number;
		auth_verify_code_per_10minutes?: number;
		auth_register_per_10minutes?: number;
		auth_login_per_10minutes?: number;
		auth_guest_per_10minutes?: number;
		auth_refresh_per_10minutes?: number;
		auth_reset_request_per_minute?: number;
		auth_reset_verify_per_10minutes?: number;
		auth_reset_per_10minutes?: number;
		moments_create_per_minute?: number;
		comments_create_per_minute?: number;
		reports_create_per_hour?: number;
	}) {
		return post<void>(ADMIN_GLOBAL_SETTINGS.UPDATE_RATE_LIMITS, data);
	}
};
