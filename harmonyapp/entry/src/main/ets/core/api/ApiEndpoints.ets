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
    GUEST: `${PUBLIC}/auth/guest`,
    REFRESH_TOKEN: `${PUBLIC}/auth/refresh_token`,
    REQUEST_PASSWORD_RESET: `${PUBLIC}/auth/request_password_reset`,
    VERIFY_RESET_CODE: `${PUBLIC}/auth/verify_reset_code`,
    RESET_PASSWORD: `${PUBLIC}/auth/reset_password`,
    // 受保护认证接口
    LOGOUT: `${PROTECTED}/auth/logout`,
    CHECK_STATUS: `${PROTECTED}/auth/check_status`,
    // 当前用户认证 (/me/ 路径)
    ME: {
      CHANGE_PASSWORD: `${PROTECTED}/auth/me/change_password`,
      SESSIONS: `${PROTECTED}/auth/me/sessions`,
      REVOKE_SESSION: `${PROTECTED}/auth/me/sessions/revoke`,
    },
  },

  // ==================== 用户管理 ====================
  USER: {
    // 公开搜索接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/user/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/user/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/user/search_by_location`,
    // 公开用户信息接口（open_the_door 包含在线状态，受隐私设置控制）
    OPEN_THE_DOOR: `${PUBLIC}/user/open_the_door`,
    STATS: `${PUBLIC}/user/stats`,
    // 公开用户列表接口（受隐私设置控制）
    FRIENDS: `${PUBLIC}/user/friends`,
    FOLLOWING: `${PUBLIC}/user/following`,
    FOLLOWERS: `${PUBLIC}/user/followers`,
    // 受保护用户接口
    RELATIONSHIP: `${PROTECTED}/user/relationship`,
    // 当前用户操作 (/me/ 路径)
    ME: {
      PROFILE: `${PROTECTED}/user/me/profile`,
      PROFILE_UPDATE: `${PROTECTED}/user/me/profile/update`,
      LOCATION: `${PROTECTED}/user/me/location`,
      UPGRADE: `${PROTECTED}/user/me/upgrade`,
      STATS: `${PROTECTED}/user/me/stats`,
      SETTINGS: `${PROTECTED}/user/me/settings`,
      SETTINGS_UPDATE: `${PROTECTED}/user/me/settings/update`,
      DELETE: `${PROTECTED}/user/me/delete`,
      // 关注关系
      FOLLOW: `${PROTECTED}/user/me/follow`,
      UNFOLLOW: `${PROTECTED}/user/me/unfollow`,
      // 拉黑关系
      BLOCK: `${PROTECTED}/user/me/block`,
      UNBLOCK: `${PROTECTED}/user/me/unblock`,
      // 列表
      FRIENDS: `${PROTECTED}/user/me/friends`,
      FOLLOWING: `${PROTECTED}/user/me/following`,
      FOLLOWERS: `${PROTECTED}/user/me/followers`,
      BLOCKED: `${PROTECTED}/user/me/blocked`,
      // 好友备注
      REMARK: `${PROTECTED}/user/me/remark`,
    },
  },

  // ==================== 生活瞬间 ====================
  MOMENTS: {
    // 公开接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/moments/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/moments/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/moments/search_by_location`,
    OPEN_THE_DOOR: `${PUBLIC}/moments/open_the_door`,
    COMMENTS: `${PUBLIC}/moments/comments`,
    USER_MOMENTS: `${PUBLIC}/user/moments`,
    LIKES: `${PUBLIC}/moments/likes`,
    // 受保护接口
    CREATE: `${PROTECTED}/moments/create`,
    UPDATE: `${PROTECTED}/moments/update`,
    DELETE: `${PROTECTED}/moments/delete`,
    LIKE: `${PROTECTED}/moments/like`,
    UNLIKE: `${PROTECTED}/moments/unlike`,
    FAVORITE: `${PROTECTED}/moments/favorite`,
    UNFAVORITE: `${PROTECTED}/moments/unfavorite`,
    // 当前用户瞬间 (/me/ 路径)
    ME: {
      LIST: `${PROTECTED}/moments/me`,
      FAVORITES: `${PROTECTED}/moments/me/favorites`,
      LIKED: `${PROTECTED}/moments/me/liked`,
      FAVORITED: `${PROTECTED}/moments/me/favorited`,
    },
  },

  // ==================== 评论 ====================
  COMMENTS: {
    // 公开接口
    LIST: `${PUBLIC}/comments`,
    REPLIES: `${PUBLIC}/comments/replies`,
    LIKES: `${PUBLIC}/comments/likes`,
    // 受保护接口
    CREATE: `${PROTECTED}/comments/create`,
    DELETE: `${PROTECTED}/comments/delete`,
    LIKE: `${PROTECTED}/comments/like`,
    UNLIKE: `${PROTECTED}/comments/unlike`,
  },

  // ==================== 私聊管理 ====================
  PRIVATE_CHATS: {
    CREATE: `${PROTECTED}/private_chats/create`,
    OPEN_THE_DOOR: `${PROTECTED}/private_chats/open_the_door`,
    UPDATE_SETTINGS: `${PROTECTED}/private_chats/update_settings`,
    DELETE: `${PROTECTED}/private_chats/delete`,
    // 当前用户私聊 (/me/ 路径)
    ME: {
      LIST: `${PROTECTED}/private_chats/me`,
    },
  },

  // ==================== 私聊消息 ====================
  PRIVATE_MESSAGES: {
    SEND: `${PROTECTED}/private_messages/send`,
    HISTORY: `${PROTECTED}/private_messages/history`,
    UPDATE: `${PROTECTED}/private_messages/update`,
    RECALL: `${PROTECTED}/private_messages/recall`,
    DELETE: `${PROTECTED}/private_messages/delete`,
    MARK_AS_READ: `${PROTECTED}/private_messages/mark_as_read`,
    ADD_REACTION: `${PROTECTED}/private_messages/add_reaction`,
    REMOVE_REACTION: `${PROTECTED}/private_messages/remove_reaction`,
    REACTIONS: `${PROTECTED}/private_messages/reactions`,
  },

  // ==================== 群组管理 ====================
  GROUP_CHATS: {
    // 公开接口
    SEARCH_BY_KEYWORD: `${PUBLIC}/group_chats/search_by_keyword`,
    SEARCH_BY_TAG: `${PUBLIC}/group_chats/search_by_tag`,
    SEARCH_BY_LOCATION: `${PUBLIC}/group_chats/search_by_location`,
    OPEN_THE_DOOR: `${PUBLIC}/group_chats/open_the_door`,
    SETTINGS: `${PUBLIC}/group_chats/settings`,
    // 受保护接口
    CREATE: `${PROTECTED}/group_chats/create`,
    UPDATE: `${PROTECTED}/group_chats/update`,
    DELETE: `${PROTECTED}/group_chats/delete`,
    TRANSFER_OWNERSHIP: `${PROTECTED}/group_chats/transfer_ownership`,
    // 成员管理
    MEMBERS: {
      LIST: `${PUBLIC}/group_chats/members`,
      INVITE: `${PROTECTED}/group_chats/members/invite`,
      JOIN: `${PROTECTED}/group_chats/members/join`,
      APPLY: `${PROTECTED}/group_chats/members/apply`,
      SET_ROLE: `${PROTECTED}/group_chats/members/set_role`,
      MUTE: `${PROTECTED}/group_chats/members/mute`,
      UNMUTE: `${PROTECTED}/group_chats/members/unmute`,
      BAN: `${PROTECTED}/group_chats/members/ban`,
      UNBAN: `${PROTECTED}/group_chats/members/unban`,
      BANNED: `${PROTECTED}/group_chats/members/banned`,
      REMOVE_MEMBER: `${PROTECTED}/group_chats/members/remove_member`,
    },
    // 申请管理
    APPLICATIONS: {
      LIST: `${PROTECTED}/group_chats/applications`,
      APPROVE: `${PROTECTED}/group_chats/applications/approve`,
      REJECT: `${PROTECTED}/group_chats/applications/reject`,
    },
    // 当前用户群组操作 (/me/ 路径)
    ME: {
      LIST: `${PROTECTED}/group_chats/me`,
      CANCEL_APPLICATION: `${PROTECTED}/group_chats/me/applications/cancel`,
      INVITES: `${PROTECTED}/group_chats/me/invites`,
      ACCEPT_INVITE: `${PROTECTED}/group_chats/me/invites/accept`,
      REJECT_INVITE: `${PROTECTED}/group_chats/me/invites/reject`,
      LEAVE: `${PROTECTED}/group_chats/me/leave`,
    },
  },

  // ==================== 群聊消息 ====================
  GROUP_MESSAGES: {
    SEND: `${PROTECTED}/group_messages/send`,
    HISTORY: `${PROTECTED}/group_messages/history`,
    UPDATE: `${PROTECTED}/group_messages/update`,
    RECALL: `${PROTECTED}/group_messages/recall`,
    DELETE: `${PROTECTED}/group_messages/delete`,
    MARK_AS_READ: `${PROTECTED}/group_messages/mark_as_read`,
    READ_STATUS: `${PROTECTED}/group_messages/read_status`,
    ADD_REACTION: `${PROTECTED}/group_messages/add_reaction`,
    REMOVE_REACTION: `${PROTECTED}/group_messages/remove_reaction`,
    REACTIONS: `${PROTECTED}/group_messages/reactions`,
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
      LIST: `${PROTECTED}/reports/me`,
    },
  },

  // ==================== 管理员举报 ====================
  ADMIN_REPORTS: {
    GET_LIST: `${ADMIN}/reports`,
    HANDLE: `${ADMIN}/reports/handle`,
  },

  // ==================== 通知订阅设置 ====================
  SUBSCRIPTIONS: {
    ME: {
      ME: `${PROTECTED}/subscriptions/me`,
      UPDATE: `${PROTECTED}/subscriptions/me/update`,
    },
  },

  // ==================== 推送设备管理 ====================
  DEVICE_TOKENS: {
    ME: {
      REGISTER: `${PROTECTED}/device_tokens/me/register`,
      UNREGISTER: `${PROTECTED}/device_tokens/me/unregister`,
      ME: `${PROTECTED}/device_tokens/me`,
    },
  },

  // ==================== 系统 ====================
  SYSTEM: {
    HEALTH: `${PUBLIC}/system/health`,
  },

  // ==================== 媒体上传 ====================
  MEDIA: {
    UPLOAD: `${PROTECTED}/media/upload`,
    INITIATE_UPLOAD: `${PROTECTED}/media/initiate_upload`,
    UPLOAD_CHUNK: `${PROTECTED}/media/upload_chunk`,
    COMPLETE_UPLOAD: `${PROTECTED}/media/complete_upload`,
    UPLOAD_STATUS: `${PROTECTED}/media/upload_status`,
    ABORT_UPLOAD: `${PROTECTED}/media/abort_upload`,
  },

  // ==================== 信息流 ====================
  FEED: {
    MOMENTS: `${PROTECTED}/feed/moments`,
    GROUPS: `${PROTECTED}/feed/groups`,
    USERS: `${PROTECTED}/feed/users`,
    TRENDING: `${PROTECTED}/feed/trending`,
    FEEDBACK: `${PROTECTED}/feed/feedback`,
  },

  // ==================== 标签 ====================
  TAGS: {
    TRENDING: `${PUBLIC}/tags/trending`,
    SUGGEST: `${PUBLIC}/tags/suggest`,
  },

  // ==================== 管理员配置 ====================
  ADMIN_CONFIG: {
    CREATE: `${ADMIN}/config/create`,
    DETAIL: `${ADMIN}/config/detail`,
    LIST: `${ADMIN}/config`,
    UPDATE: `${ADMIN}/config/update`,
    DELETE: `${ADMIN}/config/delete`,
    HISTORY: `${ADMIN}/config/history`,
    CHANGES: `${ADMIN}/config/changes`,
  },
} as const;
