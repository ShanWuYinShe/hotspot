/**
 * API 路径常量
 */

export const API_VERSION = 'v1'

// API 路径前缀
export const API = {
  PUBLIC: `/api/${API_VERSION}/public`,
  PROTECTED: `/api/${API_VERSION}/protected`,
  ADMIN: `/api/${API_VERSION}/admin`
} as const

// 认证 API
export const AUTH = {
  // 公开 API
  REGISTER: `${API.PUBLIC}/auth/register`,
  LOGIN: `${API.PUBLIC}/auth/login`,
  GUEST: `${API.PUBLIC}/auth/guest`,
  REFRESH_TOKEN: `${API.PUBLIC}/auth/refresh_token`,
  REQUEST_PASSWORD_RESET: `${API.PUBLIC}/auth/request_password_reset`,
  VERIFY_RESET_CODE: `${API.PUBLIC}/auth/verify_reset_code`,
  RESET_PASSWORD: `${API.PUBLIC}/auth/reset_password`,
  // 受保护 API
  LOGOUT: `${API.PROTECTED}/auth/logout`,
  CHECK_STATUS: `${API.PROTECTED}/auth/check_status`,
  // 当前用户认证相关 (protected, /me/ 路径)
  ME: {
    CHANGE_PASSWORD: `${API.PROTECTED}/auth/me/change_password`,
    SESSIONS: `${API.PROTECTED}/auth/me/sessions`,
    REVOKE_SESSION: `${API.PROTECTED}/auth/me/sessions/revoke`
  }
} as const

// 用户 API
export const USER = {
  // 搜索 (public)
  SEARCH_BY_KEYWORD: `${API.PUBLIC}/user/search_by_keyword`,
  SEARCH_BY_TAG: `${API.PUBLIC}/user/search_by_tag`,
  SEARCH_BY_LOCATION: `${API.PUBLIC}/user/search_by_location`,
  // 获取指定用户信息 (public)
  OPEN_THE_DOOR: `${API.PUBLIC}/user/open_the_door`,
  STATS: `${API.PUBLIC}/user/stats`,  // 获取指定用户统计数据
  // 获取指定用户列表（受隐私设置控制）
  FRIENDS: `${API.PUBLIC}/user/friends`,
  FOLLOWING: `${API.PUBLIC}/user/following`,
  FOLLOWERS: `${API.PUBLIC}/user/followers`,
  GROUPS: `${API.PUBLIC}/user/groups`,
  LIKED_MOMENTS: `${API.PUBLIC}/user/liked_moments`,
  MOMENTS: `${API.PUBLIC}/user/moments`,
  // 获取指定用户关系 (protected - 需要登录才能获取)
  RELATIONSHIP: `${API.PROTECTED}/user/relationship`,
  // 当前用户操作 (protected, /me/ 路径)
  ME: `${API.PROTECTED}/user/me`,
  PROFILE: `${API.PROTECTED}/user/me/profile`,
  PROFILE_UPDATE: `${API.PROTECTED}/user/me/profile/update`,
  LOCATION: `${API.PROTECTED}/user/me/location`,
  UPGRADE: `${API.PROTECTED}/user/me/upgrade`,
  MY_STATS: `${API.PROTECTED}/user/me/stats`,
  SETTINGS: `${API.PROTECTED}/user/me/settings`,
  SETTINGS_UPDATE: `${API.PROTECTED}/user/me/settings/update`,
  DELETE: `${API.PROTECTED}/user/me/delete`,
  MY_FRIENDS: `${API.PROTECTED}/user/me/friends`,
  MY_FOLLOWING: `${API.PROTECTED}/user/me/following`,
  MY_FOLLOWERS: `${API.PROTECTED}/user/me/followers`,
  MY_BLOCKED: `${API.PROTECTED}/user/me/blocked`,
  // 关注关系操作 (protected, /me 子路径)
  FOLLOW: `${API.PROTECTED}/user/me/follow`,
  UNFOLLOW: `${API.PROTECTED}/user/me/unfollow`,
  BLOCK: `${API.PROTECTED}/user/me/block`,
  UNBLOCK: `${API.PROTECTED}/user/me/unblock`,
  REMARK: `${API.PROTECTED}/user/me/remark`
} as const

// 生活瞬间 API
export const MOMENTS = {
  // 搜索 (public)
  SEARCH_BY_KEYWORD: `${API.PUBLIC}/moments/search_by_keyword`,
  SEARCH_BY_TAG: `${API.PUBLIC}/moments/search_by_tag`,
  SEARCH_BY_LOCATION: `${API.PUBLIC}/moments/search_by_location`,
  // 获取指定瞬间信息 (public)
  OPEN_THE_DOOR: `${API.PUBLIC}/moments/open_the_door`,
  COMMENTS: `${API.PUBLIC}/moments/comments`,
  LIKES: `${API.PUBLIC}/moments/likes`,
  // 操作 (protected)
  CREATE: `${API.PROTECTED}/moments/create`,
  UPDATE: `${API.PROTECTED}/moments/update`,
  DELETE: `${API.PROTECTED}/moments/delete`,
  LIKE: `${API.PROTECTED}/moments/like`,
  UNLIKE: `${API.PROTECTED}/moments/unlike`,
  FAVORITE: `${API.PROTECTED}/moments/favorite`,
  UNFAVORITE: `${API.PROTECTED}/moments/unfavorite`,
  // 当前用户相关 (protected, /me/ 路径)
  MY: `${API.PROTECTED}/moments/me`,
  MY_FAVORITES: `${API.PROTECTED}/moments/me/favorites`,
  LIKED: `${API.PROTECTED}/moments/me/liked`,
  FAVORITED: `${API.PROTECTED}/moments/me/favorited`
} as const

// 评论 API
export const COMMENTS = {
  CREATE: `${API.PROTECTED}/comments/create`,
  DELETE: `${API.PROTECTED}/comments/delete`,
  LIKE: `${API.PROTECTED}/comments/like`,
  UNLIKE: `${API.PROTECTED}/comments/unlike`,
  LIST: `${API.PUBLIC}/comments`,
  REPLIES: `${API.PUBLIC}/comments/replies`,
  LIKES: `${API.PUBLIC}/comments/likes`
} as const

// 私聊 API
export const PRIVATE_CHATS = {
  CREATE: `${API.PROTECTED}/private_chats/create`,
  OPEN_THE_DOOR: `${API.PROTECTED}/private_chats/open_the_door`,
  UPDATE_SETTINGS: `${API.PROTECTED}/private_chats/update_settings`,
  DELETE: `${API.PROTECTED}/private_chats/delete`,
  // 当前用户私聊 (protected, /me/ 路径)
  ME: `${API.PROTECTED}/private_chats/me`
} as const

// 私聊消息 API
export const PRIVATE_MESSAGES = {
  SEND: `${API.PROTECTED}/private_messages/send`,
  HISTORY: `${API.PROTECTED}/private_messages/history`,
  UPDATE: `${API.PROTECTED}/private_messages/update`,
  RECALL: `${API.PROTECTED}/private_messages/recall`,
  DELETE: `${API.PROTECTED}/private_messages/delete`,
  MARK_AS_READ: `${API.PROTECTED}/private_messages/mark_as_read`,
  ADD_REACTION: `${API.PROTECTED}/private_messages/add_reaction`,
  REMOVE_REACTION: `${API.PROTECTED}/private_messages/remove_reaction`,
  REACTIONS: `${API.PROTECTED}/private_messages/reactions`
} as const

// 群组 API
export const GROUPS = {
  // 搜索 (public)
  SEARCH_BY_KEYWORD: `${API.PUBLIC}/group_chats/search_by_keyword`,
  SEARCH_BY_TAG: `${API.PUBLIC}/group_chats/search_by_tag`,
  SEARCH_BY_LOCATION: `${API.PUBLIC}/group_chats/search_by_location`,
  // 获取指定群组信息 (public)
  OPEN_THE_DOOR: `${API.PUBLIC}/group_chats/open_the_door`,
  SETTINGS: `${API.PUBLIC}/group_chats/settings`,
  // 操作 (protected)
  CREATE: `${API.PROTECTED}/group_chats/create`,
  UPDATE: `${API.PROTECTED}/group_chats/update`,
  DELETE: `${API.PROTECTED}/group_chats/delete`,
  TRANSFER_OWNERSHIP: `${API.PROTECTED}/group_chats/transfer_ownership`,
  // 成员管理
  MEMBERS: {
    LIST: `${API.PUBLIC}/group_chats/members`,
    INVITE: `${API.PROTECTED}/group_chats/members/invite`,
    JOIN: `${API.PROTECTED}/group_chats/members/join`,
    APPLY: `${API.PROTECTED}/group_chats/members/apply`,
    SET_ROLE: `${API.PROTECTED}/group_chats/members/set_role`,
    MUTE: `${API.PROTECTED}/group_chats/members/mute`,
    UNMUTE: `${API.PROTECTED}/group_chats/members/unmute`,
    BAN: `${API.PROTECTED}/group_chats/members/ban`,
    UNBAN: `${API.PROTECTED}/group_chats/members/unban`,
    BANNED: `${API.PROTECTED}/group_chats/members/banned`,
    REMOVE_MEMBER: `${API.PROTECTED}/group_chats/members/remove_member`
  },
  // 申请管理（群管理员操作）
  APPLICATIONS: {
    LIST: `${API.PROTECTED}/group_chats/applications`,
    APPROVE: `${API.PROTECTED}/group_chats/applications/approve`,
    REJECT: `${API.PROTECTED}/group_chats/applications/reject`
  },
  // 当前用户群组相关 (protected, /me/ 路径)
  ME: {
    LIST: `${API.PROTECTED}/group_chats/me`,
    CANCEL_APPLICATION: `${API.PROTECTED}/group_chats/me/applications/cancel`,
    INVITES: `${API.PROTECTED}/group_chats/me/invites`,
    ACCEPT_INVITE: `${API.PROTECTED}/group_chats/me/invites/accept`,
    REJECT_INVITE: `${API.PROTECTED}/group_chats/me/invites/reject`,
    LEAVE: `${API.PROTECTED}/group_chats/me/leave`
  }
} as const

// 群聊消息 API
export const GROUP_MESSAGES = {
  SEND: `${API.PROTECTED}/group_messages/send`,
  HISTORY: `${API.PROTECTED}/group_messages/history`,
  UPDATE: `${API.PROTECTED}/group_messages/update`,
  RECALL: `${API.PROTECTED}/group_messages/recall`,
  DELETE: `${API.PROTECTED}/group_messages/delete`,
  MARK_AS_READ: `${API.PROTECTED}/group_messages/mark_as_read`,
  READ_STATUS: `${API.PROTECTED}/group_messages/read_status`,
  ADD_REACTION: `${API.PROTECTED}/group_messages/add_reaction`,
  REMOVE_REACTION: `${API.PROTECTED}/group_messages/remove_reaction`,
  REACTIONS: `${API.PROTECTED}/group_messages/reactions`
} as const

// 消息通用 API
export const MESSAGES = {
  MARK_ALL_AS_READ: `${API.PROTECTED}/messages/mark_all_as_read`
} as const

// 信息流 API
export const FEED = {
  MOMENTS: `${API.PROTECTED}/feed/moments`,
  GROUPS: `${API.PROTECTED}/feed/groups`,
  USERS: `${API.PROTECTED}/feed/users`,
  TRENDING: `${API.PROTECTED}/feed/trending`,
  FEEDBACK: `${API.PROTECTED}/feed/feedback`
} as const

// 举报 API
export const REPORTS = {
  CREATE: `${API.PROTECTED}/reports/create`,
  // 当前用户举报
  ME: `${API.PROTECTED}/reports/me`,
  // 管理员
  ADMIN_LIST: `${API.ADMIN}/reports`,
  ADMIN_HANDLE: `${API.ADMIN}/reports/handle`
} as const

// 系统配置管理（管理员）
export const ADMIN_CONFIG = {
  CREATE: `${API.ADMIN}/config/create`,
  DETAIL: `${API.ADMIN}/config/detail`,
  LIST: `${API.ADMIN}/config`,
  UPDATE: `${API.ADMIN}/config/update`,
  DELETE: `${API.ADMIN}/config/delete`,
  HISTORY: `${API.ADMIN}/config/history`,
  CHANGES: `${API.ADMIN}/config/changes`
} as const

// 管理员 RBAC API
export const ADMIN_RBAC = {
  ME: `${API.ADMIN}/rbac/me`,
  PERMISSIONS: `${API.ADMIN}/rbac/permissions`,
  ROLES_LIST: `${API.ADMIN}/rbac/roles/list`,
  ROLES_CREATE: `${API.ADMIN}/rbac/roles/create`,
  ROLES_UPDATE: `${API.ADMIN}/rbac/roles/update`,
  ROLES_DELETE: `${API.ADMIN}/rbac/roles/delete`,
  ROLES_SET_PERMISSIONS: `${API.ADMIN}/rbac/roles/set_permissions`,
  ADMINS_LIST: `${API.ADMIN}/rbac/admins/list`,
  ADMINS_GRANT: `${API.ADMIN}/rbac/admins/grant`,
  ADMINS_REVOKE: `${API.ADMIN}/rbac/admins/revoke`
} as const

// 媒体上传 API
export const MEDIA = {
  UPLOAD: `${API.PROTECTED}/media/upload`,
  INITIATE_UPLOAD: `${API.PROTECTED}/media/initiate_upload`,
  UPLOAD_CHUNK: `${API.PROTECTED}/media/upload_chunk`,
  COMPLETE_UPLOAD: `${API.PROTECTED}/media/complete_upload`,
  UPLOAD_STATUS: `${API.PROTECTED}/media/upload_status`,
  ABORT_UPLOAD: `${API.PROTECTED}/media/abort_upload`
} as const

// 标签 API
export const TAGS = {
  TRENDING: `${API.PUBLIC}/tags/trending`,
  SUGGEST: `${API.PUBLIC}/tags/suggest`
} as const

// 通知订阅 API (当前用户, /me/ 路径)
export const SUBSCRIPTIONS = {
  ME: `${API.PROTECTED}/subscriptions/me`,
  UPDATE: `${API.PROTECTED}/subscriptions/me/update`
} as const

// 设备推送 API (当前用户, /me/ 路径)
export const DEVICE_TOKENS = {
  REGISTER: `${API.PROTECTED}/device_tokens/me/register`,
  UNREGISTER: `${API.PROTECTED}/device_tokens/me/unregister`,
  ME: `${API.PROTECTED}/device_tokens/me`
} as const

// 系统 API
export const SYSTEM = {
  HEALTH: `${API.PUBLIC}/system/health`
} as const

// 地图聚合/视口查询 API（用于替代 search_by_location，实现按 zoom/bounds 的瓦片聚合与视口内明细加载）
export const MAP = {
  USERS: {
    CLUSTER: `${API.PUBLIC}/map/users/cluster`,
    SEARCH_BY_BOUNDS: `${API.PUBLIC}/map/users/search_by_bounds`
  },
  GROUPS: {
    CLUSTER: `${API.PUBLIC}/map/groups/cluster`,
    SEARCH_BY_BOUNDS: `${API.PUBLIC}/map/groups/search_by_bounds`
  },
  MOMENTS: {
    CLUSTER: `${API.PUBLIC}/map/moments/cluster`,
    SEARCH_BY_BOUNDS: `${API.PUBLIC}/map/moments/search_by_bounds`
  }
} as const
