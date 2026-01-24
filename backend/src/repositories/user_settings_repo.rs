//! 用户设置仓库
//!
//! 管理用户的隐私设置、社交设置、搜索设置等
//! 与 API 文档 UserSettingsData 完全对应

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::{PrivacyLevel, PrivateChatPermission};

/// 用户设置Row（与数据库表完全对应）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserSettings {
    pub user_id: i64,

    // ============ 隐私设置 ============
    pub privacy_level: PrivacyLevel, // 'public', 'friends', 'private'
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub following_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub followers_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub groups_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub moments_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub liked_moments_visible: PrivacyLevel, // 'public', 'friends', 'private'
    pub stats_visible: PrivacyLevel,   // 'public', 'friends', 'private'

    // ============ 社交设置 ============
    pub allow_follow: bool,
    pub allow_private_chat: PrivateChatPermission, // 'everyone', 'friends', 'nobody'

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
    pub privacy_level: Option<PrivacyLevel>,
    pub location_visible: Option<bool>,
    pub online_visible: Option<bool>,
    pub last_online_visible: Option<bool>,
    pub friends_visible: Option<PrivacyLevel>,
    pub following_visible: Option<PrivacyLevel>,
    pub followers_visible: Option<PrivacyLevel>,
    pub groups_visible: Option<PrivacyLevel>,
    pub moments_visible: Option<PrivacyLevel>,
    pub liked_moments_visible: Option<PrivacyLevel>,
    pub stats_visible: Option<PrivacyLevel>,

    // 社交设置
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<PrivateChatPermission>,

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

    /// 按用户ID查找设置（返回 Option）
    pub async fn find_by_user_id(&self, user_id: i64) -> Result<Option<UserSettings>> {
        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT 
                user_id,
                privacy_level as "privacy_level: PrivacyLevel",
                location_visible,
                online_visible,
                last_online_visible,
                friends_visible as "friends_visible: PrivacyLevel",
                following_visible as "following_visible: PrivacyLevel",
                followers_visible as "followers_visible: PrivacyLevel",
                groups_visible as "groups_visible: PrivacyLevel",
                moments_visible as "moments_visible: PrivacyLevel",
                liked_moments_visible as "liked_moments_visible: PrivacyLevel",
                stats_visible as "stats_visible: PrivacyLevel",
                allow_follow,
                allow_private_chat as "allow_private_chat: PrivateChatPermission",
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
                privacy_level = COALESCE($2, privacy_level),
                location_visible = COALESCE($3, location_visible),
                online_visible = COALESCE($4, online_visible),
                last_online_visible = COALESCE($5, last_online_visible),
                friends_visible = COALESCE($6, friends_visible),
                following_visible = COALESCE($7, following_visible),
                followers_visible = COALESCE($8, followers_visible),
                groups_visible = COALESCE($9, groups_visible),
                moments_visible = COALESCE($10, moments_visible),
                liked_moments_visible = COALESCE($11, liked_moments_visible),
                stats_visible = COALESCE($12, stats_visible),
                allow_follow = COALESCE($13, allow_follow),
                allow_private_chat = COALESCE($14, allow_private_chat),
                searchable = COALESCE($15, searchable),
                phone_searchable = COALESCE($16, phone_searchable),
                nickname_searchable = COALESCE($17, nickname_searchable),
                location_searchable = COALESCE($18, location_searchable),
                updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id,
            params.privacy_level as Option<PrivacyLevel>,
            params.location_visible,
            params.online_visible,
            params.last_online_visible,
            params.friends_visible as Option<PrivacyLevel>,
            params.following_visible as Option<PrivacyLevel>,
            params.followers_visible as Option<PrivacyLevel>,
            params.groups_visible as Option<PrivacyLevel>,
            params.moments_visible as Option<PrivacyLevel>,
            params.liked_moments_visible as Option<PrivacyLevel>,
            params.stats_visible as Option<PrivacyLevel>,
            params.allow_follow,
            params.allow_private_chat as Option<PrivateChatPermission>,
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
                privacy_level as "privacy_level: PrivacyLevel",
                location_visible,
                online_visible,
                last_online_visible,
                friends_visible as "friends_visible: PrivacyLevel",
                following_visible as "following_visible: PrivacyLevel",
                followers_visible as "followers_visible: PrivacyLevel",
                groups_visible as "groups_visible: PrivacyLevel",
                moments_visible as "moments_visible: PrivacyLevel",
                liked_moments_visible as "liked_moments_visible: PrivacyLevel",
                stats_visible as "stats_visible: PrivacyLevel",
                allow_follow,
                allow_private_chat as "allow_private_chat: PrivateChatPermission",
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
}
