//! 系统常量定义
//!
//! 这些常量用于系统配置，部分常量可能暂未使用，但为未来功能预留

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
// 分页限制
// ============================================================
pub const DEFAULT_PAGE_LIMIT: i64 = 20;
/// 分页单页最大条数
pub const MAX_PAGE_LIMIT: i32 = 100;

// ============================================================
// 用户展示
// ============================================================
/// 当用户昵称不可用时的占位展示
pub const UNKNOWN_USER_NICKNAME: &str = "未知用户";

// ============================================================
// 时间常量（秒）
// ============================================================
pub const SECS_PER_MINUTE: u64 = 60;
pub const SECS_5_MINUTES: u64 = 300;
pub const SECS_10_MINUTES: u64 = 600;
pub const SECS_PER_HOUR: u64 = 3600;
pub const SECS_PER_DAY: u64 = 86400;

// ============================================================
// 时间窗口（天/小时/分钟）
// ============================================================
pub const TRENDING_WEEKLY_DAYS: i64 = 7;
/// 热门标签默认返回条数（Feed 热门请求无 limit 时）
pub const DEFAULT_TRENDING_TAGS_LIMIT: i64 = 50;

pub const MESSAGE_RECALL_MINUTES: i64 = 2; // 消息撤回时间窗口
pub const INVITE_EXPIRE_DAYS: i64 = 7; // 邀请过期时间
/// 陌生人私聊消息每日发送上限（参考 spec/api_spec_chat.md）
pub const STRANGER_MESSAGE_LIMIT_PER_DAY: i64 = 10;

// ============================================================
// 地理位置
// ============================================================
/// 最大搜索半径（米）
pub const MAX_SEARCH_RADIUS: f64 = 50000.0;

// ============================================================
// 地图渲染
// ============================================================
/// 聚类格子尺寸（像素），用于计算 grid_meters
pub const MAP_CLUSTER_GRID_PX: f64 = 60.0;
/// 聚合/明细切换的 zoom 阈值（<= 此值显示聚合点）
pub const MAP_CLUSTER_ZOOM_THRESHOLD: f64 = 12.0;
/// 明细模式下单类型实体查询上限
pub const MAP_DETAIL_ENTITY_LIMIT: i32 = 200;

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
