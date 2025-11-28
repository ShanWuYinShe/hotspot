/**
 * API端点配置
 * 
 * ⚠️ 重要：此文件仅供参考，实际 API 调用应使用 services/*.service.ts 中的方法
 * 所有端点定义必须与 api_spec.mdc 保持一致
 * 
 * @see api_spec.mdc - API 规范文档
 */

// API基础配置
export const API_CONFIG = {
  VERSION: "v1",
  BASE_PATH: "/api/v1",
  WS_PATH: "/ws/",
} as const;

// ==================== 认证 API (3.1) ====================
export const AUTH_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/auth",
    REGISTER: "/register",                      // 3.1.1 用户注册
    LOGIN: "/login",                            // 3.1.2 用户登录
    CREATE_TEMP_USER: "/create_temp_user",      // 3.1.4 创建临时用户
    REQUEST_PASSWORD_RESET: "/request_password_reset", // 3.1.8 请求密码重置
    VERIFY_RESET_CODE: "/verify_reset_code",    // 3.1.9 验证重置码
    RESET_PASSWORD: "/reset_password",          // 3.1.10 重置密码
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/auth",
    LOGOUT: "/logout",                          // 3.1.3 用户登出
    REFRESH_TOKEN: "/refresh_token",            // 3.1.5 刷新Token
    CHECK_STATUS: "/check_status",              // 3.1.6 检查认证状态
    SET_PASSWORD: "/set_password",              // 3.1.7 设置密码
    GET_SESSIONS: "/get_sessions",              // 3.1.11 获取活跃会话列表
    REVOKE_SESSION: "/revoke_session",          // 3.1.12 撤销会话
  },
} as const;

// ==================== 用户 API (3.2) ====================
export const USER_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/user",
    DISCOVER: "/discover",                      // 3.2.1 发现用户
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/user",
    ME: "/me",                                  // 3.2.2 获取当前用户资料
    OPEN_THE_DOOR: "/open_the_door",            // 3.2.3 获取其他用户资料
    GET_STATS: "/get_stats",                    // 3.2.4 获取统计数据
    UPDATE_LOCATION: "/update_location",        // 3.2.6 更新用户位置
    UPDATE_PROFILE: "/update_profile",          // 3.2.7 更新用户资料
    UPGRADE_ACCOUNT: "/upgrade_account",        // 3.2.8 账户升级
    GET_SETTINGS: "/get_settings",              // 3.2.9 获取用户设置
    UPDATE_SETTINGS: "/update_settings",        // 3.2.10 更新用户设置
    DELETE_ACCOUNT: "/delete_account",          // 3.2.11 删除账户
    SEND_FRIEND_REQUEST: "/send_friend_request", // 3.2.12 发送好友申请
    ACCEPT_FRIEND_REQUEST: "/accept_friend_request", // 3.2.13 接受好友申请
    REJECT_FRIEND_REQUEST: "/reject_friend_request", // 3.2.14 拒绝好友申请
    REMOVE_FRIEND: "/remove_friend",            // 3.2.15 删除好友
    FOLLOW: "/follow",                          // 3.2.16 关注用户
    UNFOLLOW: "/unfollow",                      // 3.2.17 取消关注
    BLOCK_USER: "/block_user",                  // 3.2.18 拉黑用户
    GET_FRIENDS: "/get_friends",                // 3.2.19 获取好友列表
    GET_FOLLOWING: "/get_following",            // 3.2.20 获取关注列表
    GET_FOLLOWERS: "/get_followers",            // 3.2.21 获取粉丝列表
    GET_BLOCKED_USERS: "/get_blocked_users",    // 3.2.22 获取拉黑列表
    GET_FRIEND_REQUESTS: "/get_friend_requests", // 3.2.23 获取好友申请列表
    CANCEL_FRIEND_REQUEST: "/cancel_friend_request", // 3.2.24 取消好友申请
    SET_FRIEND_REMARK: "/set_friend_remark",    // 3.2.25 设置好友备注
    UNBLOCK_USER: "/unblock_user",              // 3.2.26 解除拉黑
  },
} as const;

// ==================== 生活瞬间 API (3.3) ====================
export const MOMENT_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/moments",
    SEARCH_BY_LOCATION: "/search_by_location",  // 3.3.5 按位置搜索
    SEARCH_BY_TAGS: "/search_by_tags",          // 3.3.6 按标签搜索
    OPEN_THE_DOOR: "/open_the_door",            // 3.3.7 获取详情
    GET_LIST: "/get_list",                      // 3.3.8 获取历史
    GET_LIKES: "/get_likes",                    // 3.3.11 获取点赞列表
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/moments",
    CREATE: "/create",                          // 3.3.1 创建瞬间
    UPDATE: "/update",                          // 3.3.2 编辑瞬间
    TOGGLE_FAVORITE: "/toggle_favorite",        // 3.3.3 收藏/取消收藏
    GET_FAVORITES: "/get_favorites",            // 3.3.4 我的收藏列表
    DELETE: "/delete",                          // 3.3.9 删除瞬间
    TOGGLE_LIKE: "/toggle_like",                // 3.3.10 点赞/取消点赞
  },
} as const;

// ==================== 评论 API (3.4) ====================
export const COMMENT_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/comments",
    GET_LIST: "/get_list",                      // 3.4.3 获取评论列表
    GET_REPLIES: "/get_replies",                // 3.4.4 获取子评论列表
    GET_LIKES: "/get_likes",                    // 3.4.6 获取评论点赞列表
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/comments",
    ADD: "/add",                                // 3.4.1 添加评论/回复
    REMOVE: "/remove",                          // 3.4.2 删除评论
    TOGGLE_LIKE: "/toggle_like",                // 3.4.5 点赞/取消点赞评论
  },
} as const;

// ==================== 私聊管理 API (3.5) ====================
export const PRIVATE_CHAT_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/private_chats",
    CREATE: "/create",                          // 3.5.1 发起私聊
    GET_LIST: "/get_list",                      // 3.5.2 获取私聊列表
    OPEN_THE_DOOR: "/open_the_door",            // 3.5.3 获取私聊详情
    UPDATE_SETTINGS: "/update_settings",        // 3.5.4 更新私聊设置
    DELETE: "/delete",                          // 3.5.5 删除私聊
  },
} as const;

// ==================== 群组管理 API (3.6) ====================
export const GROUP_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/group_chats",
    DISCOVER: "/discover",                      // 3.6.2 发现群组
    OPEN_THE_DOOR: "/open_the_door",            // 3.6.4 获取群组详情
    MEMBERS_GET_LIST: "/members/get_list",      // 3.7.1 获取成员列表
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/group_chats",
    CREATE: "/create",                          // 3.6.1 创建群组
    GET_JOINED: "/get_joined",                  // 3.6.3 获取我的群组
    UPDATE: "/update",                          // 3.6.5 更新群组资料
    DELETE: "/delete",                          // 3.6.6 解散群组
    // 成员管理
    MEMBERS_INVITE: "/members/invite",          // 3.7.2 邀请成员
    MEMBERS_JOIN: "/members/join",              // 3.7.3 加入群组
    MEMBERS_APPLY: "/members/apply",            // 3.7.4 申请加入
    MEMBERS_SET_ROLE: "/members/set_role",      // 3.7.5 设置成员角色
    MEMBERS_MUTE: "/members/mute",              // 3.7.6 禁言成员
    MEMBERS_UNMUTE: "/members/unmute",          // 3.7.7 取消禁言
    MEMBERS_BAN: "/members/ban",                // 3.7.8 封禁成员
    MEMBERS_REMOVE: "/members/remove",          // 3.7.9 移除成员
    MEMBERS_LEAVE: "/members/leave",            // 3.7.10 退出群组
    // 群组申请
    APPLICATIONS_GET_LIST: "/applications/get_list", // 3.7.11 获取申请列表
    APPLICATIONS_APPROVE: "/applications/approve",   // 3.7.12 审批申请
    APPLICATIONS_REJECT: "/applications/reject",     // 3.7.13 拒绝申请
    TRANSFER_OWNERSHIP: "/transfer_ownership",  // 3.7.14 转让群主
    // 群组邀请
    INVITES_GET_LIST: "/invites/get_list",      // 3.7.15 获取邀请列表
    INVITES_ACCEPT: "/invites/accept",          // 3.7.16 接受邀请
    INVITES_REJECT: "/invites/reject",          // 3.7.17 拒绝邀请
  },
} as const;

// ==================== 私聊消息 API (3.8) ====================
export const PRIVATE_MESSAGE_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/private_messages",
    SEND: "/send",                              // 3.8.1 发送私聊消息
    GET_HISTORY: "/get_history",                // 3.8.2 获取消息历史
    UPDATE: "/update",                          // 3.8.3 编辑消息
    RECALL: "/recall",                          // 3.8.4 撤回消息
    DELETE: "/delete",                          // 3.8.5 删除消息
    MARK_AS_READ: "/mark_as_read",              // 3.8.6 标记已读
  },
} as const;

// ==================== 群聊消息 API (3.9) ====================
export const GROUP_MESSAGE_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/group_messages",
    SEND: "/send",                              // 3.9.1 发送群聊消息
    GET_HISTORY: "/get_history",                // 3.9.2 获取消息历史
    UPDATE: "/update",                          // 3.9.3 编辑消息
    RECALL: "/recall",                          // 3.9.4 撤回消息
    DELETE: "/delete",                          // 3.9.5 删除消息
    MARK_AS_READ: "/mark_as_read",              // 3.9.6 标记已读
    GET_READ_STATUS: "/get_read_status",        // 3.9.7 获取已读状态
    ADD_REACTION: "/add_reaction",              // 3.9.8 添加反应
    GET_REACTIONS: "/get_reactions",            // 3.9.9 获取反应列表
  },
} as const;

// ==================== 消息通用 API (3.10) ====================
export const MESSAGE_COMMON_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/messages",
    MARK_ALL_AS_READ: "/mark_all_as_read",      // 3.10.1 标记所有消息已读
    MARK_AS_READ_BATCH: "/mark_as_read_batch",  // 3.18.1 批量标记消息已读
  },
} as const;

// ==================== 举报审核 API (3.11) ====================
export const REPORT_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/reports",
    CREATE: "/create",                          // 3.11.1 创建举报
    GET_MY_REPORTS: "/get_my_reports",          // 3.11.2 我的举报记录
    GET_ADMIN_LIST: "/get_admin_list",          // 3.11.3 举报列表（管理员）
    HANDLE: "/handle",                          // 3.11.4 处理举报（管理员）
  },
} as const;

// ==================== 通知订阅设置 API (3.12) ====================
export const SUBSCRIPTION_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/subscriptions",
    GET_CONFIG: "/get_config",                  // 3.12.1 获取订阅配置
    UPDATE: "/",                                // 3.12.2 更新订阅配置（根路径）
  },
} as const;

// ==================== 通知中心 API (3.13) ====================
export const INBOX_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/inbox",
    GET_LIST: "/get_list",                      // 3.13.1 获取通知列表
    GET_UNREAD_COUNT: "/get_unread_count",      // 3.13.2 获取未读通知数
    MARK_ALL_AS_READ: "/mark_all_as_read",      // 3.13.3 标记所有通知已读
    REMOVE: "/remove",                          // 3.13.4 删除通知
    REGISTER_DEVICE_TOKEN: "/register_device_token",     // 3.13.5 注册设备推送Token
    UNREGISTER_DEVICE_TOKEN: "/unregister_device_token", // 3.13.6 注销设备推送Token
    REMOVE_BATCH: "/remove_batch",              // 3.18.2 批量删除通知
  },
} as const;

// ==================== 系统健康检查 API (3.14) ====================
export const SYSTEM_ENDPOINTS = {
  PUBLIC: {
    BASE: "/public/system",
    HEALTH_CHECK: "/health_check",              // 3.14.1 系统健康检查
  },
} as const;

// ==================== 排行榜 API (3.15) ====================
export const LEADERBOARD_ENDPOINTS = {
  // 公开端点 - 无需认证
  PUBLIC: {
    BASE: "/public/leaderboard",
    GET_CREATORS: "/get_creators",              // 3.15.1 获取创作者排行榜
    GET_MOMENTS: "/get_moments",                // 3.15.2 获取瞬间排行榜
  },
  // 受保护端点 - 需要认证
  PROTECTED: {
    BASE: "/protected/leaderboard",
    MY_CREATOR_RANK: "/my_creator_rank",        // 3.15.3 获取我的创作者排名
    MY_MOMENT_RANK: "/my_moment_rank",          // 3.15.4 获取我的瞬间排名
  },
} as const;

// ==================== 媒体上传 API (3.16) ====================
export const MEDIA_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/media",
    UPLOAD_FILE: "/upload_file",                // 3.16.1 上传文件
    INITIATE_UPLOAD: "/initiate_upload",        // 3.16.2 初始化大文件上传
    COMPLETE_UPLOAD: "/complete_upload",        // 3.16.3 完成大文件上传
    GET_UPLOAD_STATUS: "/get_upload_status",    // 3.16.4 查询上传状态
    ABORT_UPLOAD: "/abort_upload",              // 3.16.5 取消大文件上传
  },
} as const;

// ==================== 信息流 API (3.17) ====================
export const FEED_ENDPOINTS = {
  PROTECTED: {
    BASE: "/protected/feed",
    GET_LIST: "/get_list",                      // 3.17.1 获取信息流
    FEEDBACK: "/feedback",                      // 3.17.2 反馈信息流条目
  },
} as const;

// ==================== 标签管理 API (3.19) ====================
export const TAG_ENDPOINTS = {
  PUBLIC: {
    BASE: "/public/tags",
    GET_TRENDING: "/get_trending",              // 3.19.1 获取热门标签
    SUGGEST: "/suggest",                        // 3.19.2 标签搜索建议
  },
} as const;

// ==================== 系统配置管理 API (3.20) ====================
export const ADMIN_CONFIG_ENDPOINTS = {
  ADMIN: {
    BASE: "/admin/config",
    CREATE: "/create",                          // 3.20.1 创建配置
    GET_CONFIG: "/get_config",                  // 3.20.2 获取单个配置
    GET_LIST: "/get_list",                      // 3.20.3 获取配置列表
    UPDATE: "/update",                          // 3.20.4 更新配置
    DELETE: "/delete",                          // 3.20.5 删除配置
    GET_CONFIG_HISTORY: "/get_config_history",  // 3.20.6 获取配置变更历史
    LIST_ALL_CHANGES: "/list_all_changes",      // 3.20.7 获取所有配置变更
  },
} as const;

// ==================== 其他端点 ====================
export const OTHER_ENDPOINTS = {
  WS: API_CONFIG.WS_PATH,
} as const;

// ==================== 辅助函数 ====================

/**
 * 构建完整URL路径
 */
export const buildUrl = (base: string, endpoint: string): string => {
  return `${base}${endpoint}`;
};

/**
 * 构建认证URL
 */
export const buildAuthUrl = (endpoint: string, isProtected: boolean = false): string => {
  const base = isProtected ? AUTH_ENDPOINTS.PROTECTED.BASE : AUTH_ENDPOINTS.PUBLIC.BASE;
  return buildUrl(base, endpoint);
};

/**
 * 构建用户URL
 */
export const buildUserUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? USER_ENDPOINTS.PUBLIC.BASE : USER_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

/**
 * 构建群组URL
 */
export const buildGroupUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? GROUP_ENDPOINTS.PUBLIC.BASE : GROUP_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

/**
 * 构建瞬间URL
 */
export const buildMomentUrl = (endpoint: string, isPublic: boolean = false): string => {
  const base = isPublic ? MOMENT_ENDPOINTS.PUBLIC.BASE : MOMENT_ENDPOINTS.PROTECTED.BASE;
  return buildUrl(base, endpoint);
};

// ==================== 导出所有端点配置 ====================
export const API_ENDPOINTS = {
  AUTH: AUTH_ENDPOINTS,
  USER: USER_ENDPOINTS,
  MOMENT: MOMENT_ENDPOINTS,
  COMMENT: COMMENT_ENDPOINTS,
  PRIVATE_CHAT: PRIVATE_CHAT_ENDPOINTS,
  GROUP: GROUP_ENDPOINTS,
  PRIVATE_MESSAGE: PRIVATE_MESSAGE_ENDPOINTS,
  GROUP_MESSAGE: GROUP_MESSAGE_ENDPOINTS,
  MESSAGE_COMMON: MESSAGE_COMMON_ENDPOINTS,
  REPORT: REPORT_ENDPOINTS,
  SUBSCRIPTION: SUBSCRIPTION_ENDPOINTS,
  INBOX: INBOX_ENDPOINTS,
  SYSTEM: SYSTEM_ENDPOINTS,
  LEADERBOARD: LEADERBOARD_ENDPOINTS,
  MEDIA: MEDIA_ENDPOINTS,
  FEED: FEED_ENDPOINTS,
  TAG: TAG_ENDPOINTS,
  ADMIN_CONFIG: ADMIN_CONFIG_ENDPOINTS,
  OTHER: OTHER_ENDPOINTS,
} as const;
