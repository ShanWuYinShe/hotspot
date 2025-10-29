use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct Moment {
    pub moment_id: Uuid,
    pub user_id: i64,
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Option<Vec<String>>,
    pub images: Option<Vec<String>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

