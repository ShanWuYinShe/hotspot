//! 用户相关的任务定义
use serde::{Deserialize, Serialize};

/// 欢迎邮件任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WelcomeEmailJob {
    pub user_id: i64,
    pub nickname: String,
    pub email: Option<String>,
}

/// 位置索引更新任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationIndexJob {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
    pub timestamp: i64,
}

/// 通知推送任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationJob {
    pub user_id: i64,
    pub notification_type: NotificationType,
    pub payload: String,
    pub timestamp: i64,
}

/// 通知类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum NotificationType {
    Message,        // 聊天消息
    System,         // 系统通知
    Moment,         // 动态通知
    GroupInvite,    // 群组邀请
    FriendRequest,  // 好友请求
    Welcome,        // 欢迎通知
    LocationUpdate, // 位置更新通知
}

/// 用户摘要任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserDigestJob {
    pub user_id: i64,
    pub digest_type: String,
    pub timestamp: i64,
}
