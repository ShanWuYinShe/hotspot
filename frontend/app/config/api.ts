/**
 * API端点配置
 * 统一管理所有API路径，避免路径混乱
 */

// API基础配置
export const API_CONFIG = {
  // API版本
  VERSION: "v1",

  // 基础路径（已在nuxt.config.ts中配置为baseURL）
  BASE_PATH: "/api/v1",

  // WebSocket路径（需与后端 routes::ws_routes 保持一致）
  WS_PATH: "/ws/",

  // 各模块基础路径（相对于baseURL，不包含/api/v1前缀）
  MODULES: {
    AUTH: "/auth",
    PUBLIC: "/public",
    PROTECTED: "/protected",
    USER: "/user",
    GROUP: "/group",
    CHAT: "/chat",
    MESSAGE: "/messages",
    MOMENT: "/moments",
    ADMIN: "/admin",
    FRIEND: "/friends",
  },
};

// 认证相关API端点
export const AUTH_ENDPOINTS = {
  // 基础路径
  BASE: API_CONFIG.MODULES.AUTH,

  // 公开端点（不需要认证）
  GET_NONCE: "/get_nonce",          // 🔥 新增：获取认证随机数
  LOGIN: "/login",
  REGISTER: "/register",
  CREATE_TEMP: "/create_temp",
  REFRESH_TOKEN: "/refresh_token",
  HEALTH: "/health",

  // 需要认证的端点
  CHECK_STATUS: "/check_status",
  SET_PASSWORD: "/set_password",
  PASSWORD_STATUS: "/password_status",
  LOGOUT: "/logout",
} as const;

// 用户相关API端点（与hotspot.md 7.2.2对齐）
export const USER_ENDPOINTS = {
  // 公开查询接口（无需认证）
  PUBLIC: {
    BASE: `${API_CONFIG.MODULES.PUBLIC}/user`,
    SEARCH_BY_NAME: "/search_by_name",
    SEARCH_BY_ID: "/search_by_id", 
    SEARCH_BY_LOCATION: "/search_by_location",
  },

  // 受保护管理接口（需要认证）
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/user`,
    UPDATE_LOCATION: "/update_location",
    UPDATE_PROFILE: "/update_profile",
    UPGRADE_ACCOUNT: "/upgrade_account",
    GET_SETTINGS: "/get_settings",
    UPDATE_SETTINGS: "/update_settings",
    DELETE_ACCOUNT: "/delete_account",
  },

  BASE: API_CONFIG.MODULES.USER,
} as const;

// 群组相关API端点（与hotspot.md 7.3.1对齐）
export const GROUP_ENDPOINTS = {
  // 公开查询接口（无需认证）
  PUBLIC: {
    BASE: `${API_CONFIG.MODULES.PUBLIC}/group`,
    SEARCH_BY_NAME: "/search_by_name",
    SEARCH_BY_LOCATION: "/search_by_location",
    SEARCH_BY_ID: "/search_by_id",
    MEMBERS: "/members",
  },

  // 受保护管理接口（需要认证）
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/group`,
    CREATE: "/create",
    JOIN: "/join",
    LEAVE: "/leave",
    TRANSFER_OWNERSHIP: "/transfer_ownership",
    KICK: "/kick",
  },

  BASE: API_CONFIG.MODULES.GROUP,
} as const;

// 消息相关API端点（已废弃，统一使用 CHAT_ENDPOINTS）
// 注意：根据 hotspot.md 7.4，所有消息API都在 /api/v1/protected/chat 下
// export const MESSAGE_ENDPOINTS = {
//   // 已迁移到 CHAT_ENDPOINTS.PROTECTED
// } as const;

// 生活瞬间API端点（与hotspot.md 7.5.1对齐）
export const MOMENT_ENDPOINTS = {
  // 公开查询接口（无需认证）
  PUBLIC: {
    BASE: `${API_CONFIG.MODULES.PUBLIC}/moments`,
    HISTORY: "/history",
    SEARCH_BY_LOCATION: "/search_by_location",
    SEARCH_BY_TAGS: "/search_by_tags",
    SEARCH_BY_ID: "/search_by_id",
  },

  // 受保护管理接口（需要认证）
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/moments`,
    CREATE: "/create",
    DELETE: "/delete",
    LIKE: "/like",
    UNLIKE: "/unlike",
  },

  BASE: API_CONFIG.MODULES.MOMENT,
} as const;

// 管理端API端点（与 hotspot.md 7.6 管理接口对齐）
export const ADMIN_ENDPOINTS = {
  // 受保护管理接口（需要管理员认证）
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/admin`,
    USER_ID_STATS: "/user-id/stats",
    USER_ID_VALIDATE: "/user-id/validate",
    USER_ID_INFO: "/user-id/info",
    USER_ID_RANGES: "/user-id/ranges",
    USER_ID_GENERATE: "/user-id/generate",
    USER_ID_FORMAT: "/user-id/format",
  },

  BASE: API_CONFIG.MODULES.ADMIN,
} as const;

// 聊天/消息端点（与 hotspot.md 7.4 对齐，模块前缀：/protected/chat）
export const CHAT_ENDPOINTS = {
  // 受保护聊天接口（需要认证）
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/chat`,
    // 群组
    GROUP_SEND: "/group/send",
    GROUP_HISTORY: "/group/history",
    GROUP_DELETE: "/group/delete",
    GROUP_MARK_READ: "/group/mark-read",
    GROUP_UNREAD_COUNT: "/group/unread_count",
    // 私聊（user/*）
    USER_SEND: "/user/send",
    USER_HISTORY: "/user/history",
    USER_DELETE: "/user/delete",
    USER_MARK_READ: "/user/mark-read",
    USER_UNREAD_COUNT: "/user/unread_count",
    CONVERSATIONS: "/user/conversations",
  },

  BASE: API_CONFIG.MODULES.CHAT,
} as const;

// 好友相关API端点（预留功能，暂未实现 - 参考 hotspot.md 3.3.1）
// 注意：好友系统在 hotspot.md 中标记为"预留"功能，后端暂未实现具体API端点
export const FRIEND_ENDPOINTS = {
  // 预留：当后端实现好友功能时启用
  // BASE: API_CONFIG.MODULES.FRIEND,
  // LIST: "/list",
  // REQUESTS: "/requests",
  // ... 其他端点待后端实现
} as const;

// 其他端点
export const OTHER_ENDPOINTS = {
  HEALTH: "/health",
  METRICS: "/metrics",
  WS: API_CONFIG.WS_PATH,
} as const;

// 构建完整URL的辅助函数
export const buildUrl = (base: string, endpoint: string): string => {
  return `${base}${endpoint}`;
};

// 快捷构建函数
export const buildAuthUrl = (endpoint: string): string => {
  return buildUrl(AUTH_ENDPOINTS.BASE, endpoint);
};

export const buildUserUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? USER_ENDPOINTS.PUBLIC.BASE : USER_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

export const buildGroupUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? GROUP_ENDPOINTS.PUBLIC.BASE : GROUP_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

// 消息URL构建函数（已废弃，使用 buildChatUrl）
// export const buildMessageUrl = (endpoint: string): string => {
//   return buildUrl(MESSAGE_ENDPOINTS.BASE, endpoint);
// };

export const buildMomentUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? MOMENT_ENDPOINTS.PUBLIC.BASE : MOMENT_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

// 好友URL构建函数（预留功能，暂未实现）
// export const buildFriendUrl = (endpoint: string): string => {
//   return buildUrl(FRIEND_ENDPOINTS.BASE, endpoint);
// };

export const buildChatUrl = (endpoint: string): string => {
  return buildUrl(CHAT_ENDPOINTS.PROTECTED.BASE, endpoint);
};

export const buildAdminUrl = (endpoint: string): string => {
  return buildUrl(ADMIN_ENDPOINTS.PROTECTED.BASE, endpoint);
};

// 导出所有端点配置
export const API_ENDPOINTS = {
  AUTH: AUTH_ENDPOINTS,
  USER: USER_ENDPOINTS,
  GROUP: GROUP_ENDPOINTS,
  // MESSAGE: MESSAGE_ENDPOINTS, // 已废弃，使用 CHAT
  MOMENT: MOMENT_ENDPOINTS,
  CHAT: CHAT_ENDPOINTS,
  // FRIEND: FRIEND_ENDPOINTS, // 预留功能，暂未实现
  ADMIN: ADMIN_ENDPOINTS,
  OTHER: OTHER_ENDPOINTS,
} as const;

// 类型定义
export type AuthEndpoint = keyof typeof AUTH_ENDPOINTS;
export type UserEndpoint = keyof typeof USER_ENDPOINTS;
export type GroupEndpoint = keyof typeof GROUP_ENDPOINTS;
// export type MessageEndpoint = keyof typeof MESSAGE_ENDPOINTS; // 已废弃，使用 ChatEndpoint
export type MomentEndpoint = keyof typeof MOMENT_ENDPOINTS;
export type ChatEndpoint = keyof typeof CHAT_ENDPOINTS;
// export type FriendEndpoint = keyof typeof FRIEND_ENDPOINTS; // 预留功能，暂未实现
export type AdminEndpoint = keyof typeof ADMIN_ENDPOINTS;
