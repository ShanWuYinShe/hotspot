use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone, FromRow)]
pub struct UserLocationHistoryRow {
    pub history_id: Uuid,
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct UserLocationHistoryRepository {
    pool: PgPool,
}

impl UserLocationHistoryRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn record(&self, user_id: i64, lon: f64, lat: f64) -> Result<()> {
        sqlx::query!(
            "INSERT INTO user_location_history (user_id, location) VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)",
            user_id, lon, lat
        ).execute(&self.pool).await?;
        Ok(())
    }

    pub async fn get_history(&self, user_id: i64, limit: i64) -> Result<Vec<UserLocationHistoryRow>> {
        let rows = sqlx::query_as!(
            UserLocationHistoryRow,
            "SELECT history_id, user_id, created_at FROM user_location_history WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2",
            user_id, limit
        ).fetch_all(&self.pool).await?;
        Ok(rows)
    }
}
