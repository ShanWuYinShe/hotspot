use chrono::{DateTime, Utc};
use serde::Serialize;
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct UserLocationHistory {
    pub id: i64,
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct GroupLocationHistory {
    pub id: i64,
    pub group_id: Uuid,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
}











