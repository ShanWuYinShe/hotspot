/**
 * API 端点常量定义
 * 所有 API 路径统一在此定义，方便维护和修改
 * 严格按照 api_spec.mdc 定义
 *
 * 后端统一使用 /api/v1/ 前缀，不区分 public/protected，
 * Token 可选/必须由后端中间件层控制。
 */

/**
 * API 版本前缀
 */
const V1 = '/api/v1';

/**
 * 管理员接口路径
 */
const ADMIN = `${V1}/admin`;

/**
 * API 端点常量
 */
export const API_ENDPOINTS = {
  // ==================== 认证相关 ====================
  AUTH: {
    REGISTER: `${V1}/auth/register`,
    LOGIN: `${V1}/auth/login`,
    GUEST: `${V1}/auth/guest`,
    REFRESH_TOKEN: `${V1}/auth/refresh_token`,
    CHECK_STATUS: `${V1}/auth/check_status`,
    LOGOUT: `${V1}/auth/logout`,
    ME: {
      CHANGE_PASSWORD: `${V1}/auth/me/change_password`,
      SESSIONS: `${V1}/auth/me/sessions`,
      REVOKE_SESSION: `${V1}/auth/me/sessions/revoke`,
      REVOKE_ALL: `${V1}/auth/me/sessions/revoke_all`,
    },
  },

  // ==================== 用户管理 ====================
  USER: {
    SEARCH_BY_KEYWORD: `${V1}/users/search_by_keyword`,
    SEARCH_BY_TAG: `${V1}/users/search_by_tag`,
    SEARCH_BY_LOCATION: `${V1}/users/search_by_location`,
    OPEN_THE_DOOR: `${V1}/users/open_the_door`,
    STATS: `${V1}/users/stats`,
    FRIENDS: `${V1}/users/friends`,
    FOLLOWING: `${V1}/users/following`,
    FOLLOWERS: `${V1}/users/followers`,
    RELATIONSHIP: `${V1}/users/relationship`,
    ME: {
      PROFILE: `${V1}/users/me/profile`,
      PROFILE_UPDATE: `${V1}/users/me/profile/update`,
      LOCATION: `${V1}/users/me/location`,
      UPGRADE: `${V1}/users/me/upgrade`,
      STATS: `${V1}/users/me/stats`,
      SETTINGS: `${V1}/users/me/settings`,
      SETTINGS_UPDATE: `${V1}/users/me/settings/update`,
      DELETE: `${V1}/users/me/delete`,
      FOLLOW: `${V1}/users/me/follow`,
      UNFOLLOW: `${V1}/users/me/unfollow`,
      BLOCK: `${V1}/users/me/block`,
      UNBLOCK: `${V1}/users/me/unblock`,
      FRIENDS: `${V1}/users/me/friends`,
      FOLLOWING: `${V1}/users/me/following`,
      FOLLOWERS: `${V1}/users/me/followers`,
      BLOCKED: `${V1}/users/me/blocked`,
      REMARK: `${V1}/users/me/remark`,
    },
  },

  // ==================== 生活瞬间 ====================
  MOMENTS: {
    SEARCH_BY_KEYWORD: `${V1}/moments/search_by_keyword`,
    SEARCH_BY_TAG: `${V1}/moments/search_by_tag`,
    SEARCH_BY_LOCATION: `${V1}/moments/search_by_location`,
    OPEN_THE_DOOR: `${V1}/moments/open_the_door`,
    COMMENTS: `${V1}/moments/comments`,
    USER_MOMENTS: `${V1}/users/moments`,
    LIKES: `${V1}/moments/likes`,
    CREATE: `${V1}/moments/create`,
    UPDATE: `${V1}/moments/update`,
    DELETE: `${V1}/moments/delete`,
    LIKE: `${V1}/moments/like`,
    UNLIKE: `${V1}/moments/unlike`,
    FAVORITE: `${V1}/moments/favorite`,
    UNFAVORITE: `${V1}/moments/unfavorite`,
    ME: {
      LIST: `${V1}/moments/me`,
      FAVORITES: `${V1}/moments/me/favorites`,
      LIKED: `${V1}/moments/me/liked`,
      FAVORITED: `${V1}/moments/me/favorited`,
    },
  },

  // ==================== 评论 ====================
  COMMENTS: {
    LIST: `${V1}/comments`,
    REPLIES: `${V1}/comments/replies`,
    LIKES: `${V1}/comments/likes`,
    CREATE: `${V1}/comments/create`,
    DELETE: `${V1}/comments/delete`,
    LIKE: `${V1}/comments/like`,
    UNLIKE: `${V1}/comments/unlike`,
  },

  // ==================== 私聊管理 ====================
  PRIVATE_CHATS: {
    CREATE: `${V1}/private_chats/create`,
    OPEN_THE_DOOR: `${V1}/private_chats/open_the_door`,
    UPDATE_SETTINGS: `${V1}/private_chats/update_settings`,
    DELETE: `${V1}/private_chats/delete`,
    ME: {
      LIST: `${V1}/private_chats/me`,
    },
  },

  // ==================== 私聊消息 ====================
  PRIVATE_MESSAGES: {
    SEND: `${V1}/private_messages/send`,
    HISTORY: `${V1}/private_messages/history`,
    UPDATE: `${V1}/private_messages/update`,
    RECALL: `${V1}/private_messages/recall`,
    DELETE: `${V1}/private_messages/delete`,
    MARK_AS_READ: `${V1}/private_messages/mark_as_read`,
    ADD_REACTION: `${V1}/private_messages/add_reaction`,
    REMOVE_REACTION: `${V1}/private_messages/remove_reaction`,
    REACTIONS: `${V1}/private_messages/reactions`,
  },

  // ==================== 群组管理 ====================
  GROUP_CHATS: {
    SEARCH_BY_KEYWORD: `${V1}/groups/search_by_keyword`,
    SEARCH_BY_TAG: `${V1}/groups/search_by_tag`,
    SEARCH_BY_LOCATION: `${V1}/groups/search_by_location`,
    OPEN_THE_DOOR: `${V1}/groups/open_the_door`,
    SETTINGS: `${V1}/groups/settings`,
    CREATE: `${V1}/groups/create`,
    UPDATE: `${V1}/groups/update`,
    DELETE: `${V1}/groups/delete`,
    TRANSFER_OWNERSHIP: `${V1}/groups/transfer_ownership`,
    MEMBERS: {
      LIST: `${V1}/groups/members`,
      INVITE: `${V1}/groups/members/invite`,
      JOIN: `${V1}/groups/members/join`,
      APPLY: `${V1}/groups/members/apply`,
      SET_ROLE: `${V1}/groups/members/set_role`,
      MUTE: `${V1}/groups/members/mute`,
      UNMUTE: `${V1}/groups/members/unmute`,
      BAN: `${V1}/groups/members/ban`,
      UNBAN: `${V1}/groups/members/unban`,
      BANNED: `${V1}/groups/members/banned`,
      REMOVE_MEMBER: `${V1}/groups/members/remove_member`,
    },
    APPLICATIONS: {
      LIST: `${V1}/groups/applications`,
      APPROVE: `${V1}/groups/applications/approve`,
      REJECT: `${V1}/groups/applications/reject`,
    },
    ME: {
      LIST: `${V1}/groups/me`,
      CANCEL_APPLICATION: `${V1}/groups/me/applications/cancel`,
      INVITES: `${V1}/groups/me/invites`,
      ACCEPT_INVITE: `${V1}/groups/me/invites/accept`,
      REJECT_INVITE: `${V1}/groups/me/invites/reject`,
      LEAVE: `${V1}/groups/me/leave`,
    },
  },

  // ==================== 群聊消息 ====================
  GROUP_MESSAGES: {
    SEND: `${V1}/group_messages/send`,
    HISTORY: `${V1}/group_messages/history`,
    UPDATE: `${V1}/group_messages/update`,
    RECALL: `${V1}/group_messages/recall`,
    DELETE: `${V1}/group_messages/delete`,
    MARK_AS_READ: `${V1}/group_messages/mark_as_read`,
    READ_STATUS: `${V1}/group_messages/read_status`,
    ADD_REACTION: `${V1}/group_messages/add_reaction`,
    REMOVE_REACTION: `${V1}/group_messages/remove_reaction`,
    REACTIONS: `${V1}/group_messages/reactions`,
  },

  // ==================== 消息通用 ====================
  MESSAGES: {
    MARK_ALL_AS_READ: `${V1}/messages/mark_all_as_read`,
  },

  // ==================== 举报 ====================
  REPORTS: {
    CREATE: `${V1}/reports/create`,
    ME: {
      LIST: `${V1}/reports/me`,
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
      ME: `${V1}/subscriptions/me`,
      UPDATE: `${V1}/subscriptions/me/update`,
    },
  },

  // ==================== 推送设备管理 ====================
  DEVICE_TOKENS: {
    ME: {
      REGISTER: `${V1}/device_tokens/me/register`,
      UNREGISTER: `${V1}/device_tokens/me/unregister`,
      ME: `${V1}/device_tokens/me`,
    },
  },

  // ==================== 系统 ====================
  SYSTEM: {
    HEALTH: `${V1}/system/health`,
    CLIENT_CONFIG: `${V1}/system/client_config`,
  },

  // ==================== 媒体上传 ====================
  MEDIA: {
    UPLOAD: `${V1}/media/upload`,
    INITIATE_UPLOAD: `${V1}/media/initiate_upload`,
    UPLOAD_CHUNK: `${V1}/media/upload_chunk`,
    COMPLETE_UPLOAD: `${V1}/media/complete_upload`,
    UPLOAD_STATUS: `${V1}/media/upload_status`,
    ABORT_UPLOAD: `${V1}/media/abort_upload`,
  },

  // ==================== 信息流 ====================
  FEED: {
    MOMENTS: `${V1}/feed/moments`,
    GROUPS: `${V1}/feed/groups`,
    USERS: `${V1}/feed/users`,
    TRENDING: `${V1}/feed/trending`,
    FEEDBACK: `${V1}/feed/feedback`,
  },

  // ==================== 标签 ====================
  TAGS: {
    TRENDING: `${V1}/tags/trending`,
    SUGGEST: `${V1}/tags/suggest`,
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

/**
 * 快捷引用别名
 */
export const ApiEndpoints = API_ENDPOINTS;
