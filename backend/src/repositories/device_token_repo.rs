use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::str::FromStr;

use crate::error::Result;
use crate::types::enums::DevicePlatform;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct DeviceTokenRow {
    pub token_id: uuid::Uuid,
    pub user_id: i64,
    pub platform: DevicePlatform,
    pub device_token: String,
    pub last_active_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct DeviceTokenRepository {
    pool: PgPool,
}

impl DeviceTokenRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn register(
        &self,
        user_id: i64,
        platform: DevicePlatform,
        device_token: String,
    ) -> Result<()> {
        let platform_str = platform.to_string();
        sqlx::query!(
            r#"
            INSERT INTO device_tokens (user_id, platform, device_token, last_active_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id, device_token) 
            DO UPDATE SET last_active_at = NOW(), platform = EXCLUDED.platform
            "#,
            user_id,
            platform_str,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn list_by_user(&self, user_id: i64) -> Result<Vec<DeviceTokenRow>> {
        let rows = sqlx::query_as!(
            DeviceTokenRow,
            r#"
            SELECT 
                token_id, user_id, 
                platform as "platform: DevicePlatform", 
                device_token, 
                last_active_at,
                created_at
            FROM device_tokens
            WHERE user_id = $1
            ORDER BY last_active_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn upsert(&self, user_id: i64, platform: &str, device_token: &str) -> Result<()> {
        let platform_enum =
            DevicePlatform::from_str(platform).map_err(crate::error::AppError::BadRequest)?;

        self.register(user_id, platform_enum, device_token.to_string())
            .await
    }

    pub async fn unregister(
        &self,
        user_id: i64,
        platform: DevicePlatform,
        device_token: &str,
    ) -> Result<u64> {
        let platform_str = platform.to_string();
        let result = sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE user_id = $1 AND platform = $2 AND device_token = $3
            "#,
            user_id,
            platform_str,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    pub async fn unregister_all(&self, user_id: i64) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }
}
