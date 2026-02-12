//! 系统常量定义
//!
//! 这些常量用于系统配置，部分常量可能暂未使用，但为未来功能预留

// ============================================================
// 默认位置（南天门）
// ============================================================
pub const DEFAULT_LATITUDE: f64 = 35.154922;
pub const DEFAULT_LONGITUDE: f64 = 112.682524;

// ============================================================
// Redis 键前缀
// ============================================================
pub const REDIS_KEY_CONFIG: &str = "config";
pub const REDIS_KEY_SESSION: &str = "session";
pub const REDIS_KEY_PRESENCE_ONLINE: &str = "presence:online_users";
pub const REDIS_KEY_PRESENCE_LOCATION: &str = "presence:location:";

// ============================================================
// Redis Pub/Sub 频道
// ============================================================
pub const REDIS_CHANNEL_CONFIG_UPDATES: &str = "config_updates";

// ============================================================
// Redis TTL（秒）
// ============================================================
// ============================================================
// 分页限制
// ============================================================
pub const DEFAULT_PAGE_LIMIT: i64 = 20;

// ============================================================
// 时间窗口（天/小时/分钟）
// ============================================================
pub const TRENDING_WEEKLY_DAYS: i64 = 7;

pub const MESSAGE_RECALL_MINUTES: i64 = 2; // 消息撤回时间窗口
pub const INVITE_EXPIRE_DAYS: i64 = 7; // 邀请过期时间

// ============================================================
// WebSocket 心跳配置
// ============================================================
/// 客户端心跳间隔（秒）
pub const WS_HEARTBEAT_INTERVAL_SECS: u64 = 30;
/// 心跳响应超时时间（秒）
pub const WS_HEARTBEAT_TIMEOUT_SECS: u64 = 10;
/// 服务端心跳发送间隔（秒）- 略长于客户端间隔
pub const WS_SERVER_PING_INTERVAL_SECS: u64 =
    WS_HEARTBEAT_INTERVAL_SECS + WS_HEARTBEAT_TIMEOUT_SECS;
/// 服务端连接超时时间（秒）- 与规范保持一致（90 秒）
pub const WS_CONNECTION_TIMEOUT_SECS: u64 = WS_HEARTBEAT_INTERVAL_SECS * 3;
