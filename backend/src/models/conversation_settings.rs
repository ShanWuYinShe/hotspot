use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ConversationSettings {
    pub id: Uuid,
    pub conversation_id: Uuid,
    pub user_id: i64,
    pub mute_notifications: bool,
    pub show_preview: bool,
    pub pin_conversation: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

impl ConversationSettings {
    pub fn new(conversation_id: Uuid, user_id: i64) -> Self {
        let now = chrono::Utc::now();
        Self {
            id: Uuid::new_v4(),
            conversation_id,
            user_id,
            mute_notifications: false,
            show_preview: true,
            pin_conversation: false,
            created_at: now,
            updated_at: now,
        }
    }
}

