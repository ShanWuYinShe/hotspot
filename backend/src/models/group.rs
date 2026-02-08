use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Group {
    pub group_id: i64,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,

    pub group_type: Option<String>,
    pub max_members: Option<i32>,
    pub is_muted: Option<bool>,
    pub join_type: Option<String>,

    pub metadata: Option<serde_json::Value>,
    // PostGIS types handling in Rust often requires specialized crate or manual parsing.
    // For simplicity in model, we might skip raw 'location' field if not explicitly selected,
    // or use custom type if `sqlx-postgis` is enabled.
    // For now, assume location is handled via specialized queries returning lat/long floats.
    // pub location: Option<...>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub dismissed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMember {
    pub group_id: i64,
    pub user_id: i64,
    pub nickname: Option<String>,
    pub role: Option<String>,
    pub is_muted: Option<bool>,
    pub muted_until: Option<DateTime<Utc>>,
    pub joined_at: DateTime<Utc>,
    pub inviter_id: Option<i64>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupStats {
    pub group_id: i64,
    pub member_count: i32,
    pub message_count: i32,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupAnnouncement {
    pub announcement_id: uuid::Uuid,
    pub group_id: i64,
    pub publisher_id: i64,
    pub title: String,
    pub content: String,
    pub is_pinned: Option<bool>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}
