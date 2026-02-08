use crate::types::enums::ReactionType;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChat {
    pub chat_id: uuid::Uuid,
    pub user1_id: i64,
    pub user2_id: i64,
    pub last_message_id: Option<uuid::Uuid>,
    pub last_message_content: Option<String>,
    pub last_message_type: Option<String>,
    pub last_message_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessage {
    pub message_id: uuid::Uuid,
    pub chat_id: uuid::Uuid,
    pub sender_id: i64,
    pub receiver_id: i64,
    pub content: Option<String>,
    pub message_type: Option<String>,
    pub metadata: Option<serde_json::Value>,
    pub reply_to_id: Option<uuid::Uuid>,
    pub is_recalled: Option<bool>,
    pub reaction_counts: Option<serde_json::Value>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMessage {
    pub message_id: uuid::Uuid,
    pub group_id: i64,
    pub sender_id: i64,
    pub content: Option<String>,
    pub message_type: Option<String>,
    pub metadata: Option<serde_json::Value>,
    pub reply_to_id: Option<uuid::Uuid>,
    pub is_recalled: Option<bool>,
    pub reaction_counts: Option<serde_json::Value>,
    pub at_users: Option<serde_json::Value>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ChatConversationState {
    pub user_id: i64,
    pub chat_type: String,
    pub chat_id: String,
    pub unread_count: Option<i32>,
    pub last_read_message_id: Option<uuid::Uuid>,
    pub is_pinned: Option<bool>,
    pub is_muted: Option<bool>,
    pub is_hidden: Option<bool>,
    pub updated_at: DateTime<Utc>,
}
