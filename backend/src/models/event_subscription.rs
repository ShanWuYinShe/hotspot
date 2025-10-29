use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct EventSubscription {
    pub id: Uuid,
    pub user_id: i64,
    pub event_type: String, // "private_message", "group_message", "friend_request", "group_invite", "moment", "system_notification"
    pub enabled: bool,
    pub push_enabled: bool,
    pub email_enabled: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

impl EventSubscription {
    pub fn new(user_id: i64, event_type: String) -> Self {
        let now = chrono::Utc::now();
        Self {
            id: Uuid::new_v4(),
            user_id,
            event_type,
            enabled: true,
            push_enabled: true,
            email_enabled: false,
            created_at: now,
            updated_at: now,
        }
    }

    /// 创建默认订阅配置
    pub fn create_defaults(user_id: i64) -> Vec<Self> {
        let event_types = vec![
            "private_message",
            "group_message",
            "friend_request",
            "group_invite",
            "moment",
            "system_notification",
        ];

        event_types
            .into_iter()
            .map(|event_type| Self::new(user_id, event_type.to_string()))
            .collect()
    }
}

