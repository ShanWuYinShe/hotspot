use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct Message {
    pub message_id: Uuid,
    pub target_id: String,
    pub target_type: String,
    pub sender_id: Option<i64>,
    pub content: String,
    pub message_type: String,
    pub is_edited: bool,
    pub created_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub reply_to_id: Option<Uuid>,
    pub mentions: Option<Vec<i64>>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
    pub system_event: Option<String>,
    pub affected_users: Option<Vec<i64>>,
}

