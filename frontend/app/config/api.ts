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
  // 公开认证端点（不需要认证）- 路径：/api/v1/public/auth/*
  PUBLIC: {
    BASE: "/public/auth",
    GET_NONCE: "/get_nonce",          // 获取认证随机数
    LOGIN: "/login",
    REGISTER: "/register",
    CREATE_TEMP: "/create_temp",
    REFRESH_TOKEN: "/refresh_token",
    CHECK_STATUS: "/check_status",
  },

  // 受保护认证端点（需要认证）- 路径：/api/v1/protected/auth/*
  PROTECTED: {
    BASE: "/protected/auth",
    SET_PASSWORD: "/set_password",
  },

  // 兼容性：保留旧的端点定义（将逐步废弃）
  BASE: "/public/auth",  // 默认指向公开认证
  GET_NONCE: "/get_nonce",
  LOGIN: "/login",
  REGISTER: "/register",
  CREATE_TEMP: "/create_temp",
  REFRESH_TOKEN: "/refresh_token",
  CHECK_STATUS: "/check_status",
  SET_PASSWORD: "/set_password",  // 注意：这个实际在protected路径下
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
    // 用户关系管理API（根据backend_doc.mdc 5.2.2）
    ADD_FRIEND: "/add_friend",
    ACCEPT_FRIEND: "/accept_friend",
    REJECT_FRIEND: "/reject_friend",
    FOLLOW_USER: "/follow_user",
    UNFOLLOW_USER: "/unfollow_user",
    BLOCK_USER: "/block_user",
    FRIENDS_LIST: "/friends_list",
    BLOCKED_LIST: "/blocked_list",
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
    MY_GROUPS: "/my_groups",
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

// 统一消息端点（与 backend_doc.mdc 5.2.4 对齐，统一消息架构）
export const CHAT_ENDPOINTS = {
  // 受保护聊天接口（需要认证）- 统一架构：群聊和私聊使用相同接口
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/chat`,
    // 统一消息接口（支持群聊和私聊）
    SEND: "/send",                    // 发送消息（统一接口）
    HISTORY: "/history",              // 获取消息历史（统一接口）
    DELETE: "/delete",                // 删除消息（统一接口）
    MARK_READ: "/mark-read",          // 标记消息已读（统一接口）
    UNREAD_COUNT: "/unread_count",    // 获取未读消息数（统一接口）
    CONVERSATIONS: "/conversations",   // 获取会话列表（统一接口）
  },

  BASE: API_CONFIG.MODULES.CHAT,
} as const;

// 内容安全API端点（根据backend_doc.mdc 5.2.7）
export const REPORT_ENDPOINTS = {
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/reports`,
    CREATE: "/create",              // 提交举报
    MY_REPORTS: "/my_reports",      // 我的举报记录
    LIST: "/list",                  // 举报列表（管理员）
    HANDLE: "/handle",              // 处理举报（管理员）
  },
} as const;

// 通知推送API端点（根据backend_doc.mdc 5.2.8）
export const NOTIFICATION_ENDPOINTS = {
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/notifications`,
    SEND: "/send",                  // 发送通知
    LIST: "/list",                  // 获取通知列表
    MARK_READ: "/mark_read",        // 标记通知已读
    UNREAD_COUNT: "/unread_count",  // 获取未读通知数
    SETTINGS: "/settings",          // 获取通知设置
    UPDATE_SETTINGS: "/update_settings", // 更新通知设置
  },
} as const;

// 系统状态API端点（根据backend_doc.mdc 5.2.9）
export const STATUS_ENDPOINTS = {
  PUBLIC: {
    BASE: `${API_CONFIG.MODULES.PUBLIC}/status`,
    HEALTH: "/health",              // 系统健康检查
  },
} as const;

// 文件上传API端点
export const UPLOAD_ENDPOINTS = {
  PROTECTED: {
    BASE: `${API_CONFIG.MODULES.PROTECTED}/upload`,
    AVATAR: "/avatar",              // 上传头像
    IMAGES: "/images",              // 批量上传图片
  },
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
export const buildAuthUrl = (endpoint: string, isProtected: boolean = false): string => {
  const base = isProtected ? AUTH_ENDPOINTS.PROTECTED.BASE : AUTH_ENDPOINTS.PUBLIC.BASE;
  return buildUrl(base, endpoint);
};

export const buildUserUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? USER_ENDPOINTS.PUBLIC.BASE : USER_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

export const buildGroupUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? GROUP_ENDPOINTS.PUBLIC.BASE : GROUP_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};


export const buildMomentUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? MOMENT_ENDPOINTS.PUBLIC.BASE : MOMENT_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};


export const buildChatUrl = (endpoint: string): string => {
  return buildUrl(CHAT_ENDPOINTS.PROTECTED.BASE, endpoint);
};

export const buildAdminUrl = (endpoint: string): string => {
  return buildUrl(ADMIN_ENDPOINTS.PROTECTED.BASE, endpoint);
};

export const buildUploadUrl = (endpoint: string): string => {
  return buildUrl(UPLOAD_ENDPOINTS.PROTECTED.BASE, endpoint);
};

export const buildReportUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = REPORT_ENDPOINTS.PROTECTED.BASE; // 举报功能只有受保护的端点
  return buildUrl(base, endpoint);
};

export const buildNotificationUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = NOTIFICATION_ENDPOINTS.PROTECTED.BASE; // 通知功能只有受保护的端点
  return buildUrl(base, endpoint);
};

export const buildStatusUrl = (endpoint: string, isPublic: boolean = true): string => {
  const base = STATUS_ENDPOINTS.PUBLIC.BASE; // 状态检查通常是公开的
  return buildUrl(base, endpoint);
};

// 导出所有端点配置
export const API_ENDPOINTS = {
  AUTH: AUTH_ENDPOINTS,
  USER: USER_ENDPOINTS,
  GROUP: GROUP_ENDPOINTS,
  MOMENT: MOMENT_ENDPOINTS,
  CHAT: CHAT_ENDPOINTS,
  REPORT: REPORT_ENDPOINTS,
  NOTIFICATION: NOTIFICATION_ENDPOINTS,
  STATUS: STATUS_ENDPOINTS,
  ADMIN: ADMIN_ENDPOINTS,
  UPLOAD: UPLOAD_ENDPOINTS,
  OTHER: OTHER_ENDPOINTS,
} as const;

// 类型定义
export type AuthEndpoint = keyof typeof AUTH_ENDPOINTS;
export type UserEndpoint = keyof typeof USER_ENDPOINTS;
export type GroupEndpoint = keyof typeof GROUP_ENDPOINTS;
export type MomentEndpoint = keyof typeof MOMENT_ENDPOINTS;
export type ChatEndpoint = keyof typeof CHAT_ENDPOINTS;
export type AdminEndpoint = keyof typeof ADMIN_ENDPOINTS;
export type UploadEndpoint = keyof typeof UPLOAD_ENDPOINTS;
