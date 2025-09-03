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

// 用户相关API端点
export const USER_ENDPOINTS = {
  // 基础路径
  BASE: API_CONFIG.MODULES.USER,

  // 端点
  REGISTER: "/register",
  TEMP: "/temp",
  UPGRADE: "/upgrade",
  // 文档对齐：/update_location/:user_id 与 /update_profile/:user_id
  LOCATION: "/location",                   // 需拼接 /:user_id
  PROFILE: "/profile",                     // 需拼接 /:user_id
  PROFILE_GET: "/profile/get",             // 需拼接 /:user_id
  DETAILS: "/details",                     // 用户详情端点
  PASSWORD_STATUS: "/password/status",     // 需拼接 /:user_id
  NEARBY: "/nearby",
  SEARCH: "/search",
  // 兼容旧键名（指向同一路径，便于现有服务无改动可用）
  UPDATE_LOCATION: "/update_location",
  UPDATE_PROFILE: "/update_profile",
  SEARCH_BY_NAME: "/search_by_name",
  SEARCH_BY_LOCATION: "/search_by_location",
  GET_SETTINGS: "/get_settings",
  UPDATE_SETTINGS: "/update_settings",
  DELETE_ACCOUNT: "/delete_account",
  UPGRADE_ACCOUNT: "/upgrade_account",
} as const;

// 群组相关API端点
export const GROUP_ENDPOINTS = {
  // 基础路径
  BASE: API_CONFIG.MODULES.GROUP,

  // 端点
  CREATE: "/create",
  GET: "/get",
  UPDATE: "/update",
  SEARCH_BY_LOCATION: "/search_by_location",
  SEARCH_BY_NAME: "/search_by_name",
  SEARCH_BY_ID: "/search_by_id",
  JOIN: "/join",
  LEAVE: "/leave",
  MEMBERS: "/members",
  TRANSFER_OWNERSHIP: "/transfer_ownership",
  KICK: "/kick",
  USER_GROUPS: "/user_groups",
  OWNED_GROUPS: "/owned_groups",
  INVITE: "/invite",
} as const;

// 消息相关API端点
export const MESSAGE_ENDPOINTS = {
  // 基础路径
  BASE: API_CONFIG.MODULES.MESSAGE,

  // 群组消息
  SEND_GROUP: "/send_group_message",
  GROUP_HISTORY: "/group_message_history",
  DELETE_GROUP: "/delete_group_message",
  GROUP_UNREAD_COUNT: "/group_unread_count",

  // 私聊消息
  SEND_DIRECT: "/send_direct_message",
  DIRECT_HISTORY: "/direct_message_history",
  DELETE_DIRECT: "/delete_direct_message",
  DIRECT_UNREAD_COUNT: "/direct_unread_count",

  // 通用
  MARK_READ: "/mark_read",
  CONVERSATIONS: "/conversations",
} as const;

// 动态相关API端点
export const MOMENT_ENDPOINTS = {
  // 基础路径
  BASE: API_CONFIG.MODULES.MOMENT,

  // 端点
  CREATE: "/create",
  DELETE: "/delete",
  SEARCH_BY_ID: "/search_by_id",
  SEARCH_BY_LOCATION: "/search_by_location",
  // 与 hotspot.md 对齐：/history
  HISTORY: "/history",
  // 兼容旧键名（指向相同路径）
  USER_HISTORY: "/history",
  SEARCH_BY_TAGS: "/search_by_tags",
  LIKE: "/like",
  UNLIKE: "/unlike",
} as const;

// 管理端API端点（与 hotspot.md 7.6 管理接口对齐）
export const ADMIN_ENDPOINTS = {
  BASE: API_CONFIG.MODULES.ADMIN,
  USER_ID_STATS: "/user-id/stats",
  USER_ID_VALIDATE: "/user-id/validate",
  USER_ID_INFO: "/user-id/info",
  USER_ID_RANGES: "/user-id/ranges",
  USER_ID_GENERATE: "/user-id/generate",
  USER_ID_FORMAT: "/user-id/format",
} as const;

// 聊天/消息端点（与 hotspot.md 7.4 对齐，模块前缀：/chat）
export const CHAT_ENDPOINTS = {
  BASE: API_CONFIG.MODULES.CHAT,
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
} as const;

// 好友相关API端点（与现有前端路径对齐，后端上线后可调整）
export const FRIEND_ENDPOINTS = {
  BASE: API_CONFIG.MODULES.FRIEND,
  LIST: "/list",
  REQUESTS: "/requests",
  REQUESTS_RECEIVED: "/requests/received",
  REQUESTS_SENT: "/requests/sent",
  REQUESTS_PENDING_COUNT: "/requests/pending-count",
  ACCEPT: "/requests/accept", // 实际使用时拼接 /:user_id 或 body 传参
  CANCEL: "/requests/cancel",
  SEND_REQUEST: "/send_request",
  ACCEPT_REQUEST: "/accept_request",
  REJECT_REQUEST: "/reject_request",
  REMOVE_FRIEND: "/remove_friend",
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

export const buildUserUrl = (endpoint: string): string => {
  return buildUrl(USER_ENDPOINTS.BASE, endpoint);
};

export const buildGroupUrl = (endpoint: string): string => {
  return buildUrl(GROUP_ENDPOINTS.BASE, endpoint);
};

export const buildMessageUrl = (endpoint: string): string => {
  return buildUrl(MESSAGE_ENDPOINTS.BASE, endpoint);
};

export const buildMomentUrl = (endpoint: string): string => {
  return buildUrl(MOMENT_ENDPOINTS.BASE, endpoint);
};

export const buildFriendUrl = (endpoint: string): string => {
  return buildUrl(FRIEND_ENDPOINTS.BASE, endpoint);
};

export const buildChatUrl = (endpoint: string): string => {
  return buildUrl(CHAT_ENDPOINTS.BASE, endpoint);
};

// 导出所有端点配置
export const API_ENDPOINTS = {
  AUTH: AUTH_ENDPOINTS,
  USER: USER_ENDPOINTS,
  GROUP: GROUP_ENDPOINTS,
  MESSAGE: MESSAGE_ENDPOINTS,
  MOMENT: MOMENT_ENDPOINTS,
  CHAT: CHAT_ENDPOINTS,
  FRIEND: FRIEND_ENDPOINTS,
  ADMIN: ADMIN_ENDPOINTS,
  OTHER: OTHER_ENDPOINTS,
} as const;

// 类型定义
export type AuthEndpoint = keyof typeof AUTH_ENDPOINTS;
export type UserEndpoint = keyof typeof USER_ENDPOINTS;
export type GroupEndpoint = keyof typeof GROUP_ENDPOINTS;
export type MessageEndpoint = keyof typeof MESSAGE_ENDPOINTS;
export type MomentEndpoint = keyof typeof MOMENT_ENDPOINTS;
export type ChatEndpoint = keyof typeof CHAT_ENDPOINTS;
export type FriendEndpoint = keyof typeof FRIEND_ENDPOINTS;
export type AdminEndpoint = keyof typeof ADMIN_ENDPOINTS;
