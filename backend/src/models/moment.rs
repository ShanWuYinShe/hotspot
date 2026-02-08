use crate::types::enums::ReactionType;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Moment {
    pub moment_id: uuid::Uuid,
    pub user_id: i64,
    pub description: Option<String>,
    // location: handled separately or via specialized struct
    pub location_name: Option<String>,
    pub privacy_level: Option<String>,

    pub view_count: Option<i32>,
    pub like_count: Option<i32>,
    pub comments_count: Option<i32>,
    pub share_count: Option<i32>,

    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentImage {
    pub image_id: uuid::Uuid,
    pub moment_id: uuid::Uuid,
    pub file_id: Option<uuid::Uuid>,
    pub url: String,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub sort_order: Option<i32>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentReaction {
    pub moment_id: uuid::Uuid,
    pub user_id: i64,
    pub type_: Option<ReactionType>, // 'type' is a keyword, mapped via sqlx rename if needed, or stick to snake_case in sql
    pub created_at: DateTime<Utc>,
}
