use serde::{Deserialize, Serialize};
use sqlx::FromRow;

/// 事件订阅配置模型（映射到 event_subscriptions 表）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct EventSubscription {
    pub id: i64,
    pub user_id: i64,
    
    // 社交互动订阅
    pub friend_request_enabled: bool,
    pub friend_accepted_enabled: bool,
    pub group_invite_enabled: bool,
    pub group_invite_response_enabled: bool,
    
    // 消息提及订阅
    pub message_mention_enabled: bool,
    
    // 瞬间互动订阅
    pub moment_like_enabled: bool,
    pub moment_comment_enabled: bool,
    pub comment_reply_enabled: bool,
    
    // 系统通知订阅
    pub system_announcement_enabled: bool,
    
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

impl EventSubscription {
    /// 创建默认订阅配置（所有类型默认启用）
    pub fn new_default(user_id: i64) -> Self {
        let now = chrono::Utc::now();
        Self {
            id: 0, // 由数据库自动生成
            user_id,
            friend_request_enabled: true,
            friend_accepted_enabled: true,
            group_invite_enabled: true,
            group_invite_response_enabled: true,
            message_mention_enabled: true,
            moment_like_enabled: true,
            moment_comment_enabled: true,
            comment_reply_enabled: true,
            system_announcement_enabled: true,
            created_at: now,
            updated_at: now,
        }
    }
}
