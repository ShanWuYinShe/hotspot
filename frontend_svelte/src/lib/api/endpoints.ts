/**
 * API 路径常量
 *
 * 严格遵循 spec/openapi/ 下的 OpenAPI 3.1 规范
 * 所有端点统一使用 POST /api/v1/ 前缀
 */

export const API_VERSION = 'v1';
export const API_BASE = `/api/${API_VERSION}`;

// ============================================================================
// 认证 API（spec/openapi/paths/auth.yaml）
// ============================================================================
export const AUTH = {
	REGISTER: `${API_BASE}/auth/register`,
	LOGIN: `${API_BASE}/auth/login`,
	LOGOUT: `${API_BASE}/auth/logout`,
	GUEST: `${API_BASE}/auth/guest`,
	REFRESH_TOKEN: `${API_BASE}/auth/refresh_token`,
	CHECK_STATUS: `${API_BASE}/auth/check_status`,
	REQUEST_PASSWORD_RESET: `${API_BASE}/auth/request_password_reset`,
	RESET_PASSWORD: `${API_BASE}/auth/reset_password`,
	ME: {
		CHANGE_PASSWORD: `${API_BASE}/auth/me/change_password`,
		SESSIONS_LIST: `${API_BASE}/auth/me/sessions/list`,
		REVOKE_SESSION: `${API_BASE}/auth/me/sessions/revoke`,
		REVOKE_ALL_SESSIONS: `${API_BASE}/auth/me/sessions/revoke_all`
	}
} as const;

// ============================================================================
// 用户 API（spec/openapi/paths/users.yaml）
// ============================================================================
export const USERS = {
	// 搜索（单一端点，通过请求体 keyword/tags/location 参数区分）
	SEARCH: `${API_BASE}/users/search`,
	// 获取指定用户信息
	OPEN_THE_DOOR: `${API_BASE}/users/open_the_door`,
	LIST_FRIENDS: `${API_BASE}/users/list_friends`,
	LIST_FOLLOWING: `${API_BASE}/users/list_following`,
	LIST_FOLLOWERS: `${API_BASE}/users/list_followers`,
	LIST_GROUPS: `${API_BASE}/users/list_groups`,
	LIST_MOMENTS: `${API_BASE}/users/list_moments`,
	LIST_LIKED_MOMENTS: `${API_BASE}/users/list_liked_moments`,
	// 当前用户
	ME: {
		PROFILE_GET: `${API_BASE}/users/me/profile/get`,
		PROFILE_UPDATE: `${API_BASE}/users/me/profile/update`,
		LOCATION_UPDATE: `${API_BASE}/users/me/location/update`,
		UPGRADE: `${API_BASE}/users/me/upgrade`,
		SETTINGS_GET: `${API_BASE}/users/me/settings/get`,
		SETTINGS_UPDATE: `${API_BASE}/users/me/settings/update`,
		DELETE: `${API_BASE}/users/me/delete`,
		LIST_BLOCKED: `${API_BASE}/users/me/list_blocked`,
		// 关注关系操作
		FOLLOW: `${API_BASE}/users/me/follow`,
		UNFOLLOW: `${API_BASE}/users/me/unfollow`,
		BLOCK: `${API_BASE}/users/me/block`,
		UNBLOCK: `${API_BASE}/users/me/unblock`,
		SET_REMARK: `${API_BASE}/users/me/set_remark`,
		// ⏳ 延期：邮箱/手机号变更（依赖验证码发送服务）
		EMAIL_REQUEST_CHANGE: `${API_BASE}/users/me/email/request_change`,
		EMAIL_CONFIRM_CHANGE: `${API_BASE}/users/me/email/confirm_change`,
		PHONE_REQUEST_CHANGE: `${API_BASE}/users/me/phone/request_change`,
		PHONE_CONFIRM_CHANGE: `${API_BASE}/users/me/phone/confirm_change`
	}
} as const;

// ============================================================================
// 生活瞬间 API（spec/openapi/paths/moments.yaml）
// ============================================================================
export const MOMENTS = {
	// 搜索（单一端点，通过请求体参数区分）
	SEARCH: `${API_BASE}/moments/search`,
	// 获取指定瞬间信息
	OPEN_THE_DOOR: `${API_BASE}/moments/open_the_door`,
	LIST_LIKES: `${API_BASE}/moments/list_likes`,
	// 操作
	CREATE: `${API_BASE}/moments/create`,
	UPDATE: `${API_BASE}/moments/update`,
	DELETE: `${API_BASE}/moments/delete`,
	// 当前用户操作
	ME: {
		LIKE: `${API_BASE}/moments/me/like`,
		UNLIKE: `${API_BASE}/moments/me/unlike`,
		FAVORITE: `${API_BASE}/moments/me/favorite`,
		UNFAVORITE: `${API_BASE}/moments/me/unfavorite`,
		LIST: `${API_BASE}/moments/me/list`,
		LIST_FAVORITES: `${API_BASE}/moments/me/list_favorites`
	}
} as const;

// ============================================================================
// 评论 API（spec/openapi/paths/moments.yaml - 评论部分）
// ============================================================================
export const COMMENTS = {
	LIST: `${API_BASE}/comments/list`,
	LIST_REPLIES: `${API_BASE}/comments/list_replies`,
	LIST_LIKES: `${API_BASE}/comments/list_likes`,
	CREATE: `${API_BASE}/comments/create`,
	DELETE: `${API_BASE}/comments/delete`,
	ME: {
		LIKE: `${API_BASE}/comments/me/like`,
		UNLIKE: `${API_BASE}/comments/me/unlike`
	}
} as const;

// ============================================================================
// 私聊 API（spec/openapi/paths/chat.yaml）
// ============================================================================
export const PRIVATE_CHATS = {
	OPEN_THE_DOOR: `${API_BASE}/private_chats/open_the_door`,
	ME: {
		LIST: `${API_BASE}/private_chats/me/list`,
		UPDATE_SETTINGS: `${API_BASE}/private_chats/me/update_settings`,
		// ⏳ 延期：删除私聊会话
		DELETE: `${API_BASE}/private_chats/me/delete`
	}
} as const;

// ============================================================================
// 私聊消息 API（spec/openapi/paths/chat.yaml）
// ============================================================================
export const PRIVATE_MESSAGES = {
	SEND: `${API_BASE}/private_messages/send`,
	GET_HISTORY: `${API_BASE}/private_messages/get_history`,
	UPDATE: `${API_BASE}/private_messages/update`,
	RECALL: `${API_BASE}/private_messages/recall`,
	ME: {
		MARK_AS_READ: `${API_BASE}/private_messages/me/mark_as_read`,
		MARK_ALL_READ: `${API_BASE}/private_messages/me/mark_all_read`
	}
} as const;

// ============================================================================
// 群组 API（spec/openapi/paths/groups.yaml）
// ============================================================================
export const GROUPS = {
	// 搜索（单一端点，通过请求体参数区分）
	SEARCH: `${API_BASE}/groups/search`,
	// 获取指定群组信息
	OPEN_THE_DOOR: `${API_BASE}/groups/open_the_door`,
	// 操作
	CREATE: `${API_BASE}/groups/create`,
	UPDATE: `${API_BASE}/groups/update`,
	DISMISS: `${API_BASE}/groups/dismiss`,
	// 成员管理
	MEMBERS: {
		LIST: `${API_BASE}/groups/members/list`,
		SEARCH: `${API_BASE}/groups/members/search`,
		INVITE: `${API_BASE}/groups/members/invite`,
		SET_ROLE: `${API_BASE}/groups/members/set_role`,
		MUTE: `${API_BASE}/groups/members/mute`,
		UNMUTE: `${API_BASE}/groups/members/unmute`,
		BAN: `${API_BASE}/groups/members/ban`,
		UNBAN: `${API_BASE}/groups/members/unban`,
		LIST_BANNED: `${API_BASE}/groups/members/list_banned`,
		REMOVE_MEMBER: `${API_BASE}/groups/members/remove_member`
	},
	// 申请管理
	APPLICATIONS: {
		LIST: `${API_BASE}/groups/applications/list`,
		APPROVE: `${API_BASE}/groups/applications/approve`,
		REJECT: `${API_BASE}/groups/applications/reject`
	},
	// 群公告
	ANNOUNCEMENTS: {
		LIST: `${API_BASE}/groups/announcements/list`,
		CREATE: `${API_BASE}/groups/announcements/create`,
		UPDATE: `${API_BASE}/groups/announcements/update`,
		DELETE: `${API_BASE}/groups/announcements/delete`
	},
	// 当前用户
	ME: {
		UPDATE_SETTINGS: `${API_BASE}/groups/me/update_settings`,
		JOIN: `${API_BASE}/groups/me/join`,
		TRANSFER_OWNERSHIP: `${API_BASE}/groups/me/transfer_ownership`,
		LIST: `${API_BASE}/groups/me/list`,
		APPLICATIONS_LIST: `${API_BASE}/groups/me/applications/list`,
		CANCEL_APPLICATION: `${API_BASE}/groups/me/applications/cancel`,
		INVITES_LIST: `${API_BASE}/groups/me/invites/list`,
		ACCEPT_INVITE: `${API_BASE}/groups/me/invites/accept`,
		REJECT_INVITE: `${API_BASE}/groups/me/invites/reject`,
		LEAVE: `${API_BASE}/groups/me/leave`
	}
} as const;

// ============================================================================
// 群聊消息 API（spec/openapi/paths/chat.yaml）
// ============================================================================
export const GROUP_MESSAGES = {
	SEND: `${API_BASE}/group_messages/send`,
	GET_HISTORY: `${API_BASE}/group_messages/get_history`,
	UPDATE: `${API_BASE}/group_messages/update`,
	RECALL: `${API_BASE}/group_messages/recall`,
	ME: {
		MARK_AS_READ: `${API_BASE}/group_messages/me/mark_as_read`,
		MARK_ALL_READ: `${API_BASE}/group_messages/me/mark_all_read`
	}
} as const;

// ============================================================================
// 信息流 API（spec/openapi/paths/feed.yaml）
// ============================================================================
export const FEED = {
	LIST_MOMENTS: `${API_BASE}/feed/list_moments`,
	LIST_GROUPS: `${API_BASE}/feed/list_groups`,
	LIST_USERS: `${API_BASE}/feed/list_users`,
	LIST_TRENDING: `${API_BASE}/feed/list_trending`,
	ME: {
		DISMISS: `${API_BASE}/feed/me/dismiss`
	}
} as const;

// ============================================================================
// 搜索 API（spec/openapi/paths/search.yaml）
// ============================================================================
export const SEARCH = {
	SUGGEST: `${API_BASE}/search/suggest`
} as const;

// ============================================================================
// 地图 API（spec/openapi/paths/map.yaml）
// ============================================================================
export const MAP = {
	RENDER: `${API_BASE}/map/render`
} as const;

// ============================================================================
// 分享 API（spec/openapi/paths/share.yaml）
// ============================================================================
export const SHARE = {
	CREATE: `${API_BASE}/share/create`,
	GET: `${API_BASE}/share/get`,
	// ME_LIST deferred: share/me/list
	REVOKE: `${API_BASE}/share/revoke`
} as const;

// ============================================================================
// 媒体上传 API（spec/openapi/paths/media.yaml）
// ============================================================================
export const MEDIA = {
	PRESIGN_UPLOAD: `${API_BASE}/media/presign_upload`,
	BATCH_PRESIGN_UPLOAD: `${API_BASE}/media/batch_presign_upload`,
	COMPLETE_UPLOAD: `${API_BASE}/media/complete_upload`
} as const;

// ============================================================================
// 通知 API（spec/api_spec_notification.mdc）
// ============================================================================
export const NOTIFICATIONS = {
	LIST: `${API_BASE}/notifications/list`,
	UNREAD_COUNT: `${API_BASE}/notifications/unread_count`,
	MARK_AS_READ: `${API_BASE}/notifications/mark_as_read`,
	MARK_ALL_READ: `${API_BASE}/notifications/mark_all_read`
} as const;

// ============================================================================
// 举报 API（spec/openapi/paths/report.yaml）
// ============================================================================
export const REPORTS = {
	CREATE: `${API_BASE}/reports/create`,
	ME_LIST: `${API_BASE}/reports/me/list`
} as const;

// ============================================================================
// 系统 API（spec/openapi/paths/system.yaml）
// ============================================================================
export const SYSTEM = {
	GET_META: `${API_BASE}/system/get_meta`,
	HEALTH: `${API_BASE}/system/health`,
	LIST_ANNOUNCEMENTS: `${API_BASE}/system/list_announcements`,
	GET_CLIENT_CONFIG: `${API_BASE}/system/get_client_config`
} as const;

// ============================================================================
// 管理员 API（spec/openapi/paths/admin-core.yaml + admin-moderation.yaml）
// ============================================================================

// 管理员 - RBAC
export const ADMIN_RBAC = {
	GET_ME: `${API_BASE}/admin/rbac/get_me`,
	LIST_PERMISSIONS: `${API_BASE}/admin/rbac/list_permissions`,
	ROLES: {
		LIST: `${API_BASE}/admin/rbac/roles/list`,
		CREATE: `${API_BASE}/admin/rbac/roles/create`,
		UPDATE: `${API_BASE}/admin/rbac/roles/update`,
		DELETE: `${API_BASE}/admin/rbac/roles/delete`,
		SET_PERMISSIONS: `${API_BASE}/admin/rbac/roles/set_permissions`
	},
	ADMINS: {
		LIST: `${API_BASE}/admin/rbac/admins/list`,
		GRANT: `${API_BASE}/admin/rbac/admins/grant`,
		REVOKE: `${API_BASE}/admin/rbac/admins/revoke`
	}
} as const;

// 管理员 - 应用配置（spec 统一路径 admin/app_config/*）
export const ADMIN_APP_CONFIG = {
	GET: `${API_BASE}/admin/app_config/get`,
	UPDATE_UPLOAD: `${API_BASE}/admin/app_config/update_upload`,
	UPDATE_DEFAULTS: `${API_BASE}/admin/app_config/update_defaults`,
	UPDATE_RATE_LIMITS: `${API_BASE}/admin/app_config/update_rate_limits`,
	UPDATE_RESPONSE_DEFAULTS: `${API_BASE}/admin/app_config/update_response_defaults`,
	UPDATE_FEATURES: `${API_BASE}/admin/app_config/update_features`,
	UPDATE_LIMITS: `${API_BASE}/admin/app_config/update_limits`,
	UPDATE_CONTACT: `${API_BASE}/admin/app_config/update_contact`
} as const;

// 管理员 - 用户管理
export const ADMIN_USERS = {
	SEARCH: `${API_BASE}/admin/users/search`,
	OPEN_THE_DOOR: `${API_BASE}/admin/users/open_the_door`,
	BAN: `${API_BASE}/admin/users/ban`,
	UNBAN: `${API_BASE}/admin/users/unban`
} as const;

// 管理员 - 群组管理
export const ADMIN_GROUPS = {
	SEARCH: `${API_BASE}/admin/groups/search`,
	OPEN_THE_DOOR: `${API_BASE}/admin/groups/open_the_door`,
	DISMISS: `${API_BASE}/admin/groups/dismiss`
} as const;

// 管理员 - 系统公告
export const ADMIN_ANNOUNCEMENTS = {
	SEARCH: `${API_BASE}/admin/announcements/search`,
	CREATE: `${API_BASE}/admin/announcements/create`,
	UPDATE: `${API_BASE}/admin/announcements/update`,
	DELETE: `${API_BASE}/admin/announcements/delete`
} as const;

// 管理员 - 审计日志
export const ADMIN_AUDIT = {
	LIST: `${API_BASE}/admin/audit/list`,
	OPEN_THE_DOOR: `${API_BASE}/admin/audit/open_the_door`
} as const;

// 管理员 - 举报审核
export const ADMIN_REPORTS = {
	SEARCH: `${API_BASE}/admin/reports/search`,
	HANDLE: `${API_BASE}/admin/reports/handle`
} as const;

// 管理员 - 标签管理
export const ADMIN_TAGS = {
	SEARCH: `${API_BASE}/admin/tags/search`,
	DELETE: `${API_BASE}/admin/tags/delete`
} as const;

// ============================================================================
// 所有端点导出
// ============================================================================
export const ENDPOINTS = {
	AUTH,
	USERS,
	MOMENTS,
	COMMENTS,
	PRIVATE_CHATS,
	PRIVATE_MESSAGES,
	GROUPS,
	GROUP_MESSAGES,
	FEED,
	SEARCH,
	MAP,
	SHARE,
	MEDIA,
	NOTIFICATIONS,
	REPORTS,
	SYSTEM,
	ADMIN_RBAC,
	ADMIN_APP_CONFIG,
	ADMIN_USERS,
	ADMIN_GROUPS,
	ADMIN_ANNOUNCEMENTS,
	ADMIN_AUDIT,
	ADMIN_REPORTS,
	ADMIN_TAGS
} as const;
