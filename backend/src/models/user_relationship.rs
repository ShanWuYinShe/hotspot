use chrono::{DateTime, Utc};
use sqlx::FromRow;
use crate::types::enums::{RelationshipType, RelationshipStatus};

#[derive(Debug, Clone, FromRow)]
pub struct UserRelationship {
    pub relationship_id: i64,
    pub from_user_id: i64,
    pub to_user_id: i64,
    pub relationship_type: RelationshipType,
    pub status: RelationshipStatus,
    pub remark: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

