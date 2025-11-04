// 默认位置（南天门）
pub const DEFAULT_LATITUDE: f64 = 35.154922;
pub const DEFAULT_LONGITUDE: f64 = 112.682524;
pub const DEFAULT_LOCATION_NAME: &str = "南天门";

// 位置变化阈值（米）
pub const LOCATION_CHANGE_THRESHOLD: f64 = 500.0;

// 系统用户ID（预定义的官方账号）
pub const SYSTEM_USER_ID: i64 = 216172782113783809; // 0x0300000000000001 - 系统通知
pub const SECURITY_CENTER_USER_ID: i64 = 216172782113783810; // 0x0300000000000002 - 安全中心
pub const OPERATIONS_TEAM_USER_ID: i64 = 216172782113783811; // 0x0300000000000003 - 运营团队

// 用户类型
pub const USER_TYPE_TEMP: u8 = 0x01;
pub const USER_TYPE_NORMAL: u8 = 0x02;
pub const USER_TYPE_SYSTEM: u8 = 0x03;

// Redis 键前缀
pub const REDIS_KEY_ONLINE: &str = "online";
pub const REDIS_KEY_UNREAD: &str = "unread";
pub const REDIS_KEY_CONFIG: &str = "config";
pub const REDIS_KEY_LEADERBOARD: &str = "leaderboard";
pub const REDIS_KEY_FEED: &str = "feed";
pub const REDIS_KEY_TYPING: &str = "typing";
pub const REDIS_KEY_SESSION: &str = "session";

// Redis Pub/Sub 频道
pub const REDIS_CHANNEL_CONFIG_UPDATES: &str = "config_updates";
pub const REDIS_CHANNEL_WEBSOCKET_EVENTS: &str = "websocket_events";

// Token 有效期（秒）
pub const ACCESS_TOKEN_TTL: i64 = 3600; // 1小时
pub const REFRESH_TOKEN_TTL: i64 = 2592000; // 30天

// 上传限制（字节）
pub const MAX_UPLOAD_SIZE_AVATAR: usize = 5_242_880; // 5MB
pub const MAX_UPLOAD_SIZE_IMAGE: usize = 10_485_760; // 10MB

// 频率限制（次/小时）
pub const SEARCH_RATE_LIMIT: u32 = 100;
pub const MESSAGE_RATE_LIMIT: u32 = 1000;

// Redis TTL（秒）
pub const REDIS_TTL_ONLINE_STATUS: i64 = 300; // 5分钟
pub const REDIS_TTL_CONFIG: i64 = 60; // 1分钟
pub const REDIS_TTL_LEADERBOARD: i64 = 3600; // 1小时
pub const REDIS_TTL_FEED: i64 = 300; // 5分钟
pub const REDIS_TTL_TYPING: i64 = 10; // 10秒

// 分页限制
pub const DEFAULT_PAGE_LIMIT: i64 = 20;
pub const MAX_PAGE_LIMIT: i64 = 100;

// 地理位置限制
pub const DEFAULT_SEARCH_RADIUS: f64 = 5000.0; // 5公里
pub const MAX_SEARCH_RADIUS: f64 = 50000.0; // 50公里

