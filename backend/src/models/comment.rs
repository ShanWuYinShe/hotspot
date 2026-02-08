use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Comment {
    pub comment_id: uuid::Uuid,
    pub moment_id: uuid::Uuid,
    pub user_id: i64,
    pub content: String,
    pub parent_id: Option<uuid::Uuid>,
    pub reply_to_comment_id: Option<uuid::Uuid>,
    pub reply_to_user_id: Option<i64>,
    pub like_count: Option<i32>,
    pub reply_count: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}
