use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Notification {
    pub notification_id: uuid::Uuid,
    pub user_id: i64,
    pub notification_type: String,
    pub is_read: bool,
    pub payload: serde_json::Value,
    pub created_at: DateTime<Utc>,
}
