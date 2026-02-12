use sqlx::PgPool;

use crate::error::Result;

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
}
