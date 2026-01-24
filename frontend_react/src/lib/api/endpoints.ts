/**
 * API 路径常量定义
 * 禁止在代码中硬编码 API 路径，必须通过此文件引用
 */

const API_BASE = '/api/v1';

export const ENDPOINTS = {
  // ============================================================================
  // 认证 API
  // ============================================================================
  AUTH: {
    LOGIN: `${API_BASE}/auth/login`,
    REGISTER: `${API_BASE}/auth/register`,
    LOGOUT: `${API_BASE}/auth/logout`,
    GUEST: `${API_BASE}/auth/guest`,
    REFRESH: `${API_BASE}/auth/refresh`,
    CHECK_STATUS: `${API_BASE}/auth/status`,
    CHANGE_PASSWORD: `${API_BASE}/auth/password`,
    REQUEST_RESET: `${API_BASE}/auth/password/reset/request`,
    VERIFY_RESET: `${API_BASE}/auth/password/reset/verify`,
    RESET_PASSWORD: `${API_BASE}/auth/password/reset`,
    SESSIONS: `${API_BASE}/auth/sessions`,
    REVOKE_SESSION: (sessionId: string) => `${API_BASE}/auth/sessions/${sessionId}`,
  },

  // ============================================================================
  // 用户 API
  // ============================================================================
  USER: {
    // 搜索
    SEARCH_BY_KEYWORD: `${API_BASE}/user/search/keyword`,
    SEARCH_BY_TAG: `${API_BASE}/user/search/tag`,
    SEARCH_BY_LOCATION: `${API_BASE}/user/search/location`,
    // 详情
    DETAIL: (userId: number) => `${API_BASE}/user/${userId}`,
    STATS: (userId: number) => `${API_BASE}/user/${userId}/stats`,
    FRIENDS: (userId: number) => `${API_BASE}/user/${userId}/friends`,
    FOLLOWING: (userId: number) => `${API_BASE}/user/${userId}/following`,
    FOLLOWERS: (userId: number) => `${API_BASE}/user/${userId}/followers`,
    GROUPS: (userId: number) => `${API_BASE}/user/${userId}/groups`,
    MOMENTS: (userId: number) => `${API_BASE}/user/${userId}/moments`,
    // 当前用户
    ME: `${API_BASE}/user/me`,
    ME_PROFILE: `${API_BASE}/user/me/profile`,
    ME_LOCATION: `${API_BASE}/user/me/location`,
    ME_SETTINGS: `${API_BASE}/user/me/settings`,
    ME_STATS: `${API_BASE}/user/me/stats`,
    ME_FRIENDS: `${API_BASE}/user/me/friends`,
    ME_FOLLOWING: `${API_BASE}/user/me/following`,
    ME_FOLLOWERS: `${API_BASE}/user/me/followers`,
    ME_BLOCKED: `${API_BASE}/user/me/blocked`,
    ME_GROUPS: `${API_BASE}/user/me/groups`,
    ME_MOMENTS: `${API_BASE}/user/me/moments`,
    ME_FAVORITES: `${API_BASE}/user/me/favorites`,
    // 关系操作
    FOLLOW: (userId: number) => `${API_BASE}/user/${userId}/follow`,
    UNFOLLOW: (userId: number) => `${API_BASE}/user/${userId}/unfollow`,
    BLOCK: (userId: number) => `${API_BASE}/user/${userId}/block`,
    UNBLOCK: (userId: number) => `${API_BASE}/user/${userId}/unblock`,
    REMARK: (userId: number) => `${API_BASE}/user/${userId}/remark`,
    // 账号操作
    UPGRADE: `${API_BASE}/user/me/upgrade`,
    DELETE: `${API_BASE}/user/me`,
  },

  // ============================================================================
  // 瞬间 API
  // ============================================================================
  MOMENTS: {
    // 搜索
    SEARCH_BY_KEYWORD: `${API_BASE}/moments/search/keyword`,
    SEARCH_BY_TAG: `${API_BASE}/moments/search/tag`,
    SEARCH_BY_LOCATION: `${API_BASE}/moments/search/location`,
    // CRUD
    CREATE: `${API_BASE}/moments`,
    DETAIL: (momentId: string) => `${API_BASE}/moments/${momentId}`,
    UPDATE: (momentId: string) => `${API_BASE}/moments/${momentId}`,
    DELETE: (momentId: string) => `${API_BASE}/moments/${momentId}`,
    // 互动
    LIKE: (momentId: string) => `${API_BASE}/moments/${momentId}/like`,
    UNLIKE: (momentId: string) => `${API_BASE}/moments/${momentId}/unlike`,
    FAVORITE: (momentId: string) => `${API_BASE}/moments/${momentId}/favorite`,
    UNFAVORITE: (momentId: string) => `${API_BASE}/moments/${momentId}/unfavorite`,
    LIKES: (momentId: string) => `${API_BASE}/moments/${momentId}/likes`,
    // 评论
    COMMENTS: (momentId: string) => `${API_BASE}/moments/${momentId}/comments`,
  },

  // ============================================================================
  // 评论 API
  // ============================================================================
  COMMENTS: {
    CREATE: `${API_BASE}/comments`,
    DELETE: (commentId: string) => `${API_BASE}/comments/${commentId}`,
    REPLIES: (commentId: string) => `${API_BASE}/comments/${commentId}/replies`,
    LIKE: (commentId: string) => `${API_BASE}/comments/${commentId}/like`,
    UNLIKE: (commentId: string) => `${API_BASE}/comments/${commentId}/unlike`,
    LIKES: (commentId: string) => `${API_BASE}/comments/${commentId}/likes`,
  },

  // ============================================================================
  // 私聊 API
  // ============================================================================
  PRIVATE_CHATS: {
    LIST: `${API_BASE}/private_chats`,
    DETAIL: (userId: number) => `${API_BASE}/private_chats/${userId}`,
    SETTINGS: (userId: number) => `${API_BASE}/private_chats/${userId}/settings`,
    DELETE: (userId: number) => `${API_BASE}/private_chats/${userId}`,
  },

  PRIVATE_MESSAGES: {
    HISTORY: (userId: number) => `${API_BASE}/private_messages/${userId}`,
    SEND: `${API_BASE}/private_messages`,
    UPDATE: (messageId: string) => `${API_BASE}/private_messages/${messageId}`,
    RECALL: (messageId: string) => `${API_BASE}/private_messages/${messageId}/recall`,
    DELETE: (messageId: string) => `${API_BASE}/private_messages/${messageId}`,
    MARK_READ: (userId: number) => `${API_BASE}/private_messages/${userId}/read`,
    REACTIONS: (messageId: string) => `${API_BASE}/private_messages/${messageId}/reactions`,
    ADD_REACTION: (messageId: string) => `${API_BASE}/private_messages/${messageId}/reactions`,
    REMOVE_REACTION: (messageId: string, type: string) =>
      `${API_BASE}/private_messages/${messageId}/reactions/${type}`,
  },

  // ============================================================================
  // 群组 API
  // ============================================================================
  GROUP_CHATS: {
    // 搜索
    SEARCH_BY_KEYWORD: `${API_BASE}/group_chats/search/keyword`,
    SEARCH_BY_TAG: `${API_BASE}/group_chats/search/tag`,
    SEARCH_BY_LOCATION: `${API_BASE}/group_chats/search/location`,
    // CRUD
    CREATE: `${API_BASE}/group_chats`,
    DETAIL: (groupId: string) => `${API_BASE}/group_chats/${groupId}`,
    UPDATE: (groupId: string) => `${API_BASE}/group_chats/${groupId}`,
    DELETE: (groupId: string) => `${API_BASE}/group_chats/${groupId}`,
    SETTINGS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/settings`,
    MY_SETTINGS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/my_settings`,
    // 成员管理
    MEMBERS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/members`,
    MEMBER: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}`,
    MEMBER_ROLE: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}/role`,
    MEMBER_MUTE: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}/mute`,
    MEMBER_UNMUTE: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}/unmute`,
    MEMBER_BAN: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}/ban`,
    MEMBER_UNBAN: (groupId: string, userId: number) => `${API_BASE}/group_chats/${groupId}/members/${userId}/unban`,
    BANNED_MEMBERS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/banned`,
    // 加入/退出
    JOIN: (groupId: string) => `${API_BASE}/group_chats/${groupId}/join`,
    LEAVE: (groupId: string) => `${API_BASE}/group_chats/${groupId}/leave`,
    APPLY: (groupId: string) => `${API_BASE}/group_chats/${groupId}/apply`,
    INVITE: (groupId: string) => `${API_BASE}/group_chats/${groupId}/invite`,
    TRANSFER: (groupId: string) => `${API_BASE}/group_chats/${groupId}/transfer`,
    // 申请管理
    APPLICATIONS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/applications`,
    APPROVE_APPLICATION: (applicationId: string) => `${API_BASE}/group_chats/applications/${applicationId}/approve`,
    REJECT_APPLICATION: (applicationId: string) => `${API_BASE}/group_chats/applications/${applicationId}/reject`,
    // 邀请管理
    MY_INVITES: `${API_BASE}/group_chats/invites`,
    ACCEPT_INVITE: (inviteId: string) => `${API_BASE}/group_chats/invites/${inviteId}/accept`,
    REJECT_INVITE: (inviteId: string) => `${API_BASE}/group_chats/invites/${inviteId}/reject`,
    // 公告
    ANNOUNCEMENTS: (groupId: string) => `${API_BASE}/group_chats/${groupId}/announcements`,
    ANNOUNCEMENT: (announcementId: string) => `${API_BASE}/group_chats/announcements/${announcementId}`,
  },

  GROUP_MESSAGES: {
    HISTORY: (groupId: string) => `${API_BASE}/group_messages/${groupId}`,
    SEND: `${API_BASE}/group_messages`,
    UPDATE: (messageId: string) => `${API_BASE}/group_messages/${messageId}`,
    RECALL: (messageId: string) => `${API_BASE}/group_messages/${messageId}/recall`,
    DELETE: (messageId: string) => `${API_BASE}/group_messages/${messageId}`,
    MARK_READ: (groupId: string) => `${API_BASE}/group_messages/${groupId}/read`,
    READ_STATUS: (messageId: string) => `${API_BASE}/group_messages/${messageId}/read_status`,
    REACTIONS: (messageId: string) => `${API_BASE}/group_messages/${messageId}/reactions`,
    ADD_REACTION: (messageId: string) => `${API_BASE}/group_messages/${messageId}/reactions`,
    REMOVE_REACTION: (messageId: string, type: string) =>
      `${API_BASE}/group_messages/${messageId}/reactions/${type}`,
  },

  // ============================================================================
  // 信息流 API
  // ============================================================================
  FEED: {
    MOMENTS: `${API_BASE}/feed/moments`,
    GROUPS: `${API_BASE}/feed/groups`,
    USERS: `${API_BASE}/feed/users`,
    TRENDING: `${API_BASE}/feed/trending`,
    FEEDBACK: `${API_BASE}/feed/feedback`,
  },

  // ============================================================================
  // 标签 API
  // ============================================================================
  TAGS: {
    TRENDING: `${API_BASE}/tags/trending`,
    SUGGEST: `${API_BASE}/tags/suggest`,
  },

  // ============================================================================
  // 举报 API
  // ============================================================================
  REPORTS: {
    CREATE: `${API_BASE}/reports`,
    MY_REPORTS: `${API_BASE}/reports/me`,
  },

  // ============================================================================
  // 通知 API
  // ============================================================================
  NOTIFICATIONS: {
    LIST: `${API_BASE}/notifications`,
    MARK_READ: `${API_BASE}/notifications/read`,
    MARK_ALL_READ: `${API_BASE}/notifications/read/all`,
    UNREAD_COUNT: `${API_BASE}/notifications/unread`,
  },

  SUBSCRIPTIONS: {
    GET: `${API_BASE}/subscriptions`,
    UPDATE: `${API_BASE}/subscriptions`,
  },

  DEVICE_TOKENS: {
    REGISTER: `${API_BASE}/device_tokens`,
    UNREGISTER: `${API_BASE}/device_tokens`,
    LIST: `${API_BASE}/device_tokens`,
  },

  // ============================================================================
  // 地图 API
  // ============================================================================
  MAP: {
    RENDER: `${API_BASE}/map/render`,
  },

  // ============================================================================
  // 媒体上传 API
  // ============================================================================
  MEDIA: {
    PRESIGN: `${API_BASE}/media/presign`,
    COMPLETE: `${API_BASE}/media/complete`,
    INIT_CHUNKED: `${API_BASE}/media/chunked/init`,
    FINISH_CHUNKED: `${API_BASE}/media/chunked/finish`,
    CHECK_CHUNKED: `${API_BASE}/media/chunked/check`,
  },

  // ============================================================================
  // 系统 API
  // ============================================================================
  SYSTEM: {
    HEALTH: `${API_BASE}/system/health`,
    GLOBAL_SETTINGS: `${API_BASE}/global_settings`,
  },

  // ============================================================================
  // 公告 API
  // ============================================================================
  ANNOUNCEMENTS: {
    LIST: `${API_BASE}/announcements`,
    DETAIL: (announcementId: string) => `${API_BASE}/announcements/${announcementId}`,
  },

  // ============================================================================
  // 管理员 API
  // ============================================================================
  ADMIN: {
    // RBAC
    ME: `${API_BASE}/admin/rbac/me`,
    PERMISSIONS: `${API_BASE}/admin/rbac/permissions`,
    ROLES: `${API_BASE}/admin/rbac/roles`,
    ROLE: (roleId: string) => `${API_BASE}/admin/rbac/roles/${roleId}`,
    ADMINS: `${API_BASE}/admin/rbac/admins`,
    ADMIN: (userId: number) => `${API_BASE}/admin/rbac/admins/${userId}`,
    ADMIN_ROLES: (userId: number) => `${API_BASE}/admin/rbac/admins/${userId}/roles`,
    // 用户管理
    USERS_SEARCH: `${API_BASE}/admin/users/search`,
    USER_DETAIL: (userId: number) => `${API_BASE}/admin/users/${userId}`,
    USER_BAN: (userId: number) => `${API_BASE}/admin/users/${userId}/ban`,
    USER_UNBAN: (userId: number) => `${API_BASE}/admin/users/${userId}/unban`,
    // 群组管理
    GROUPS_SEARCH: `${API_BASE}/admin/groups/search`,
    GROUP_DETAIL: (groupId: string) => `${API_BASE}/admin/groups/${groupId}`,
    GROUP_DISMISS: (groupId: string) => `${API_BASE}/admin/groups/${groupId}/dismiss`,
    // 举报管理
    REPORTS: `${API_BASE}/admin/reports`,
    REPORT: (reportId: string) => `${API_BASE}/admin/reports/${reportId}`,
    REPORT_HANDLE: (reportId: string) => `${API_BASE}/admin/reports/${reportId}/handle`,
    // 公告管理
    ANNOUNCEMENTS: `${API_BASE}/admin/announcements`,
    ANNOUNCEMENT: (announcementId: string) => `${API_BASE}/admin/announcements/${announcementId}`,
    // 审计日志
    AUDIT_LOGS: `${API_BASE}/admin/audit/logs`,
    AUDIT_EXPORT: `${API_BASE}/admin/audit/export`,
    // 系统配置
    CONFIG: `${API_BASE}/admin/config`,
    CONFIG_UPLOAD: `${API_BASE}/admin/config/upload`,
    CONFIG_DEFAULTS: `${API_BASE}/admin/config/defaults`,
    CONFIG_FEATURES: `${API_BASE}/admin/config/features`,
    CONFIG_CONTACT: `${API_BASE}/admin/config/contact`,
    CONFIG_LIMITS: `${API_BASE}/admin/config/limits`,
  },
} as const;
