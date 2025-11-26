use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::{AppError, Result};

/// 用户设置Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserSettings {
    pub user_id: i64,
    pub privacy_level: String, // 'public', 'friends', 'private'
    pub is_location_sharing: bool,
    pub can_be_followed: bool,
    pub allow_private_chat: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Clone)]
pub struct UserSettingsRepository {
    pool: PgPool,
}

impl UserSettingsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取用户设置
    pub async fn get(&self, user_id: i64) -> Result<UserSettings> {
        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT user_id, privacy_level::TEXT as "privacy_level!", 
                   is_location_sharing, can_be_followed, allow_private_chat,
                   created_at, updated_at
            FROM user_settings
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            if matches!(e, sqlx::Error::RowNotFound) {
                AppError::NotFound("用户设置不存在".to_string())
            } else {
                AppError::from(e)
            }
        })?;

        Ok(settings)
    }

    /// 更新用户设置
    pub async fn update(
        &self,
        user_id: i64,
        privacy_level: Option<&str>,
        is_location_sharing: Option<bool>,
        can_be_followed: Option<bool>,
        allow_private_chat: Option<bool>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_settings
            SET privacy_level = COALESCE($2::TEXT)::privacy_level,
                is_location_sharing = COALESCE($3, is_location_sharing),
                can_be_followed = COALESCE($4, can_be_followed),
                allow_private_chat = COALESCE($5, allow_private_chat),
                updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id,
            privacy_level,
            is_location_sharing,
            can_be_followed,
            allow_private_chat
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 创建用户设置（通常由触发器自动创建，此方法用于特殊情况）
    pub async fn create(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_settings (user_id)
            VALUES ($1)
            ON CONFLICT (user_id) DO NOTHING
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 批量获取用户设置
    pub async fn get_batch(&self, user_ids: &[i64]) -> Result<Vec<UserSettings>> {
        if user_ids.is_empty() {
            return Ok(Vec::new());
        }

        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT user_id, privacy_level::TEXT as "privacy_level!", 
                   is_location_sharing, can_be_followed, allow_private_chat,
                   created_at, updated_at
            FROM user_settings
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(settings)
    }
}
