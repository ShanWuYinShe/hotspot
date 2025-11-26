use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::str::FromStr;

use crate::error::Result;
use crate::types::enums::Platform;

/// 设备Token Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct DeviceTokenRow {
    pub id: i64,
    pub user_id: i64,
    pub platform: Platform,
    pub device_token: String,
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

    /// 注册设备Token
    pub async fn register(
        &self,
        user_id: i64,
        platform: Platform,
        device_token: String,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO device_tokens (user_id, platform, device_token)
            VALUES ($1, $2::platform, $3)
            ON CONFLICT (user_id, platform, device_token) DO NOTHING
            "#,
            user_id,
            platform as Platform,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 注销设备Token
    pub async fn unregister(
        &self,
        user_id: i64,
        platform: Platform,
        device_token: String,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE user_id = $1 AND platform = $2::platform AND device_token = $3
            "#,
            user_id,
            platform as Platform,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取用户的所有设备Token
    pub async fn find_by_user(&self, user_id: i64) -> Result<Vec<DeviceTokenRow>> {
        let rows = sqlx::query_as!(
            DeviceTokenRow,
            r#"
            SELECT id, user_id, platform as "platform: Platform", device_token, created_at
            FROM device_tokens
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 注册或更新设备Token (upsert)
    pub async fn upsert(
        &self,
        user_id: i64,
        platform: &str,
        device_token: &str,
    ) -> Result<()> {
        let platform_enum = Platform::from_str(platform)
            .map_err(|e| crate::error::AppError::BadRequest(e))?;
            
        self.register(user_id, platform_enum, device_token.to_string()).await
    }

    /// 注销指定Token (deactivate)
    pub async fn deactivate(
        &self,
        user_id: i64,
        device_token: &str,
    ) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE user_id = $1 AND device_token = $2
            "#,
            user_id,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    /// 注销所有设备Token (deactivate_all)
    pub async fn deactivate_all(
        &self,
        user_id: i64,
    ) -> Result<u64> {
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
