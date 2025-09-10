/// 作业类型定义

use serde::{Deserialize, Serialize};

/// 作业类型枚举
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum JobType {
    /// 用户位置初始化
    LocationInit,
    /// 欢迎邮件发送
    WelcomeEmail,
    /// 通知推送
    NotificationPush,
    /// 数据清理
    DataCleanup,
}

/// 作业状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum JobStatus {
    /// 待处理
    Pending,
    /// 处理中
    Processing,
    /// 已完成
    Completed,
    /// 失败
    Failed,
    /// 已取消
    Cancelled,
}

/// 作业优先级
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum JobPriority {
    /// 低优先级
    Low,
    /// 普通优先级
    Normal,
    /// 高优先级
    High,
    /// 紧急
    Urgent,
}

/// 作业配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JobConfig {
    /// 最大重试次数
    pub max_retries: u32,
    /// 超时时间（秒）
    pub timeout_seconds: u64,
    /// 优先级
    pub priority: JobPriority,
}

// === 作业类型常量 ===

/// 用户欢迎邮件作业类型
pub const USER_WELCOME_EMAIL: &str = "user_welcome_email";

/// 用户位置索引作业类型
pub const USER_LOCATION_INDEX: &str = "user_location_index";

/// 群组成员通知作业类型
pub const GROUP_MEMBER_NOTIFICATION: &str = "group_member_notification";

/// 数据清理作业类型
pub const DATA_CLEANUP: &str = "data_cleanup";

/// 通知推送作业类型
pub const NOTIFICATION_PUSH: &str = "notification_push";

/// 系统维护作业类型
pub const SYSTEM_MAINTENANCE: &str = "system_maintenance";

/// 瞬间内容索引作业类型
pub const MOMENT_CONTENT_INDEX: &str = "moment_content_index";

/// 消息通知作业类型
pub const MESSAGE_NOTIFICATION: &str = "message_notification";

/// 清理过期会话作业类型
pub const CLEANUP_EXPIRED_SESSIONS: &str = "cleanup_expired_sessions";


