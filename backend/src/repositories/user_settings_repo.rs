//! 用户设置仓库
//!
//! 管理用户的隐私设置、社交设置、搜索设置等
//! 与 API 文档 UserSettingsData 完全对应

#![allow(dead_code)]

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::{AppError, Result};

/// 用户设置Row（与数据库表完全对应）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserSettings {
    pub user_id: i64,
    
    // ============ 隐私设置 ============
    pub privacy_level: String,           // 'public', 'friends', 'private'
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: String,         // 'public', 'friends', 'private'
    pub following_visible: String,       // 'public', 'friends', 'private'
    pub followers_visible: String,       // 'public', 'friends', 'private'
    pub moments_visible: String,         // 'public', 'friends', 'private'
    pub stats_visible: String,           // 'public', 'friends', 'private'
    
    // ============ 社交设置 ============
    pub allow_follow: bool,
    pub allow_private_chat: String,      // 'everyone', 'friends', 'nobody'
    
    // ============ 搜索设置 ============
    pub searchable: bool,
    pub phone_searchable: bool,
    pub nickname_searchable: bool,
    pub location_searchable: bool,
    
    // ============ 时间戳 ============
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

/// 用户设置更新参数
#[derive(Debug, Clone, Default)]
pub struct UpdateSettingsParams {
    // 隐私设置
    pub privacy_level: Option<String>,
    pub location_visible: Option<bool>,
    pub online_visible: Option<bool>,
    pub last_online_visible: Option<bool>,
    pub friends_visible: Option<String>,
    pub following_visible: Option<String>,
    pub followers_visible: Option<String>,
    pub moments_visible: Option<String>,
    pub stats_visible: Option<String>,
    
    // 社交设置
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<String>,
    
    // 搜索设置
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

    /// 获取用户设置
    pub async fn get(&self, user_id: i64) -> Result<UserSettings> {
        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT 
                user_id,
                privacy_level::TEXT as "privacy_level!",
                location_visible,
                online_visible,
                last_online_visible,
                friends_visible::TEXT as "friends_visible!",
                following_visible::TEXT as "following_visible!",
                followers_visible::TEXT as "followers_visible!",
                moments_visible::TEXT as "moments_visible!",
                stats_visible::TEXT as "stats_visible!",
                allow_follow,
                allow_private_chat::TEXT as "allow_private_chat!",
                searchable,
                phone_searchable,
                nickname_searchable,
                location_searchable,
                created_at,
                updated_at
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

    /// 按用户ID查找设置（返回 Option）
    pub async fn find_by_user_id(&self, user_id: i64) -> Result<Option<UserSettings>> {
        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT 
                user_id,
                privacy_level::TEXT as "privacy_level!",
                location_visible,
                online_visible,
                last_online_visible,
                friends_visible::TEXT as "friends_visible!",
                following_visible::TEXT as "following_visible!",
                followers_visible::TEXT as "followers_visible!",
                moments_visible::TEXT as "moments_visible!",
                stats_visible::TEXT as "stats_visible!",
                allow_follow,
                allow_private_chat::TEXT as "allow_private_chat!",
                searchable,
                phone_searchable,
                nickname_searchable,
                location_searchable,
                created_at,
                updated_at
            FROM user_settings
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(settings)
    }

    /// 更新用户设置（部分更新）
    pub async fn update(&self, user_id: i64, params: &UpdateSettingsParams) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_settings
            SET 
                privacy_level = COALESCE($2::TEXT, privacy_level::TEXT)::privacy_level,
                location_visible = COALESCE($3, location_visible),
                online_visible = COALESCE($4, online_visible),
                last_online_visible = COALESCE($5, last_online_visible),
                friends_visible = COALESCE($6::TEXT, friends_visible::TEXT)::privacy_level,
                following_visible = COALESCE($7::TEXT, following_visible::TEXT)::privacy_level,
                followers_visible = COALESCE($8::TEXT, followers_visible::TEXT)::privacy_level,
                moments_visible = COALESCE($9::TEXT, moments_visible::TEXT)::privacy_level,
                stats_visible = COALESCE($10::TEXT, stats_visible::TEXT)::privacy_level,
                allow_follow = COALESCE($11, allow_follow),
                allow_private_chat = COALESCE($12::TEXT, allow_private_chat::TEXT)::chat_permission,
                searchable = COALESCE($13, searchable),
                phone_searchable = COALESCE($14, phone_searchable),
                nickname_searchable = COALESCE($15, nickname_searchable),
                location_searchable = COALESCE($16, location_searchable),
                updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id,
            params.privacy_level.as_deref(),
            params.location_visible,
            params.online_visible,
            params.last_online_visible,
            params.friends_visible.as_deref(),
            params.following_visible.as_deref(),
            params.followers_visible.as_deref(),
            params.moments_visible.as_deref(),
            params.stats_visible.as_deref(),
            params.allow_follow,
            params.allow_private_chat.as_deref(),
            params.searchable,
            params.phone_searchable,
            params.nickname_searchable,
            params.location_searchable
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 创建用户设置（使用数据库默认值）
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
            SELECT 
                user_id,
                privacy_level::TEXT as "privacy_level!",
                location_visible,
                online_visible,
                last_online_visible,
                friends_visible::TEXT as "friends_visible!",
                following_visible::TEXT as "following_visible!",
                followers_visible::TEXT as "followers_visible!",
                moments_visible::TEXT as "moments_visible!",
                stats_visible::TEXT as "stats_visible!",
                allow_follow,
                allow_private_chat::TEXT as "allow_private_chat!",
                searchable,
                phone_searchable,
                nickname_searchable,
                location_searchable,
                created_at,
                updated_at
            FROM user_settings
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(settings)
    }
    
    /// 检查用户是否允许私聊
    pub async fn can_receive_private_chat(&self, user_id: i64, _from_user_id: i64, is_friend: bool) -> Result<bool> {
        let settings = self.find_by_user_id(user_id).await?;
        
        match settings {
            Some(s) => {
                match s.allow_private_chat.as_str() {
                    "everyone" => Ok(true),
                    "friends" => Ok(is_friend),
                    "nobody" => Ok(false),
                    _ => Ok(true), // 默认允许
                }
            }
            None => Ok(true), // 没有设置默认允许
        }
    }
    
    /// 检查用户是否允许被关注
    pub async fn can_be_followed(&self, user_id: i64) -> Result<bool> {
        let settings = self.find_by_user_id(user_id).await?;
        Ok(settings.map(|s| s.allow_follow).unwrap_or(true))
    }
}
