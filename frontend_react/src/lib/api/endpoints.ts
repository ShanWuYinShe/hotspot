/**
 * HotSpot API 端点常量
 * 基于 @api_spec.mdc
 * 
 * 规则：所有 API 路径统一使用 POST 方法
 */

const API_PREFIX = '/api/v1';

export const ENDPOINTS = {
  // 认证
  AUTH: {
    LOGIN: `${API_PREFIX}/auth/login`,
    REGISTER: `${API_PREFIX}/auth/register`,
    GUEST: `${API_PREFIX}/auth/guest`,
    REFRESH_TOKEN: `${API_PREFIX}/auth/refresh_token`,
    CHECK_STATUS: `${API_PREFIX}/auth/check_status`,
    LOGOUT: `${API_PREFIX}/auth/logout`,
    // 密码相关
    CHANGE_PASSWORD: `${API_PREFIX}/auth/me/change_password`,
    SEND_VERIFICATION_CODE: `${API_PREFIX}/auth/send_verification_code`,
    VERIFY_CODE: `${API_PREFIX}/auth/verify_code`,
    REQUEST_PASSWORD_RESET: `${API_PREFIX}/auth/request_password_reset`,
    VERIFY_RESET_CODE: `${API_PREFIX}/auth/verify_reset_code`,
    RESET_PASSWORD: `${API_PREFIX}/auth/reset_password`,
    // 会话管理
    SESSIONS: `${API_PREFIX}/auth/me/sessions`,
    SESSIONS_REVOKE: `${API_PREFIX}/auth/me/sessions/revoke`,
  },
  
  // 用户
  USER: {
    SEARCH_BY_KEYWORD: `${API_PREFIX}/user/search_by_keyword`,
    SEARCH_BY_TAG: `${API_PREFIX}/user/search_by_tag`,
    SEARCH_BY_LOCATION: `${API_PREFIX}/user/search_by_location`,
    OPEN_THE_DOOR: `${API_PREFIX}/user/open_the_door`,
    RELATIONSHIP: `${API_PREFIX}/user/relationship`,
    STATS: `${API_PREFIX}/user/stats`,
    FRIENDS: `${API_PREFIX}/user/friends`,
    FOLLOWING: `${API_PREFIX}/user/following`,
    FOLLOWERS: `${API_PREFIX}/user/followers`,
    GROUPS: `${API_PREFIX}/user/groups`,
    LIKED_MOMENTS: `${API_PREFIX}/user/liked_moments`,
    MOMENTS: `${API_PREFIX}/user/moments`,
    // /me/ 端点
    ME: {
      PROFILE: `${API_PREFIX}/user/me/profile`,
      PROFILE_UPDATE: `${API_PREFIX}/user/me/profile/update`,
      LOCATION: `${API_PREFIX}/user/me/location`,
      UPGRADE: `${API_PREFIX}/user/me/upgrade`,
      STATS: `${API_PREFIX}/user/me/stats`,
      SETTINGS: `${API_PREFIX}/user/me/settings`,
      SETTINGS_UPDATE: `${API_PREFIX}/user/me/settings/update`,
      DELETE: `${API_PREFIX}/user/me/delete`,
      FOLLOW: `${API_PREFIX}/user/me/follow`,
      UNFOLLOW: `${API_PREFIX}/user/me/unfollow`,
      BLOCK: `${API_PREFIX}/user/me/block`,
      UNBLOCK: `${API_PREFIX}/user/me/unblock`,
      FRIENDS: `${API_PREFIX}/user/me/friends`,
      FOLLOWING: `${API_PREFIX}/user/me/following`,
      FOLLOWERS: `${API_PREFIX}/user/me/followers`,
      FOLLOW_REQUESTS: `${API_PREFIX}/user/me/follow_requests`,
      FOLLOW_REQUESTS_HANDLE: `${API_PREFIX}/user/me/follow_requests/handle`,
      BLOCKED: `${API_PREFIX}/user/me/blocked`,
      REMARK: `${API_PREFIX}/user/me/remark`,
    },
  },
  
  // 瞬间
  MOMENTS: {
    SEARCH_BY_KEYWORD: `${API_PREFIX}/moments/search_by_keyword`,
    SEARCH_BY_TAG: `${API_PREFIX}/moments/search_by_tag`,
    SEARCH_BY_LOCATION: `${API_PREFIX}/moments/search_by_location`,
    OPEN_THE_DOOR: `${API_PREFIX}/moments/open_the_door`,
    INTERACTION_STATE: `${API_PREFIX}/moments/interaction_state`,
    CREATE: `${API_PREFIX}/moments/create`,
    UPDATE: `${API_PREFIX}/moments/update`,
    DELETE: `${API_PREFIX}/moments/delete`,
    LIKE: `${API_PREFIX}/moments/like`,
    UNLIKE: `${API_PREFIX}/moments/unlike`,
    FAVORITE: `${API_PREFIX}/moments/favorite`,
    UNFAVORITE: `${API_PREFIX}/moments/unfavorite`,
    LIKES: `${API_PREFIX}/moments/likes`,
    // /me/ 端点
    ME: `${API_PREFIX}/moments/me`,
    ME_FAVORITES: `${API_PREFIX}/moments/me/favorites`,
    ME_LIKED: `${API_PREFIX}/moments/me/liked`,
    ME_FAVORITED: `${API_PREFIX}/moments/me/favorited`,
  },
  
  // 评论
  COMMENTS: {
    CREATE: `${API_PREFIX}/comments/create`,
    DELETE: `${API_PREFIX}/comments/delete`,
    LIST: `${API_PREFIX}/comments`,
    REPLIES: `${API_PREFIX}/comments/replies`,
    LIKE: `${API_PREFIX}/comments/like`,
    UNLIKE: `${API_PREFIX}/comments/unlike`,
    LIKES: `${API_PREFIX}/comments/likes`,
  },
  
  // 私聊
  PRIVATE_CHATS: {
    ME: `${API_PREFIX}/private_chats/me`,
    OPEN_THE_DOOR: `${API_PREFIX}/private_chats/open_the_door`,
    UPDATE_SETTINGS: `${API_PREFIX}/private_chats/update_settings`,
    DELETE: `${API_PREFIX}/private_chats/delete`,
  },
  
  // 私聊消息
  PRIVATE_MESSAGES: {
    SEND: `${API_PREFIX}/private_messages/send`,
    HISTORY: `${API_PREFIX}/private_messages/history`,
    UPDATE: `${API_PREFIX}/private_messages/update`,
    RECALL: `${API_PREFIX}/private_messages/recall`,
    DELETE: `${API_PREFIX}/private_messages/delete`,
    MARK_AS_READ: `${API_PREFIX}/private_messages/mark_as_read`,
    ADD_REACTION: `${API_PREFIX}/private_messages/add_reaction`,
    REMOVE_REACTION: `${API_PREFIX}/private_messages/remove_reaction`,
    REACTIONS: `${API_PREFIX}/private_messages/reactions`,
  },
  
  // 群组
  GROUP_CHATS: {
    SEARCH_BY_KEYWORD: `${API_PREFIX}/group_chats/search_by_keyword`,
    SEARCH_BY_TAG: `${API_PREFIX}/group_chats/search_by_tag`,
    SEARCH_BY_LOCATION: `${API_PREFIX}/group_chats/search_by_location`,
    OPEN_THE_DOOR: `${API_PREFIX}/group_chats/open_the_door`,
    MY_STATE: `${API_PREFIX}/group_chats/my_state`,
    SETTINGS: `${API_PREFIX}/group_chats/settings`,
    CREATE: `${API_PREFIX}/group_chats/create`,
    UPDATE: `${API_PREFIX}/group_chats/update`,
    UPDATE_MY_SETTINGS: `${API_PREFIX}/group_chats/update_my_settings`,
    DELETE: `${API_PREFIX}/group_chats/delete`,
    MEMBERS: `${API_PREFIX}/group_chats/members`,
    MEMBERS_INVITE: `${API_PREFIX}/group_chats/members/invite`,
    MEMBERS_JOIN: `${API_PREFIX}/group_chats/members/join`,
    MEMBERS_APPLY: `${API_PREFIX}/group_chats/members/apply`,
    MEMBERS_SET_ROLE: `${API_PREFIX}/group_chats/members/set_role`,
    MEMBERS_MUTE: `${API_PREFIX}/group_chats/members/mute`,
    MEMBERS_UNMUTE: `${API_PREFIX}/group_chats/members/unmute`,
    MEMBERS_BAN: `${API_PREFIX}/group_chats/members/ban`,
    MEMBERS_UNBAN: `${API_PREFIX}/group_chats/members/unban`,
    MEMBERS_BANNED: `${API_PREFIX}/group_chats/members/banned`,
    MEMBERS_REMOVE: `${API_PREFIX}/group_chats/members/remove_member`,
    APPLICATIONS: `${API_PREFIX}/group_chats/applications`,
    APPLICATIONS_APPROVE: `${API_PREFIX}/group_chats/applications/approve`,
    APPLICATIONS_REJECT: `${API_PREFIX}/group_chats/applications/reject`,
    TRANSFER_OWNERSHIP: `${API_PREFIX}/group_chats/transfer_ownership`,
    ANNOUNCEMENTS: `${API_PREFIX}/group_chats/announcements`,
    ANNOUNCEMENTS_CREATE: `${API_PREFIX}/group_chats/announcements/create`,
    ANNOUNCEMENTS_UPDATE: `${API_PREFIX}/group_chats/announcements/update`,
    ANNOUNCEMENTS_DELETE: `${API_PREFIX}/group_chats/announcements/delete`,
    // /me/ 端点
    ME: `${API_PREFIX}/group_chats/me`,
    ME_APPLICATIONS_CANCEL: `${API_PREFIX}/group_chats/me/applications/cancel`,
    ME_INVITES: `${API_PREFIX}/group_chats/me/invites`,
    ME_INVITES_ACCEPT: `${API_PREFIX}/group_chats/me/invites/accept`,
    ME_INVITES_REJECT: `${API_PREFIX}/group_chats/me/invites/reject`,
    ME_LEAVE: `${API_PREFIX}/group_chats/me/leave`,
  },
  
  // 群消息
  GROUP_MESSAGES: {
    SEND: `${API_PREFIX}/group_messages/send`,
    HISTORY: `${API_PREFIX}/group_messages/history`,
    UPDATE: `${API_PREFIX}/group_messages/update`,
    RECALL: `${API_PREFIX}/group_messages/recall`,
    DELETE: `${API_PREFIX}/group_messages/delete`,
    MARK_AS_READ: `${API_PREFIX}/group_messages/mark_as_read`,
    READ_STATUS: `${API_PREFIX}/group_messages/read_status`,
    ADD_REACTION: `${API_PREFIX}/group_messages/add_reaction`,
    REMOVE_REACTION: `${API_PREFIX}/group_messages/remove_reaction`,
    REACTIONS: `${API_PREFIX}/group_messages/reactions`,
  },
  
  // 消息通用
  MESSAGES: {
    MARK_ALL_AS_READ: `${API_PREFIX}/messages/mark_all_as_read`,
  },
  
  // 信息流
  FEED: {
    MOMENTS: `${API_PREFIX}/feed/moments`,
    GROUPS: `${API_PREFIX}/feed/groups`,
    USERS: `${API_PREFIX}/feed/users`,
    TRENDING: `${API_PREFIX}/feed/trending`,
    FEEDBACK: `${API_PREFIX}/feed/feedback`,
  },
  
  // 标签
  TAGS: {
    TRENDING: `${API_PREFIX}/tags/trending`,
    SUGGEST: `${API_PREFIX}/tags/suggest`,
  },
  
  // 地图
  MAP: {
    RENDER: `${API_PREFIX}/map/render`,
  },
  
  // 通知
  NOTIFICATIONS: {
    LIST: `${API_PREFIX}/notifications`,
    UNREAD_COUNT: `${API_PREFIX}/notifications/unread_count`,
    MARK_READ: `${API_PREFIX}/notifications/mark_read`,
  },
  
  // 订阅设置
  SUBSCRIPTIONS: {
    ME: `${API_PREFIX}/subscriptions/me`,
    ME_UPDATE: `${API_PREFIX}/subscriptions/me/update`,
  },
  
  // 设备令牌
  DEVICE_TOKENS: {
    ME_REGISTER: `${API_PREFIX}/device_tokens/me/register`,
    ME_UNREGISTER: `${API_PREFIX}/device_tokens/me/unregister`,
    ME: `${API_PREFIX}/device_tokens/me`,
  },
  
  // 媒体
  MEDIA: {
    PRESIGN: `${API_PREFIX}/media/presign`,
    COMPLETE: `${API_PREFIX}/media/complete`,
  },
  
  // 举报
  REPORTS: {
    CREATE: `${API_PREFIX}/reports/create`,
  },
  
  // 系统
  SYSTEM: {
    HEALTH: `${API_PREFIX}/system/health`,
    CONFIG: `${API_PREFIX}/system/config`,
    ANNOUNCEMENTS: `${API_PREFIX}/system/announcements`,
  },
  
  // 管理员
  ADMIN: {
    TAGS_DELETE: `${API_PREFIX}/admin/tags/delete`,
  },
} as const;
