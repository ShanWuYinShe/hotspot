use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;
use crate::types::enums::MessageType;

/// 消息模型（映射到 messages 表，统一私聊和群聊消息）
#[derive(Debug, Clone, FromRow)]
pub struct Message {
    pub id: i64,
    pub message_id: Uuid,
    pub room_id: Uuid,
    pub sender_id: Option<i64>,
    pub content: String,
    pub message_type: MessageType,
    pub is_edited: bool,
    pub created_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub reply_to: Option<Uuid>,
    pub status: String,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub mentions: Option<Vec<i64>>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
    pub system_event: Option<String>,
    pub affected_users: Option<Vec<i64>>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
}

