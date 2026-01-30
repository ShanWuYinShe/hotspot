/**
 * API 路径常量
 *
 * 严格遵循 @spec/api_spec.mdc 规范
 * 所有端点统一使用 /api/v1/ 前缀
 */

export const API_VERSION = 'v1';
export const API_BASE = `/api/${API_VERSION}`;

// ============================================================================
// 认证 API
// ============================================================================
export const AUTH = {
	REGISTER: `${API_BASE}/auth/register`,
	LOGIN: `${API_BASE}/auth/login`,
	LOGOUT: `${API_BASE}/auth/logout`,
	GUEST: `${API_BASE}/auth/guest`,
	REFRESH_TOKEN: `${API_BASE}/auth/refresh_token`,
	CHECK_STATUS: `${API_BASE}/auth/check_status`,
	SEND_VERIFICATION_CODE: `${API_BASE}/auth/send_verification_code`,
	VERIFY_CODE: `${API_BASE}/auth/verify_code`,
	REQUEST_PASSWORD_RESET: `${API_BASE}/auth/request_password_reset`,
	VERIFY_RESET_CODE: `${API_BASE}/auth/verify_reset_code`,
	RESET_PASSWORD: `${API_BASE}/auth/reset_password`,
	ME: {
		CHANGE_PASSWORD: `${API_BASE}/auth/me/change_password`,
		SESSIONS: `${API_BASE}/auth/me/sessions`,
		REVOKE_SESSION: `${API_BASE}/auth/me/sessions/revoke`
	}
} as const;

// ============================================================================
// 用户 API
// ============================================================================
export const USER = {
	// 搜索
	SEARCH_BY_KEYWORD: `${API_BASE}/user/search_by_keyword`,
	SEARCH_BY_TAG: `${API_BASE}/user/search_by_tag`,
	SEARCH_BY_LOCATION: `${API_BASE}/user/search_by_location`,
	SUGGEST: `${API_BASE}/user/suggest`,
	// 获取指定用户信息
	OPEN_THE_DOOR: `${API_BASE}/user/open_the_door`,
	RELATIONSHIP: `${API_BASE}/user/relationship`,
	STATS: `${API_BASE}/user/stats`,
	FRIENDS: `${API_BASE}/user/friends`,
	FOLLOWING: `${API_BASE}/user/following`,
	FOLLOWERS: `${API_BASE}/user/followers`,
	GROUPS: `${API_BASE}/user/groups`,
	LIKED_MOMENTS: `${API_BASE}/user/liked_moments`,
	MOMENTS: `${API_BASE}/user/moments`,
	// 当前用户
	ME: {
		PROFILE: `${API_BASE}/user/me/profile`,
		PROFILE_UPDATE: `${API_BASE}/user/me/profile/update`,
		LOCATION: `${API_BASE}/user/me/location`,
		UPGRADE: `${API_BASE}/user/me/upgrade`,
		STATS: `${API_BASE}/user/me/stats`,
		SETTINGS: `${API_BASE}/user/me/settings`,
		SETTINGS_UPDATE: `${API_BASE}/user/me/settings/update`,
		DELETE: `${API_BASE}/user/me/delete`,
		FRIENDS: `${API_BASE}/user/me/friends`,
		FOLLOWING: `${API_BASE}/user/me/following`,
		FOLLOWERS: `${API_BASE}/user/me/followers`,
		BLOCKED: `${API_BASE}/user/me/blocked`,
		FOLLOW_REQUESTS: `${API_BASE}/user/me/follow_requests`,
		FOLLOW_REQUESTS_HANDLE: `${API_BASE}/user/me/follow_requests/handle`,
		// 关注关系操作
		FOLLOW: `${API_BASE}/user/me/follow`,
		UNFOLLOW: `${API_BASE}/user/me/unfollow`,
		BLOCK: `${API_BASE}/user/me/block`,
		UNBLOCK: `${API_BASE}/user/me/unblock`,
		REMARK: `${API_BASE}/user/me/remark`
	}
} as const;

// ============================================================================
// 生活瞬间 API
// ============================================================================
export const MOMENTS = {
	// 搜索
	SEARCH_BY_KEYWORD: `${API_BASE}/moments/search_by_keyword`,
	SEARCH_BY_TAG: `${API_BASE}/moments/search_by_tag`,
	SEARCH_BY_LOCATION: `${API_BASE}/moments/search_by_location`,
	SUGGEST: `${API_BASE}/moments/suggest`,
	// 获取指定瞬间信息
	OPEN_THE_DOOR: `${API_BASE}/moments/open_the_door`,
	INTERACTION_STATE: `${API_BASE}/moments/interaction_state`,
	LIKES: `${API_BASE}/moments/likes`,
	// 操作
	CREATE: `${API_BASE}/moments/create`,
	UPDATE: `${API_BASE}/moments/update`,
	DELETE: `${API_BASE}/moments/delete`,
	LIKE: `${API_BASE}/moments/like`,
	UNLIKE: `${API_BASE}/moments/unlike`,
	FAVORITE: `${API_BASE}/moments/favorite`,
	UNFAVORITE: `${API_BASE}/moments/unfavorite`,
	// 当前用户
	ME: {
		LIST: `${API_BASE}/moments/me`,
		FAVORITES: `${API_BASE}/moments/me/favorites`,
		LIKED: `${API_BASE}/moments/me/liked`,
		FAVORITED: `${API_BASE}/moments/me/favorited`
	}
} as const;

// ============================================================================
// 评论 API
// ============================================================================
export const COMMENTS = {
	LIST: `${API_BASE}/comments`,
	REPLIES: `${API_BASE}/comments/replies`,
	LIKES: `${API_BASE}/comments/likes`,
	CREATE: `${API_BASE}/comments/create`,
	DELETE: `${API_BASE}/comments/delete`,
	LIKE: `${API_BASE}/comments/like`,
	UNLIKE: `${API_BASE}/comments/unlike`
} as const;

// ============================================================================
// 私聊 API
// ============================================================================
export const PRIVATE_CHATS = {
	ME: `${API_BASE}/private_chats/me`,
	OPEN_THE_DOOR: `${API_BASE}/private_chats/open_the_door`,
	UPDATE_SETTINGS: `${API_BASE}/private_chats/update_settings`,
	DELETE: `${API_BASE}/private_chats/delete`
} as const;

// ============================================================================
// 私聊消息 API
// ============================================================================
export const PRIVATE_MESSAGES = {
	SEND: `${API_BASE}/private_messages/send`,
	HISTORY: `${API_BASE}/private_messages/history`,
	UPDATE: `${API_BASE}/private_messages/update`,
	RECALL: `${API_BASE}/private_messages/recall`,
	DELETE: `${API_BASE}/private_messages/delete`,
	MARK_AS_READ: `${API_BASE}/private_messages/mark_as_read`,
	ADD_REACTION: `${API_BASE}/private_messages/add_reaction`,
	REMOVE_REACTION: `${API_BASE}/private_messages/remove_reaction`,
	REACTIONS: `${API_BASE}/private_messages/reactions`
} as const;

// ============================================================================
// 群组 API
// ============================================================================
export const GROUPS = {
	// 搜索
	SEARCH_BY_KEYWORD: `${API_BASE}/group_chats/search_by_keyword`,
	SEARCH_BY_TAG: `${API_BASE}/group_chats/search_by_tag`,
	SEARCH_BY_LOCATION: `${API_BASE}/group_chats/search_by_location`,
	SUGGEST: `${API_BASE}/group_chats/suggest`,
	// 获取指定群组信息
	OPEN_THE_DOOR: `${API_BASE}/group_chats/open_the_door`,
	MY_STATE: `${API_BASE}/group_chats/my_state`,
	SETTINGS: `${API_BASE}/group_chats/settings`,
	// 操作
	CREATE: `${API_BASE}/group_chats/create`,
	UPDATE: `${API_BASE}/group_chats/update`,
	UPDATE_MY_SETTINGS: `${API_BASE}/group_chats/update_my_settings`,
	DELETE: `${API_BASE}/group_chats/delete`,
	TRANSFER_OWNERSHIP: `${API_BASE}/group_chats/transfer_ownership`,
	// 成员管理
	MEMBERS: {
		LIST: `${API_BASE}/group_chats/members`,
		INVITE: `${API_BASE}/group_chats/members/invite`,
		JOIN: `${API_BASE}/group_chats/members/join`,
		APPLY: `${API_BASE}/group_chats/members/apply`,
		SET_ROLE: `${API_BASE}/group_chats/members/set_role`,
		MUTE: `${API_BASE}/group_chats/members/mute`,
		UNMUTE: `${API_BASE}/group_chats/members/unmute`,
		BAN: `${API_BASE}/group_chats/members/ban`,
		UNBAN: `${API_BASE}/group_chats/members/unban`,
		BANNED: `${API_BASE}/group_chats/members/banned`,
		REMOVE_MEMBER: `${API_BASE}/group_chats/members/remove_member`
	},
	// 申请管理
	APPLICATIONS: {
		LIST: `${API_BASE}/group_chats/applications`,
		APPROVE: `${API_BASE}/group_chats/applications/approve`,
		REJECT: `${API_BASE}/group_chats/applications/reject`
	},
	// 群公告
	ANNOUNCEMENTS: {
		LIST: `${API_BASE}/group_chats/announcements`,
		CREATE: `${API_BASE}/group_chats/announcements/create`,
		UPDATE: `${API_BASE}/group_chats/announcements/update`,
		DELETE: `${API_BASE}/group_chats/announcements/delete`
	},
	// 当前用户
	ME: {
		LIST: `${API_BASE}/group_chats/me`,
		CANCEL_APPLICATION: `${API_BASE}/group_chats/me/applications/cancel`,
		INVITES: `${API_BASE}/group_chats/me/invites`,
		ACCEPT_INVITE: `${API_BASE}/group_chats/me/invites/accept`,
		REJECT_INVITE: `${API_BASE}/group_chats/me/invites/reject`,
		LEAVE: `${API_BASE}/group_chats/me/leave`
	}
} as const;

// ============================================================================
// 群聊消息 API
// ============================================================================
export const GROUP_MESSAGES = {
	SEND: `${API_BASE}/group_messages/send`,
	HISTORY: `${API_BASE}/group_messages/history`,
	UPDATE: `${API_BASE}/group_messages/update`,
	RECALL: `${API_BASE}/group_messages/recall`,
	DELETE: `${API_BASE}/group_messages/delete`,
	MARK_AS_READ: `${API_BASE}/group_messages/mark_as_read`,
	READ_STATUS: `${API_BASE}/group_messages/read_status`,
	ADD_REACTION: `${API_BASE}/group_messages/add_reaction`,
	REMOVE_REACTION: `${API_BASE}/group_messages/remove_reaction`,
	REACTIONS: `${API_BASE}/group_messages/reactions`
} as const;

// ============================================================================
// 消息通用 API
// ============================================================================
export const MESSAGES = {
	MARK_ALL_AS_READ: `${API_BASE}/messages/mark_all_as_read`
} as const;

// ============================================================================
// 信息流 API
// ============================================================================
export const FEED = {
	MOMENTS: `${API_BASE}/feed/moments`,
	GROUPS: `${API_BASE}/feed/groups`,
	USERS: `${API_BASE}/feed/users`,
	TRENDING: `${API_BASE}/feed/trending`,
	FEEDBACK: `${API_BASE}/feed/feedback`,
	SUGGEST: `${API_BASE}/feed/suggest`
} as const;

// ============================================================================
// 标签 API
// ============================================================================
export const TAGS = {
	TRENDING: `${API_BASE}/tags/trending`,
	SUGGEST: `${API_BASE}/tags/suggest`
} as const;

// ============================================================================
// 地图 API
// ============================================================================
export const MAP = {
	RENDER: `${API_BASE}/map/render`
} as const;

// ============================================================================
// 媒体上传 API
// ============================================================================
export const MEDIA = {
	PRESIGN_UPLOAD: `${API_BASE}/media/presign_upload`,
	COMPLETE_UPLOAD: `${API_BASE}/media/complete_upload`
} as const;

// ============================================================================
// 通知中心 API
// ============================================================================
export const NOTIFICATIONS = {
	LIST: `${API_BASE}/notifications`,
	UNREAD_COUNT: `${API_BASE}/notifications/unread_count`,
	MARK_READ: `${API_BASE}/notifications/mark_read`
} as const;

// ============================================================================
// 订阅设置 API
// ============================================================================
export const SUBSCRIPTIONS = {
	ME: `${API_BASE}/subscriptions/me`,
	UPDATE: `${API_BASE}/subscriptions/me/update`
} as const;

// ============================================================================
// 设备推送 API
// ============================================================================
export const DEVICE_TOKENS = {
	ME: `${API_BASE}/device_tokens/me`,
	REGISTER: `${API_BASE}/device_tokens/me/register`,
	UNREGISTER: `${API_BASE}/device_tokens/me/unregister`
} as const;

// ============================================================================
// 举报 API
// ============================================================================
export const REPORTS = {
	CREATE: `${API_BASE}/reports/create`,
	ME: `${API_BASE}/reports/me`
} as const;

// ============================================================================
// 系统 API
// ============================================================================
export const SYSTEM = {
	HEALTH: `${API_BASE}/system/health`
} as const;

// ============================================================================
// 全局配置 API
// ============================================================================
export const GLOBAL_SETTINGS = {
	FETCH: `${API_BASE}/global_settings/fetch`
} as const;

// ============================================================================
// 系统公告 API（公开）
// ============================================================================
export const ANNOUNCEMENTS = {
	LIST: `${API_BASE}/announcements`
} as const;

// ============================================================================
// 管理员 API
// ============================================================================

// 管理员 - 全局配置
export const ADMIN_GLOBAL_SETTINGS = {
	FETCH: `${API_BASE}/admin/global_settings`,
	UPDATE_UPLOAD: `${API_BASE}/admin/global_settings/update_upload`,
	UPDATE_DEFAULTS: `${API_BASE}/admin/global_settings/update_defaults`,
	UPDATE_FEATURES: `${API_BASE}/admin/global_settings/update_features`,
	UPDATE_CONTACT: `${API_BASE}/admin/global_settings/update_contact`,
	UPDATE_LIMITS: `${API_BASE}/admin/global_settings/update_limits`
} as const;

// 管理员 - RBAC
export const ADMIN_RBAC = {
	ME: `${API_BASE}/admin/rbac/me`,
	PERMISSIONS: `${API_BASE}/admin/rbac/permissions`,
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

// 管理员 - 用户管理
export const ADMIN_USERS = {
	SEARCH: `${API_BASE}/admin/users/search`,
	DETAIL: `${API_BASE}/admin/users/detail`,
	BAN: `${API_BASE}/admin/users/ban`,
	UNBAN: `${API_BASE}/admin/users/unban`
} as const;

// 管理员 - 群组管理
export const ADMIN_GROUPS = {
	SEARCH: `${API_BASE}/admin/groups/search`,
	DETAIL: `${API_BASE}/admin/groups/detail`,
	DISMISS: `${API_BASE}/admin/groups/dismiss`
} as const;

// 管理员 - 系统公告
export const ADMIN_ANNOUNCEMENTS = {
	LIST: `${API_BASE}/admin/announcements/list`,
	CREATE: `${API_BASE}/admin/announcements/create`,
	DELETE: `${API_BASE}/admin/announcements/delete`
} as const;

// 管理员 - 审计日志
export const ADMIN_AUDIT = {
	LIST: `${API_BASE}/admin/audit/list`,
	DETAIL: `${API_BASE}/admin/audit/detail`,
	EXPORT: `${API_BASE}/admin/audit/export`,
	EXPORT_STATUS: `${API_BASE}/admin/audit/export/status`
} as const;

// 管理员 - 举报审核
export const ADMIN_REPORTS = {
	LIST: `${API_BASE}/admin/reports`,
	HANDLE: `${API_BASE}/admin/reports/handle`
} as const;

// 管理员 - 标签管理
export const ADMIN_TAGS = {
	DELETE: `${API_BASE}/admin/tags/delete`
} as const;

// ============================================================================
// 所有端点导出
// ============================================================================
export const ENDPOINTS = {
	AUTH,
	USER,
	MOMENTS,
	COMMENTS,
	PRIVATE_CHATS,
	PRIVATE_MESSAGES,
	GROUPS,
	GROUP_MESSAGES,
	MESSAGES,
	FEED,
	TAGS,
	MAP,
	MEDIA,
	NOTIFICATIONS,
	SUBSCRIPTIONS,
	DEVICE_TOKENS,
	REPORTS,
	SYSTEM,
	GLOBAL_SETTINGS,
	ANNOUNCEMENTS,
	ADMIN_GLOBAL_SETTINGS,
	ADMIN_RBAC,
	ADMIN_USERS,
	ADMIN_GROUPS,
	ADMIN_ANNOUNCEMENTS,
	ADMIN_AUDIT,
	ADMIN_REPORTS,
	ADMIN_TAGS
} as const;
