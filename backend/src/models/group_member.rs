use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct GroupMember {
    pub id: i64,
    pub group_id: Uuid,
    pub user_id: i64,
    pub role: String,
    pub joined_at: DateTime<Utc>,
    pub is_muted: bool,
    pub muted_until: Option<DateTime<Utc>>,
}

