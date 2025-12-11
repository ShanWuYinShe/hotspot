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
  CREATE_TEMP: `${API.PUBLIC}/auth/create_temp_user`,
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
    GET_SESSIONS: `${API.PROTECTED}/auth/me/get_sessions`,
    REVOKE_SESSION: `${API.PROTECTED}/auth/me/revoke_session`
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
  GET_STATS: `${API.PUBLIC}/user/get_stats`,  // 获取指定用户统计数据
  // 获取指定用户列表（受隐私设置控制）
  GET_USER_FRIENDS_LIST: `${API.PUBLIC}/user/get_user_friends_list`,
  GET_USER_FOLLOWING_LIST: `${API.PUBLIC}/user/get_user_following_list`,
  GET_USER_FOLLOWERS_LIST: `${API.PUBLIC}/user/get_user_followers_list`,
  // 获取指定用户关系 (protected - 需要登录才能获取)
  GET_RELATIONSHIP: `${API.PROTECTED}/user/get_relationship`,
  // 当前用户操作 (protected, /me/ 路径)
  ME: `${API.PROTECTED}/user/me`,
  UPDATE_PROFILE: `${API.PROTECTED}/user/me/update_profile`,
  UPDATE_LOCATION: `${API.PROTECTED}/user/me/update_location`,
  UPGRADE_ACCOUNT: `${API.PROTECTED}/user/me/upgrade_account`,
  GET_MY_STATS: `${API.PROTECTED}/user/me/get_stats`,
  GET_SETTINGS: `${API.PROTECTED}/user/me/get_settings`,
  UPDATE_SETTINGS: `${API.PROTECTED}/user/me/update_settings`,
  DELETE_ACCOUNT: `${API.PROTECTED}/user/me/delete_account`,
  GET_FRIENDS_LIST: `${API.PROTECTED}/user/me/get_friends_list`,
  GET_FOLLOWING_LIST: `${API.PROTECTED}/user/me/get_following_list`,
  GET_FOLLOWERS_LIST: `${API.PROTECTED}/user/me/get_followers_list`,
  GET_BLOCKED_LIST: `${API.PROTECTED}/user/me/get_blocked_list`,
  // 关注关系操作 (protected, /me 子路径)
  FOLLOW_USER: `${API.PROTECTED}/user/me/follow_user`,
  UNFOLLOW_USER: `${API.PROTECTED}/user/me/unfollow_user`,
  BLOCK_USER: `${API.PROTECTED}/user/me/block_user`,
  UNBLOCK_USER: `${API.PROTECTED}/user/me/unblock_user`,
  SET_FRIEND_REMARK: `${API.PROTECTED}/user/me/set_friend_remark`
} as const

// 生活瞬间 API
export const MOMENTS = {
  // 搜索 (public)
  SEARCH_BY_KEYWORD: `${API.PUBLIC}/moments/search_by_keyword`,
  SEARCH_BY_TAG: `${API.PUBLIC}/moments/search_by_tag`,
  SEARCH_BY_LOCATION: `${API.PUBLIC}/moments/search_by_location`,
  // 获取指定瞬间信息 (public)
  OPEN_THE_DOOR: `${API.PUBLIC}/moments/open_the_door`,
  GET_COMMENTS_LIST: `${API.PUBLIC}/moments/get_comments_list`,
  GET_USER_MOMENTS_LIST: `${API.PUBLIC}/moments/get_user_moments_list`,
  GET_LIKES_LIST: `${API.PUBLIC}/moments/get_likes_list`,
  // 操作 (protected)
  CREATE: `${API.PROTECTED}/moments/create`,
  UPDATE: `${API.PROTECTED}/moments/update`,
  DELETE: `${API.PROTECTED}/moments/delete`,
  LIKE_MOMENT: `${API.PROTECTED}/moments/like_moment`,
  UNLIKE_MOMENT: `${API.PROTECTED}/moments/unlike_moment`,
  FAVORITE_MOMENT: `${API.PROTECTED}/moments/favorite_moment`,
  UNFAVORITE_MOMENT: `${API.PROTECTED}/moments/unfavorite_moment`,
  // 当前用户相关 (protected, /me/ 路径)
  GET_MY_LIST: `${API.PROTECTED}/moments/me/get_list`,
  GET_FAVORITES_LIST: `${API.PROTECTED}/moments/me/get_favorites_list`,
  CHECK_LIKED: `${API.PROTECTED}/moments/me/check_liked`,
  CHECK_FAVORITED: `${API.PROTECTED}/moments/me/check_favorited`
} as const

// 评论 API
export const COMMENTS = {
  ADD: `${API.PROTECTED}/comments/add`,
  REMOVE: `${API.PROTECTED}/comments/remove`,
  LIKE_COMMENT: `${API.PROTECTED}/comments/like_comment`,
  UNLIKE_COMMENT: `${API.PROTECTED}/comments/unlike_comment`,
  GET_LIST: `${API.PUBLIC}/comments/get_list`,
  GET_REPLIES: `${API.PUBLIC}/comments/get_replies`,
  GET_LIKES_LIST: `${API.PUBLIC}/comments/get_likes_list`
} as const

// 私聊 API
export const PRIVATE_CHATS = {
  CREATE: `${API.PROTECTED}/private_chats/create`,
  OPEN_THE_DOOR: `${API.PROTECTED}/private_chats/open_the_door`,
  UPDATE_SETTINGS: `${API.PROTECTED}/private_chats/update_settings`,
  DELETE: `${API.PROTECTED}/private_chats/delete`,
  // 当前用户私聊 (protected, /me/ 路径)
  GET_LIST: `${API.PROTECTED}/private_chats/me/get_list`
} as const

// 私聊消息 API
export const PRIVATE_MESSAGES = {
  SEND: `${API.PROTECTED}/private_messages/send`,
  GET_HISTORY: `${API.PROTECTED}/private_messages/get_history`,
  UPDATE: `${API.PROTECTED}/private_messages/update`,
  RECALL: `${API.PROTECTED}/private_messages/recall`,
  DELETE: `${API.PROTECTED}/private_messages/delete`,
  MARK_AS_READ: `${API.PROTECTED}/private_messages/mark_as_read`,
  ADD_REACTION: `${API.PROTECTED}/private_messages/add_reaction`,
  REMOVE_REACTION: `${API.PROTECTED}/private_messages/remove_reaction`,
  GET_REACTIONS_LIST: `${API.PROTECTED}/private_messages/get_reactions_list`
} as const

// 群组 API
export const GROUPS = {
  // 搜索 (public)
  SEARCH_BY_KEYWORD: `${API.PUBLIC}/group_chats/search_by_keyword`,
  SEARCH_BY_TAG: `${API.PUBLIC}/group_chats/search_by_tag`,
  SEARCH_BY_LOCATION: `${API.PUBLIC}/group_chats/search_by_location`,
  // 获取指定群组信息 (public)
  OPEN_THE_DOOR: `${API.PUBLIC}/group_chats/open_the_door`,
  GET_SETTINGS: `${API.PUBLIC}/group_chats/get_settings`,
  // 操作 (protected)
  CREATE: `${API.PROTECTED}/group_chats/create`,
  UPDATE: `${API.PROTECTED}/group_chats/update`,
  DELETE: `${API.PROTECTED}/group_chats/delete`,
  TRANSFER_OWNERSHIP: `${API.PROTECTED}/group_chats/transfer_ownership`,
  // 成员管理
  MEMBERS: {
    GET_LIST: `${API.PUBLIC}/group_chats/members/get_list`,
    INVITE: `${API.PROTECTED}/group_chats/members/invite`,
    JOIN: `${API.PROTECTED}/group_chats/members/join`,
    APPLY: `${API.PROTECTED}/group_chats/members/apply`,
    SET_ROLE: `${API.PROTECTED}/group_chats/members/set_role`,
    MUTE: `${API.PROTECTED}/group_chats/members/mute`,
    UNMUTE: `${API.PROTECTED}/group_chats/members/unmute`,
    BAN_MEMBER: `${API.PROTECTED}/group_chats/members/ban_member`,
    UNBAN_MEMBER: `${API.PROTECTED}/group_chats/members/unban_member`,
    GET_BANNED_LIST: `${API.PROTECTED}/group_chats/members/get_banned_list`,
    REMOVE_MEMBER: `${API.PROTECTED}/group_chats/members/remove_member`
  },
  // 申请管理（群管理员操作）
  APPLICATIONS: {
    GET_LIST: `${API.PROTECTED}/group_chats/applications/get_list`,
    APPROVE: `${API.PROTECTED}/group_chats/applications/approve`,
    REJECT: `${API.PROTECTED}/group_chats/applications/reject`
  },
  // 当前用户群组相关 (protected, /me/ 路径)
  ME: {
    GET_JOINED: `${API.PROTECTED}/group_chats/me/get_joined`,
    CANCEL_APPLICATION: `${API.PROTECTED}/group_chats/me/cancel_application`,
    GET_INVITES_LIST: `${API.PROTECTED}/group_chats/me/get_invites_list`,
    ACCEPT_INVITE: `${API.PROTECTED}/group_chats/me/accept_invite`,
    REJECT_INVITE: `${API.PROTECTED}/group_chats/me/reject_invite`,
    LEAVE_GROUP: `${API.PROTECTED}/group_chats/me/leave_group`
  }
} as const

// 群聊消息 API
export const GROUP_MESSAGES = {
  SEND: `${API.PROTECTED}/group_messages/send`,
  GET_HISTORY: `${API.PROTECTED}/group_messages/get_history`,
  UPDATE: `${API.PROTECTED}/group_messages/update`,
  RECALL: `${API.PROTECTED}/group_messages/recall`,
  DELETE: `${API.PROTECTED}/group_messages/delete`,
  MARK_AS_READ: `${API.PROTECTED}/group_messages/mark_as_read`,
  GET_READ_STATUS: `${API.PROTECTED}/group_messages/get_read_status`,
  ADD_REACTION: `${API.PROTECTED}/group_messages/add_reaction`,
  REMOVE_REACTION: `${API.PROTECTED}/group_messages/remove_reaction`,
  GET_REACTIONS_LIST: `${API.PROTECTED}/group_messages/get_reactions_list`
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
  GET_LIST: `${API.ADMIN}/reports/get_list`,
  HANDLE: `${API.ADMIN}/reports/handle`,
  // 当前用户举报 (protected, /me/ 路径)
  ME_GET_LIST: `${API.PROTECTED}/reports/me/get_list`
} as const

// 媒体上传 API
export const MEDIA = {
  UPLOAD_FILE: `${API.PROTECTED}/media/upload_file`,
  INITIATE_UPLOAD: `${API.PROTECTED}/media/initiate_upload`,
  UPLOAD_CHUNK: `${API.PROTECTED}/media/upload_chunk`,
  COMPLETE_UPLOAD: `${API.PROTECTED}/media/complete_upload`,
  GET_UPLOAD_STATUS: `${API.PROTECTED}/media/get_upload_status`,
  ABORT_UPLOAD: `${API.PROTECTED}/media/abort_upload`
} as const

// 标签 API
export const TAGS = {
  GET_TRENDING: `${API.PUBLIC}/tags/get_trending`,
  SUGGEST: `${API.PUBLIC}/tags/suggest`
} as const

// 通知订阅 API (当前用户, /me/ 路径)
export const SUBSCRIPTIONS = {
  GET_CONFIG: `${API.PROTECTED}/subscriptions/me/get_config`,
  UPDATE: `${API.PROTECTED}/subscriptions/me/update`
} as const

// 设备推送 API (当前用户, /me/ 路径)
export const DEVICE_TOKENS = {
  REGISTER: `${API.PROTECTED}/device_tokens/me/register`,
  UNREGISTER: `${API.PROTECTED}/device_tokens/me/unregister`,
  GET_LIST: `${API.PROTECTED}/device_tokens/me/get_list`
} as const

// 系统 API
export const SYSTEM = {
  HEALTH_CHECK: `${API.PUBLIC}/system/health_check`,
  REVERSE_GEOCODE: `${API.PUBLIC}/system/reverse_geocode`
} as const
