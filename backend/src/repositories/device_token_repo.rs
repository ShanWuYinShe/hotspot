use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
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
        let platform_str = platform.to_string();
        sqlx::query!(
            r#"
            INSERT INTO device_tokens (user_id, platform, device_token)
            VALUES ($1, $2::text::platform, $3)
            ON CONFLICT (user_id, platform, device_token) DO NOTHING
            "#,
            user_id,
            platform_str,
            device_token
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取用户的设备Token列表（分页）
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<DeviceTokenRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            use crate::utils::pagination::{decode_cursor, IdCursor};
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            DeviceTokenRow,
            r#"
            SELECT id, user_id, platform as "platform: Platform", device_token, created_at
            FROM device_tokens
            WHERE user_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|last| {
                use crate::utils::pagination::encode_cursor;
                encode_cursor(&last.id)
            })
            .transpose()?;

        Ok((rows, next_cursor))
    }

    /// 注册或更新设备Token (upsert)
    pub async fn upsert(&self, user_id: i64, platform: &str, device_token: &str) -> Result<()> {
        let platform_enum =
            Platform::from_str(platform).map_err(crate::error::AppError::BadRequest)?;

        self.register(user_id, platform_enum, device_token.to_string())
            .await
    }

    /// 注销指定Token
    pub async fn unregister(&self, user_id: i64, device_token: &str) -> Result<u64> {
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

    /// 注销用户所有设备Token
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
