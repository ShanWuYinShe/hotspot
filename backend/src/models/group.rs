use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct Group {
    pub group_id: Uuid,
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub password_hash: Option<String>,
    pub max_members: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
}

