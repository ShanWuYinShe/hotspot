/**
 * API 端点常量定义
 * 所有 API 路径统一在此定义，方便维护和修改
 * 严格按照 api_spec.mdc 定义
 */

/**
 * API 版本前缀
 */
const API_V1 = '/api/v1';

/**
 * 公开接口路径（无需认证）
 */
const PUBLIC = `${API_V1}/public`;

/**
 * 受保护接口路径（需要认证）
 */
const PROTECTED = `${API_V1}/protected`;

/**
 * 管理员接口路径（需要管理员权限）
 */
const ADMIN = `${API_V1}/admin`;

/**
 * API 端点常量
 */
export const API_ENDPOINTS = {
  // ==================== 认证相关 ====================
  AUTH: {
    // 公开认证接口
    REGISTER: `${PUBLIC}/auth/register`,
    LOGIN: `${PUBLIC}/auth/login`,
    CREATE_TEMP: `${PUBLIC}/auth/create_temp_user`,
    REQUEST_PASSWORD_RESET: `${PUBLIC}/auth/request_password_reset`,
    VERIFY_RESET_CODE: `${PUBLIC}/auth/verify_reset_code`,
    RESET_PASSWORD: `${PUBLIC}/auth/reset_password`,
    // 受保护认证接口
    LOGOUT: `${PROTECTED}/auth/logout`,
    REFRESH_TOKEN: `${PROTECTED}/auth/refresh_token`,
    CHECK_STATUS: `${PROTECTED}/auth/check_status`,
    // 当前用户认证 (/me/ 路径)
    ME: {
      CHANGE_PASSWORD: `${PROTECTED}/auth/me/change_password`,
      GET_SESSIONS: `${PROTECTED}/auth/me/get_sessions`,
      REVOKE_SESSION: `${PROTECTED}/auth/me/revoke_session`,
    },
  },

  // ==================== 用户管理 ====================
  USER: {
    // 公开搜索接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/user/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/user/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/user/search_by_location`,
    // 公开用户信息接口
    OPEN_THE_DOOR: `${PUBLIC}/user/open_the_door`,
    GET_STATUS: `${PUBLIC}/user/get_status`,
    GET_STATS: `${PUBLIC}/user/get_stats`,
    // 公开用户列表接口（受隐私设置控制）
    GET_USER_FRIENDS_LIST: `${PUBLIC}/user/get_user_friends_list`,
    GET_USER_FOLLOWING_LIST: `${PUBLIC}/user/get_user_following_list`,
    GET_USER_FOLLOWERS_LIST: `${PUBLIC}/user/get_user_followers_list`,
    // 受保护用户接口
    GET_RELATIONSHIP: `${PROTECTED}/user/get_relationship`,
    // 当前用户操作 (/me/ 路径)
    ME: {
      PROFILE: `${PROTECTED}/user/me`,
      UPDATE_PROFILE: `${PROTECTED}/user/me/update_profile`,
      UPDATE_LOCATION: `${PROTECTED}/user/me/update_location`,
      UPGRADE_ACCOUNT: `${PROTECTED}/user/me/upgrade_account`,
      GET_STATS: `${PROTECTED}/user/me/get_stats`,
      GET_SETTINGS: `${PROTECTED}/user/me/get_settings`,
      UPDATE_SETTINGS: `${PROTECTED}/user/me/update_settings`,
      DELETE_ACCOUNT: `${PROTECTED}/user/me/delete_account`,
      // 关注关系
      FOLLOW_USER: `${PROTECTED}/user/me/follow_user`,
      UNFOLLOW_USER: `${PROTECTED}/user/me/unfollow_user`,
      // 拉黑关系
      BLOCK_USER: `${PROTECTED}/user/me/block_user`,
      UNBLOCK_USER: `${PROTECTED}/user/me/unblock_user`,
      // 列表
      GET_FRIENDS_LIST: `${PROTECTED}/user/me/get_friends_list`,
      GET_FOLLOWING_LIST: `${PROTECTED}/user/me/get_following_list`,
      GET_FOLLOWERS_LIST: `${PROTECTED}/user/me/get_followers_list`,
      GET_BLOCKED_LIST: `${PROTECTED}/user/me/get_blocked_list`,
      // 好友备注
      SET_FRIEND_REMARK: `${PROTECTED}/user/me/set_friend_remark`,
    },
  },

  // ==================== 生活瞬间 ====================
  MOMENTS: {
    // 公开接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/moments/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/moments/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/moments/search_by_location`,
    OPEN_THE_DOOR: `${PUBLIC}/moments/open_the_door`,
    GET_COMMENTS_LIST: `${PUBLIC}/moments/get_comments_list`,
    GET_USER_MOMENTS_LIST: `${PUBLIC}/moments/get_user_moments_list`,
    GET_LIKES_LIST: `${PUBLIC}/moments/get_likes_list`,
    // 受保护接口
    CREATE: `${PROTECTED}/moments/create`,
    UPDATE: `${PROTECTED}/moments/update`,
    DELETE: `${PROTECTED}/moments/delete`,
    LIKE_MOMENT: `${PROTECTED}/moments/like_moment`,
    UNLIKE_MOMENT: `${PROTECTED}/moments/unlike_moment`,
    FAVORITE_MOMENT: `${PROTECTED}/moments/favorite_moment`,
    UNFAVORITE_MOMENT: `${PROTECTED}/moments/unfavorite_moment`,
    // 当前用户瞬间 (/me/ 路径)
    ME: {
      GET_LIST: `${PROTECTED}/moments/me/get_list`,
      GET_FAVORITES_LIST: `${PROTECTED}/moments/me/get_favorites_list`,
      CHECK_LIKED: `${PROTECTED}/moments/me/check_liked`,
      CHECK_FAVORITED: `${PROTECTED}/moments/me/check_favorited`,
    },
  },

  // ==================== 评论 ====================
  COMMENTS: {
    // 公开接口
    GET_LIST: `${PUBLIC}/comments/get_list`,
    GET_REPLIES: `${PUBLIC}/comments/get_replies`,
    GET_LIKES_LIST: `${PUBLIC}/comments/get_likes_list`,
    // 受保护接口
    ADD: `${PROTECTED}/comments/add`,
    REMOVE: `${PROTECTED}/comments/remove`,
    LIKE_COMMENT: `${PROTECTED}/comments/like_comment`,
    UNLIKE_COMMENT: `${PROTECTED}/comments/unlike_comment`,
  },

  // ==================== 私聊管理 ====================
  PRIVATE_CHATS: {
    CREATE: `${PROTECTED}/private_chats/create`,
    OPEN_THE_DOOR: `${PROTECTED}/private_chats/open_the_door`,
    UPDATE_SETTINGS: `${PROTECTED}/private_chats/update_settings`,
    DELETE: `${PROTECTED}/private_chats/delete`,
    // 当前用户私聊 (/me/ 路径)
    ME: {
      GET_LIST: `${PROTECTED}/private_chats/me/get_list`,
    },
  },

  // ==================== 私聊消息 ====================
  PRIVATE_MESSAGES: {
    SEND: `${PROTECTED}/private_messages/send`,
    GET_HISTORY: `${PROTECTED}/private_messages/get_history`,
    UPDATE: `${PROTECTED}/private_messages/update`,
    RECALL: `${PROTECTED}/private_messages/recall`,
    DELETE: `${PROTECTED}/private_messages/delete`,
    MARK_AS_READ: `${PROTECTED}/private_messages/mark_as_read`,
    ADD_REACTION: `${PROTECTED}/private_messages/add_reaction`,
    REMOVE_REACTION: `${PROTECTED}/private_messages/remove_reaction`,
    GET_REACTIONS_LIST: `${PROTECTED}/private_messages/get_reactions_list`,
  },

  // ==================== 群组管理 ====================
  GROUP_CHATS: {
    // 公开接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/group_chats/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/group_chats/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/group_chats/search_by_location`,
    OPEN_THE_DOOR: `${PUBLIC}/group_chats/open_the_door`,
    GET_SETTINGS: `${PUBLIC}/group_chats/get_settings`,
    // 受保护接口
    CREATE: `${PROTECTED}/group_chats/create`,
    UPDATE: `${PROTECTED}/group_chats/update`,
    DELETE: `${PROTECTED}/group_chats/delete`,
    TRANSFER_OWNERSHIP: `${PROTECTED}/group_chats/transfer_ownership`,
    // 成员管理
    MEMBERS: {
      GET_LIST: `${PUBLIC}/group_chats/members/get_list`,
      INVITE: `${PROTECTED}/group_chats/members/invite`,
      JOIN: `${PROTECTED}/group_chats/members/join`,
      APPLY: `${PROTECTED}/group_chats/members/apply`,
      SET_ROLE: `${PROTECTED}/group_chats/members/set_role`,
      MUTE: `${PROTECTED}/group_chats/members/mute`,
      UNMUTE: `${PROTECTED}/group_chats/members/unmute`,
      BAN_MEMBER: `${PROTECTED}/group_chats/members/ban_member`,
      UNBAN_MEMBER: `${PROTECTED}/group_chats/members/unban_member`,
      GET_BANNED_LIST: `${PROTECTED}/group_chats/members/get_banned_list`,
      REMOVE_MEMBER: `${PROTECTED}/group_chats/members/remove_member`,
    },
    // 申请管理
    APPLICATIONS: {
      GET_LIST: `${PROTECTED}/group_chats/applications/get_list`,
      APPROVE_APPLICATION: `${PROTECTED}/group_chats/applications/approve_application`,
      REJECT_APPLICATION: `${PROTECTED}/group_chats/applications/reject_application`,
    },
    // 当前用户群组操作 (/me/ 路径)
    ME: {
      GET_JOINED: `${PROTECTED}/group_chats/me/get_joined`,
      CANCEL_APPLICATION: `${PROTECTED}/group_chats/me/cancel_application`,
      GET_INVITES_LIST: `${PROTECTED}/group_chats/me/get_invites_list`,
      ACCEPT_INVITE: `${PROTECTED}/group_chats/me/accept_invite`,
      REJECT_INVITE: `${PROTECTED}/group_chats/me/reject_invite`,
      LEAVE_GROUP: `${PROTECTED}/group_chats/me/leave_group`,
    },
  },

  // ==================== 群聊消息 ====================
  GROUP_MESSAGES: {
    SEND: `${PROTECTED}/group_messages/send`,
    GET_HISTORY: `${PROTECTED}/group_messages/get_history`,
    UPDATE: `${PROTECTED}/group_messages/update`,
    RECALL: `${PROTECTED}/group_messages/recall`,
    DELETE: `${PROTECTED}/group_messages/delete`,
    MARK_AS_READ: `${PROTECTED}/group_messages/mark_as_read`,
    GET_READ_STATUS: `${PROTECTED}/group_messages/get_read_status`,
    ADD_REACTION: `${PROTECTED}/group_messages/add_reaction`,
    REMOVE_REACTION: `${PROTECTED}/group_messages/remove_reaction`,
    GET_REACTIONS_LIST: `${PROTECTED}/group_messages/get_reactions_list`,
  },

  // ==================== 消息通用 ====================
  MESSAGES: {
    MARK_ALL_AS_READ: `${PROTECTED}/messages/mark_all_as_read`,
  },

  // ==================== 举报 ====================
  REPORTS: {
    CREATE: `${PROTECTED}/reports/create`,
    // 当前用户举报 (/me/ 路径)
    ME: {
      GET_LIST: `${PROTECTED}/reports/me/get_list`,
    },
  },

  // ==================== 管理员举报 ====================
  ADMIN_REPORTS: {
    GET_LIST: `${ADMIN}/reports/get_list`,
    HANDLE: `${ADMIN}/reports/handle`,
  },

  // ==================== 通知订阅设置 ====================
  SUBSCRIPTIONS: {
    ME: {
      GET_CONFIG: `${PROTECTED}/subscriptions/me/get_config`,
      UPDATE: `${PROTECTED}/subscriptions/me/update`,
    },
  },

  // ==================== 推送设备管理 ====================
  DEVICE_TOKENS: {
    ME: {
      REGISTER: `${PROTECTED}/device_tokens/me/register`,
      UNREGISTER: `${PROTECTED}/device_tokens/me/unregister`,
      GET_LIST: `${PROTECTED}/device_tokens/me/get_list`,
    },
  },

  // ==================== 系统 ====================
  SYSTEM: {
    HEALTH_CHECK: `${PUBLIC}/system/health_check`,
  },

  // ==================== 媒体上传 ====================
  MEDIA: {
    UPLOAD_FILE: `${PROTECTED}/media/upload_file`,
    INITIATE_UPLOAD: `${PROTECTED}/media/initiate_upload`,
    UPLOAD_CHUNK: `${PROTECTED}/media/upload_chunk`,
    COMPLETE_UPLOAD: `${PROTECTED}/media/complete_upload`,
    GET_UPLOAD_STATUS: `${PROTECTED}/media/get_upload_status`,
    ABORT_UPLOAD: `${PROTECTED}/media/abort_upload`,
  },

  // ==================== 信息流 ====================
  FEED: {
    ME: {
      GET_LIST: `${PROTECTED}/feed/me/get_list`,
      FEEDBACK: `${PROTECTED}/feed/me/feedback`,
    },
  },

  // ==================== 标签 ====================
  TAGS: {
    GET_TRENDING: `${PUBLIC}/tags/get_trending`,
    SUGGEST: `${PUBLIC}/tags/suggest`,
  },

  // ==================== 管理员配置 ====================
  ADMIN_CONFIG: {
    CREATE: `${ADMIN}/config/create`,
    GET_CONFIG: `${ADMIN}/config/get_config`,
    GET_LIST: `${ADMIN}/config/get_list`,
    UPDATE: `${ADMIN}/config/update`,
    DELETE: `${ADMIN}/config/delete`,
    GET_CONFIG_HISTORY: `${ADMIN}/config/get_config_history`,
    GET_CHANGES: `${ADMIN}/config/get_changes`,
  },
} as const;
