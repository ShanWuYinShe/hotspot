//! 系统常量定义
//! 
//! 这些常量用于系统配置，部分常量可能暂未使用，但为未来功能预留

#![allow(dead_code)]

// ============================================================
// 默认位置（南天门）
// ============================================================
pub const DEFAULT_LATITUDE: f64 = 35.154921;
pub const DEFAULT_LONGITUDE: f64 = 112.68253;
pub const DEFAULT_LOCATION_NAME: &str = "南天门";

// 位置变化阈值（米）
pub const LOCATION_CHANGE_THRESHOLD: f64 = 500.0;

// ============================================================
// 系统用户ID（预定义的官方账号）
// 系统用户ID区间：1 - 1,000,000
// ============================================================
pub const SYSTEM_USER_ID: i64 = 1;           // 系统通知
pub const SECURITY_CENTER_USER_ID: i64 = 2;  // 安全中心
pub const OPERATIONS_TEAM_USER_ID: i64 = 3;  // 运营团队

// 系统用户ID区间边界
pub const SYSTEM_USER_ID_MAX: i64 = 1_000_000;     // 系统用户区间上限
pub const REGULAR_USER_ID_START: i64 = 1_000_001;  // 普通用户区间起始

// ============================================================
// Redis 键前缀
// ============================================================
pub const REDIS_KEY_ONLINE: &str = "online";
pub const REDIS_KEY_UNREAD: &str = "unread";
pub const REDIS_KEY_CONFIG: &str = "config";
pub const REDIS_KEY_FEED: &str = "feed";
pub const REDIS_KEY_TYPING: &str = "typing";
pub const REDIS_KEY_SESSION: &str = "session";
pub const REDIS_KEY_PRESENCE_ONLINE: &str = "presence:online_users";
pub const REDIS_KEY_PRESENCE_LOCATION: &str = "presence:location:";

// ============================================================
// Redis Pub/Sub 频道
// ============================================================
pub const REDIS_CHANNEL_CONFIG_UPDATES: &str = "config_updates";
pub const REDIS_CHANNEL_WEBSOCKET_EVENTS: &str = "websocket_events";

// ============================================================
// Token 有效期（秒）
// ============================================================
pub const ACCESS_TOKEN_TTL: i64 = 3600; // 1小时
pub const REFRESH_TOKEN_TTL: i64 = 2592000; // 30天

// ============================================================
// 上传限制（字节）
// ============================================================
pub const MAX_UPLOAD_SIZE_AVATAR: usize = 5_242_880; // 5MB
pub const MAX_UPLOAD_SIZE_IMAGE: usize = 10_485_760; // 10MB

// ============================================================
// 频率限制（次/小时）
// ============================================================
pub const SEARCH_RATE_LIMIT: u32 = 100;
pub const MESSAGE_RATE_LIMIT: u32 = 1000;

// ============================================================
// Redis TTL（秒）
// ============================================================
pub const REDIS_TTL_ONLINE_STATUS: u64 = 300; // 5分钟
pub const REDIS_TTL_CONFIG: i64 = 60; // 1分钟
pub const REDIS_TTL_FEED: i64 = 300; // 5分钟
pub const REDIS_TTL_TYPING: i64 = 10; // 10秒

// ============================================================
// 分页限制
// ============================================================
pub const DEFAULT_PAGE_LIMIT: i64 = 20;
pub const MAX_PAGE_LIMIT: i64 = 100;

// ============================================================
// 地理位置限制
// ============================================================
pub const DEFAULT_SEARCH_RADIUS: f64 = 5000.0; // 5公里
pub const MAX_SEARCH_RADIUS: f64 = 50000.0; // 50公里

// ============================================================
// 时间窗口（天/小时/分钟）
// ============================================================
pub const TRENDING_DAILY_DAYS: i64 = 1;
pub const TRENDING_WEEKLY_DAYS: i64 = 7;
pub const TRENDING_MONTHLY_DAYS: i64 = 30;

pub const MESSAGE_RECALL_MINUTES: i64 = 2; // 消息撤回时间窗口
pub const INVITE_EXPIRE_DAYS: i64 = 7; // 邀请过期时间
pub const UPLOAD_SESSION_HOURS: i64 = 24; // 上传会话过期时间

// ============================================================
// WebSocket 心跳配置
// ============================================================
/// 客户端心跳间隔（秒）
pub const WS_HEARTBEAT_INTERVAL_SECS: u64 = 45;
/// 心跳响应超时时间（秒）
pub const WS_HEARTBEAT_TIMEOUT_SECS: u64 = 10;
/// 最大允许丢失心跳次数
pub const WS_MAX_MISSED_HEARTBEATS: u32 = 3;
/// 服务端心跳发送间隔（秒）- 略长于客户端间隔
pub const WS_SERVER_PING_INTERVAL_SECS: u64 = 60;
/// 服务端连接超时时间（秒）= HEARTBEAT_INTERVAL * 2 + HEARTBEAT_TIMEOUT
pub const WS_CONNECTION_TIMEOUT_SECS: u64 = WS_HEARTBEAT_INTERVAL_SECS * 2 + WS_HEARTBEAT_TIMEOUT_SECS;
