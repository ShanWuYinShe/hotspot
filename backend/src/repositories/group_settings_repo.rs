//! 群组设置仓库
//!
//! 管理群组的加入方式、密码、成员上限等设置

#![allow(dead_code)]

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};

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

    /// 获取群组设置
    pub async fn get(&self, group_chat_id: Uuid) -> Result<GroupSettings> {
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
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            if matches!(e, sqlx::Error::RowNotFound) {
                AppError::NotFound("群组设置不存在".to_string())
            } else {
                AppError::from(e)
            }
        })?;

        Ok(settings)
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

    /// 批量获取群组设置
    pub async fn get_batch(&self, group_chat_ids: &[Uuid]) -> Result<Vec<GroupSettings>> {
        if group_chat_ids.is_empty() {
            return Ok(Vec::new());
        }

        let settings = sqlx::query_as!(
            GroupSettings,
            r#"
            SELECT group_chat_id, join_method::TEXT as "join_method!", 
                   is_searchable, max_members, password_hash,
                   created_at, updated_at
            FROM group_settings
            WHERE group_chat_id = ANY($1)
            "#,
            group_chat_ids as &[Uuid]
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(settings)
    }

    /// 检查群组是否可搜索
    pub async fn is_searchable(&self, group_chat_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT is_searchable
            FROM group_settings
            WHERE group_chat_id = $1
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.is_searchable)
    }

    /// 获取群组最大成员数
    pub async fn get_max_members(&self, group_chat_id: Uuid) -> Result<Option<i32>> {
        let result = sqlx::query!(
            r#"
            SELECT max_members
            FROM group_settings
            WHERE group_chat_id = $1
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.max_members)
    }
}
