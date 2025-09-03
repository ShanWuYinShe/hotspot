//! WebSocket系统常量定义

use crate::utils::user_id_partition::UserIdPartition;
use crate::utils::JsSafeI64;

/// WebSocket连接相关常量
pub mod connection {
    /// 默认的广播通道容量
    #[allow(dead_code)]
    pub const DEFAULT_BROADCAST_CAPACITY: usize = 1000;
    
    /// 事件总线广播通道容量
    #[allow(dead_code)]
    pub const BROADCAST_CHANNEL_CAPACITY: usize = 1000;
    
    /// 房间广播通道容量
    #[allow(dead_code)]
    pub const ROOM_BROADCAST_CAPACITY: usize = 100;
    
    /// 连接超时时间（秒）
    #[allow(dead_code)]
    pub const CONNECTION_TIMEOUT_SECS: u64 = 30;
    
    /// 心跳检测间隔（秒）
    #[allow(dead_code)]
    pub const HEARTBEAT_INTERVAL_SECS: u64 = 30;
    
    /// 最大并发连接数
    pub const MAX_CONNECTIONS: usize = 10000;
    
    /// 最大消息大小（字节）
    #[allow(dead_code)]
    pub const MAX_MESSAGE_SIZE: usize = 64 * 1024; // 64KB
}

/// WebSocket消息相关常量
pub mod message {
    use super::JsSafeI64;
    /// 未知用户昵称
    #[allow(dead_code)]
    pub const UNKNOWN_USER_NICKNAME: &str = "未知用户";
    
    /// 系统用户ID
    #[allow(dead_code)]
    pub const SYSTEM_USER_ID_RAW: i64 = {
        // 使用系统用户类型，序列号为0
        4611686018427387904i64 // 预计算的系统用户ID
    };
    
    /// 系统用户ID（JavaScript安全）
    #[allow(dead_code)]
    pub fn system_user_id() -> JsSafeI64 {
        JsSafeI64::new(SYSTEM_USER_ID_RAW)
    }
    
    /// 消息重试次数
    #[allow(dead_code)]
    pub const MESSAGE_RETRY_COUNT: usize = 3;
}

/// WebSocket错误码
pub mod error_codes {
    pub const INVALID_JSON: &str = "INVALID_JSON";
    pub const FORBIDDEN: &str = "FORBIDDEN";
    #[allow(dead_code)]
    pub const INVALID_TOKEN: &str = "INVALID_TOKEN";
    #[allow(dead_code)]
    pub const INVALID_MESSAGE_TYPE: &str = "INVALID_MESSAGE_TYPE";
    #[allow(dead_code)]
    pub const INVALID_AUTH_MESSAGE: &str = "INVALID_AUTH_MESSAGE";
    #[allow(dead_code)]
    pub const UNAUTHORIZED: &str = "UNAUTHORIZED";
    #[allow(dead_code)]
    pub const ROOM_NOT_FOUND: &str = "ROOM_NOT_FOUND";
    pub const INTERNAL_ERROR: &str = "INTERNAL_ERROR";
    pub const CONNECTION_LIMIT_EXCEEDED: &str = "CONNECTION_LIMIT_EXCEEDED";
    #[allow(dead_code)]
    pub const MESSAGE_TOO_LARGE: &str = "MESSAGE_TOO_LARGE";
    #[allow(dead_code)]
    pub const ERROR_CODE_SEND_MESSAGE_FAILED: &str = "SEND_MESSAGE_FAILED";
}

/// WebSocket错误消息
pub mod error_messages {
    pub const INVALID_JSON_FORMAT: &str = "无效的JSON格式";
    pub const FORBIDDEN: &str = "禁止访问";
    #[allow(dead_code)]
    pub const INVALID_TOKEN: &str = "无效的认证令牌";
    #[allow(dead_code)]
    pub const NEED_TEXT_MESSAGE: &str = "需要文本消息进行认证";
    #[allow(dead_code)]
    pub const FIRST_MESSAGE_MUST_BE_AUTH: &str = "首条消息必须是连接认证消息";
    #[allow(dead_code)]
    pub const UNAUTHORIZED_ACCESS: &str = "未授权访问";
    #[allow(dead_code)]
    pub const ROOM_NOT_EXISTS: &str = "房间不存在";
    pub const INTERNAL_SERVER_ERROR: &str = "服务器内部错误";
    pub const TOO_MANY_CONNECTIONS: &str = "连接数量超过限制";
    #[allow(dead_code)]
    pub const MESSAGE_TOO_LARGE: &str = "消息大小超过限制";
}

/// 默认配置值
pub mod defaults {
    /// 默认分页大小
    #[allow(dead_code)]
    pub const DEFAULT_PAGE_SIZE: usize = 20;
    
    /// 最大分页大小
    #[allow(dead_code)]
    pub const MAX_PAGE_SIZE: usize = 100;
}

/// 获取系统用户ID（动态计算）
    #[allow(dead_code)]
    pub fn get_system_user_id() -> i64 {
        UserIdPartition::generate_system_user_id()
    }
