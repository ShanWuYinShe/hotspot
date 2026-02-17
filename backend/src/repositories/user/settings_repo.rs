use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::{PrivacyLevel, PrivateChatPermission};

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserSettingsRow {
    pub user_id: i64,
    pub language: String,
    pub theme: String,
    pub notifications_enabled: bool,
    pub privacy_settings: serde_json::Value,
    pub preferences: serde_json::Value,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Default)]
pub struct UpdateSettingsParams {
    pub privacy_level: Option<PrivacyLevel>,
    pub location_visible: Option<bool>,
    pub presence_visible: Option<bool>,
    pub social_lists_visible: Option<bool>,
    pub groups_visible: Option<bool>,
    pub liked_moments_visible: Option<bool>,
    pub stats_visible: Option<bool>,
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<PrivateChatPermission>,
    pub searchable: Option<bool>,
    pub phone_searchable: Option<bool>,
    pub nickname_searchable: Option<bool>,
    pub location_searchable: Option<bool>,
}

#[derive(Clone)]
pub struct UserSettingsRepository {
    pool: PgPool,
}

impl UserSettingsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 初始化用户设置
    pub async fn create(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            "INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING",
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn find_by_user_id(&self, user_id: i64) -> Result<Option<UserSettingsRow>> {
        let row = sqlx::query_as!(
            UserSettingsRow,
            r#"
            SELECT user_id,
                   COALESCE(language, 'zh-CN') as "language!",
                   COALESCE(theme, 'system') as "theme!",
                   COALESCE(notifications_enabled, TRUE) as "notifications_enabled!",
                   COALESCE(privacy_settings, '{}'::jsonb) as "privacy_settings!",
                   COALESCE(preferences, '{}'::jsonb) as "preferences!",
                   updated_at
            FROM user_settings
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;
        Ok(row)
    }

    pub async fn get_batch(&self, user_ids: &[i64]) -> Result<Vec<UserSettingsRow>> {
        let rows = sqlx::query_as!(
            UserSettingsRow,
            r#"
            SELECT user_id,
                   COALESCE(language, 'zh-CN') as "language!",
                   COALESCE(theme, 'system') as "theme!",
                   COALESCE(notifications_enabled, TRUE) as "notifications_enabled!",
                   COALESCE(privacy_settings, '{}'::jsonb) as "privacy_settings!",
                   COALESCE(preferences, '{}'::jsonb) as "preferences!",
                   updated_at
            FROM user_settings
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    pub async fn update_privacy(&self, user_id: i64, settings: serde_json::Value) -> Result<()> {
        sqlx::query!(
            "UPDATE user_settings SET privacy_settings = privacy_settings || $2, updated_at = NOW() WHERE user_id = $1",
            user_id, settings
        ).execute(&self.pool).await?;
        Ok(())
    }

    pub async fn update(&self, user_id: i64, params: &UpdateSettingsParams) -> Result<()> {
        let mut map = serde_json::Map::new();

        if let Some(value) = params.privacy_level {
            map.insert("privacy_level".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.location_visible {
            map.insert("location_visible".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.presence_visible {
            map.insert("presence_visible".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.social_lists_visible {
            map.insert("friends_visible".to_string(), serde_json::json!(value));
            map.insert("following_visible".to_string(), serde_json::json!(value));
            map.insert("followers_visible".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.groups_visible {
            map.insert("groups_visible".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.liked_moments_visible {
            map.insert(
                "liked_moments_visible".to_string(),
                serde_json::json!(value),
            );
        }
        if let Some(value) = params.stats_visible {
            map.insert("stats_visible".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.allow_follow {
            map.insert("allow_follow".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.allow_private_chat {
            map.insert("allow_private_chat".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.searchable {
            map.insert("searchable".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.phone_searchable {
            map.insert("phone_searchable".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.nickname_searchable {
            map.insert("nickname_searchable".to_string(), serde_json::json!(value));
        }
        if let Some(value) = params.location_searchable {
            map.insert("location_searchable".to_string(), serde_json::json!(value));
        }

        if map.is_empty() {
            return Ok(());
        }

        self.update_privacy(user_id, Value::Object(map)).await
    }
}
