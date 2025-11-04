use sqlx::PgPool;

use crate::{
    error::Result,
    models::device_token::DeviceToken,
};

#[derive(Debug, Clone)]
pub struct DeviceTokenRepository {
    db: PgPool,
}

impl DeviceTokenRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 注册或更新设备Token
    pub async fn upsert(&self, user_id: i64, platform: &str, token: &str) -> Result<DeviceToken> {
        let device_token = sqlx::query_as!(
            DeviceToken,
            r#"
            INSERT INTO device_tokens (user_id, platform, token, is_active)
            VALUES ($1, $2, $3, TRUE)
            ON CONFLICT (user_id, platform, token)
            DO UPDATE SET 
                updated_at = CURRENT_TIMESTAMP,
                is_active = TRUE
            RETURNING id, user_id, platform, token, created_at, updated_at, last_used_at, is_active
            "#,
            user_id,
            platform,
            token
        )
        .fetch_one(&self.db)
        .await?;

        Ok(device_token)
    }

    /// 注销设备Token（软删除，设置为不活跃）
    pub async fn deactivate(&self, user_id: i64, token: &str) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE device_tokens
            SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
            WHERE user_id = $1 AND token = $2
            "#,
            user_id,
            token
        )
        .execute(&self.db)
        .await?;

        Ok(result.rows_affected())
    }

    /// 注销用户所有设备Token
    pub async fn deactivate_all(&self, user_id: i64) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE device_tokens
            SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.db)
        .await?;

        Ok(result.rows_affected())
    }

    /// 获取用户所有活跃的设备Token
    pub async fn find_active_by_user(&self, user_id: i64) -> Result<Vec<DeviceToken>> {
        let tokens = sqlx::query_as!(
            DeviceToken,
            r#"
            SELECT id, user_id, platform, token, created_at, updated_at, last_used_at, is_active
            FROM device_tokens
            WHERE user_id = $1 AND is_active = TRUE
            ORDER BY updated_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.db)
        .await?;

        Ok(tokens)
    }

    /// 更新Token最后使用时间
    pub async fn update_last_used(&self, id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE device_tokens
            SET last_used_at = CURRENT_TIMESTAMP
            WHERE id = $1
            "#,
            id
        )
        .execute(&self.db)
        .await?;

        Ok(())
    }

    /// 删除不活跃的旧Token（清理任务）
    pub async fn delete_inactive_tokens(&self, days: i32) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE is_active = FALSE 
              AND updated_at < CURRENT_TIMESTAMP - INTERVAL '1 day' * $1
            "#,
            days as f64
        )
        .execute(&self.db)
        .await?;

        Ok(result.rows_affected())
    }
}

