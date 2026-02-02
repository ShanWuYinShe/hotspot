//! 群组设置仓库
//!
//! 管理群组的加入方式、密码、成员上限等设置

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

/// 群组设置Row（如果group_repo.rs中已定义，可以从那里导入）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupSettings {
    pub group_chat_id: Uuid,
    pub join_method: String,
    pub is_searchable: bool,
    pub max_members: Option<i32>,
    pub password_hash: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

/// 用户群组偏好设置Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupUserPreferences {
    pub id: i64,
    pub group_chat_id: Uuid,
    pub user_id: i64,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Clone)]
pub struct GroupSettingsRepository {
    pool: PgPool,
}

impl GroupSettingsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 根据群聊ID查找群组设置（静态方法）
    pub async fn find_by_group_id(
        pool: &PgPool,
        group_chat_id: Uuid,
    ) -> Result<Option<GroupSettings>> {
        let settings = sqlx::query_as!(
            GroupSettings,
            r#"
            SELECT group_chat_id, join_method::TEXT as "join_method!", 
                   is_searchable, max_members, password_hash,
                   created_at, updated_at
            FROM group_settings
            WHERE group_chat_id = $1
            "#,
            group_chat_id
        )
        .fetch_optional(pool)
        .await?;

        Ok(settings)
    }

    /// 根据群聊ID查找群组设置（实例方法）
    pub async fn find_by_group(&self, group_chat_id: Uuid) -> Result<Option<GroupSettings>> {
        Self::find_by_group_id(&self.pool, group_chat_id).await
    }

    /// 更新群组设置
    pub async fn update(
        &self,
        group_chat_id: Uuid,
        join_method: Option<&str>,
        is_searchable: Option<bool>,
        max_members: Option<i32>,
        password_hash: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_settings
            SET join_method = COALESCE($2, join_method::TEXT)::join_method,
                is_searchable = COALESCE($3, is_searchable),
                max_members = COALESCE($4, max_members),
                password_hash = COALESCE($5, password_hash),
                updated_at = NOW()
            WHERE group_chat_id = $1
            "#,
            group_chat_id,
            join_method,
            is_searchable,
            max_members,
            password_hash
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 清除群组密码
    pub async fn clear_password(&self, group_chat_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_settings
            SET password_hash = NULL,
                updated_at = NOW()
            WHERE group_chat_id = $1
            "#,
            group_chat_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 创建群组设置（通常由触发器自动创建，此方法用于特殊情况）
    pub async fn create(
        &self,
        group_chat_id: Uuid,
        join_method: &str,
        is_searchable: bool,
        max_members: Option<i32>,
        password_hash: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO group_settings (group_chat_id, join_method, is_searchable, max_members, password_hash)
            VALUES ($1, $2::TEXT::join_method, $3, $4, $5)
            ON CONFLICT (group_chat_id) DO NOTHING
            "#,
            group_chat_id,
            join_method,
            is_searchable,
            max_members,
            password_hash
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    // ==================== 用户群组偏好设置 ====================

    /// 获取用户群组偏好设置
    pub async fn find_user_preferences(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupUserPreferences>> {
        let prefs = sqlx::query_as!(
            GroupUserPreferences,
            r#"
            SELECT id, group_chat_id, user_id, is_muted, is_pinned, created_at, updated_at
            FROM group_user_preferences
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(prefs)
    }

    /// 创建或更新用户群组偏好设置
    pub async fn upsert_user_preferences(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO group_user_preferences (group_chat_id, user_id, is_muted, is_pinned)
            VALUES ($1, $2, COALESCE($3, FALSE), COALESCE($4, FALSE))
            ON CONFLICT (group_chat_id, user_id) DO UPDATE
            SET is_muted = COALESCE($3, group_user_preferences.is_muted),
                is_pinned = COALESCE($4, group_user_preferences.is_pinned),
                updated_at = NOW()
            "#,
            group_chat_id,
            user_id,
            is_muted,
            is_pinned
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
