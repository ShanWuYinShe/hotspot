use chrono::{DateTime, Utc};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct UserRelationship {
    pub relationship_id: i64,
    pub from_user_id: i64,
    pub to_user_id: i64,
    pub relationship_type: String, // 'friend', 'following', 'blocked'
    pub status: String,            // 'pending', 'accepted', 'rejected'
    pub remark: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

