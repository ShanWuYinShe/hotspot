use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct Comment {
    pub comment_id: Uuid,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub content: String,
    pub parent_id: Option<Uuid>,
    pub reply_to_user_id: Option<i64>,
    pub created_at: DateTime<Utc>,
}

