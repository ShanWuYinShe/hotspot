use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 用户Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserRow {
    pub user_id: i64,
    pub cursor_id: Uuid,
    pub login_id: Option<String>,
    pub nickname: String,
    pub password_hash: Option<String>,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub current_latitude: Option<f64>,
    pub current_longitude: Option<f64>,
    pub current_location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建用户
    pub async fn create(
        &self,
        user_id: i64,
        login_id: Option<String>,
        nickname: String,
        password_hash: Option<String>,
    ) -> Result<UserRow> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            INSERT INTO users (user_id, login_id, nickname, password_hash)
            VALUES ($1, $2, $3, $4)
            RETURNING user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                      email, phone, current_latitude, current_longitude, current_location_name,
                      created_at, is_deleted, deleted_at
            "#,
            user_id,
            login_id,
            nickname,
            password_hash
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找用户
    pub async fn find_by_id(&self, user_id: i64) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID批量查找用户
    pub async fn find_by_ids(&self, user_ids: &[i64]) -> Result<Vec<UserRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE user_id = ANY($1) AND is_deleted = FALSE
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 按登录ID查找用户
    pub async fn find_by_login_id(&self, login_id: &str) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE login_id = $1 AND is_deleted = FALSE
            "#,
            login_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查登录ID是否存在
    pub async fn exists_by_login_id(&self, login_id: &str) -> Result<bool> {
        let exists = sqlx::query!(
            r#"
            SELECT EXISTS(SELECT 1 FROM users WHERE login_id = $1 AND is_deleted = FALSE) AS "exists!"
            "#,
            login_id
        )
        .fetch_one(&self.pool)
        .await?
        .exists;

        Ok(exists)
    }

    /// 更新用户资料
    pub async fn update_profile(
        &self,
        user_id: i64,
        nickname: Option<&str>,
        avatar_url: Option<&str>,
        bio: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET nickname = COALESCE($2, nickname),
                avatar_url = COALESCE($3, avatar_url),
                bio = COALESCE($4, bio)
            WHERE user_id = $1
            "#,
            user_id,
            nickname,
            avatar_url,
            bio
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新用户密码
    pub async fn update_password(&self, user_id: i64, password_hash: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET password_hash = $2
            WHERE user_id = $1
            "#,
            user_id,
            password_hash
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除用户
    pub async fn soft_delete(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 解析 UUID cursor 为自增主键 (user_id)
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT user_id FROM users WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按昵称搜索用户
    pub async fn search_by_name(&self, name: &str, limit: i64, cursor: Option<Uuid>) -> Result<(Vec<UserRow>, Option<String>)> {
        let search_pattern = format!("%{}%", name);
        let cursor_pk = self.resolve_cursor(cursor).await?;

        let users = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE nickname ILIKE $1 AND is_deleted = FALSE AND user_id < $2
            ORDER BY user_id DESC
            LIMIT $3
            "#,
            search_pattern,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if users.len() == limit as usize {
            users.last().map(|u| u.cursor_id.to_string())
        } else {
            None
        };

        Ok((users, next_cursor))
    }

    /// 按位置搜索附近用户（使用 cursor 分页）
    pub async fn search_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius: f64,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<UserRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let users = sqlx::query_as!(
            UserRow,
            r#"
            SELECT DISTINCT u.user_id, u.cursor_id, u.login_id, u.nickname, u.password_hash, u.avatar_url, u.bio,
                   u.email, u.phone, u.current_latitude, u.current_longitude, u.current_location_name,
                   u.created_at, u.is_deleted, u.deleted_at
            FROM users u
            INNER JOIN (
                SELECT DISTINCT ON (user_id) user_id, latitude, longitude
                FROM user_location_history
                ORDER BY user_id, recorded_at DESC
            ) ulh ON u.user_id = ulh.user_id
            WHERE u.is_deleted = FALSE
              AND u.user_id < $4
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  ST_SetSRID(ST_MakePoint(ulh.longitude, ulh.latitude), 4326)::geography,
                  $3
              )
            ORDER BY u.user_id DESC
            LIMIT $5
            "#,
            longitude,
            latitude,
            radius,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(users)
    }

    /// 获取用户统计数据
    pub async fn get_user_stats(&self, user_id: i64) -> Result<(i32, i32, i32, i32, i32)> {
        let moments_count = sqlx::query!(
            r#"SELECT COUNT(*) as "count!" FROM moments WHERE user_id = $1 AND is_deleted = FALSE"#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let friends_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_friend_requests
            WHERE (from_user_id = $1 OR to_user_id = $1) 
              AND status = 'accepted'::invitation_status
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let followers_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!" 
            FROM user_follows 
            WHERE following_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let following_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!" 
            FROM user_follows 
            WHERE follower_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let groups_count = sqlx::query!(
            r#"SELECT COUNT(*) as "count!" FROM group_members WHERE user_id = $1 AND is_deleted = FALSE"#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        Ok((
            moments_count,
            friends_count,
            followers_count,
            following_count,
            groups_count,
        ))
    }

    /// 获取用户未读统计 (好友请求、群组邀请)
    pub async fn get_unread_stats(&self, user_id: i64) -> Result<(i32, i32)> {
        let friend_requests = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_friend_requests
            WHERE to_user_id = $1 
              AND status = 'pending'::invitation_status
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        // 假设群组邀请在另一个表，或者 inbox 中，或者 group_members status=pending
        // 这里假设 group_members 没有 pending 状态 (邀请是单独的 logical entity)
        // 如果 group_invites 是通过 Inbox 记录的，这里可能只需要查询 Inbox 中未读的 invite 类型
        // 但 UserService 似乎想从 user_repo 获取。
        // 我们暂时返回 0 或实现简单查询
        
        // 假设有一个 group_invitations 表? 不存在。
        // 假设通过 inbox items?
        // 为了修复编译，暂时返回 0 或查询 inbox
        // sqlx query on inbox_items:
        let group_invites = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM inbox_items
            WHERE user_id = $1 
              AND type = 'group_invite' 
              AND is_read = FALSE
              AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        Ok((friend_requests, group_invites))
    }

    /// 获取用户设置
    pub async fn get_settings(&self, user_id: i64) -> Result<(String, bool, bool, bool)> {
        let result = sqlx::query!(
            r#"
            SELECT privacy_level::text as "privacy_level!", is_location_sharing, can_be_followed, allow_private_chat
            FROM user_settings
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        if let Some(s) = result {
            Ok((
                s.privacy_level,
                s.is_location_sharing,
                s.can_be_followed,
                s.allow_private_chat,
            ))
        } else {
            // 默认值：公开、不分享位置、允许关注、允许私聊
            Ok(("public".to_string(), false, true, true))
        }
    }

    /// 更新用户设置
    pub async fn update_settings(
        &self,
        user_id: i64,
        privacy_level: Option<&str>,
        location_sharing: Option<bool>,
        allow_follow: Option<bool>,
    ) -> Result<()> {
        // 使用 COALESCE 在 SQL 中处理可选参数
        sqlx::query(
            r#"
            INSERT INTO user_settings (user_id, privacy_level, is_location_sharing, can_be_followed)
            VALUES ($1, COALESCE($2, 'public')::privacy_level, COALESCE($3, FALSE), COALESCE($4, FALSE))
            ON CONFLICT (user_id) DO UPDATE
            SET privacy_level = COALESCE($2, user_settings.privacy_level::text)::privacy_level,
                is_location_sharing = COALESCE($3, user_settings.is_location_sharing),
                can_be_followed = COALESCE($4, user_settings.can_be_followed),
                updated_at = NOW()
            "#
        )
        .bind(user_id)
        .bind(privacy_level)
        .bind(location_sharing)
        .bind(allow_follow)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 统计用户发布的生活瞬间数量
    pub async fn count_moments(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM moments
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 统计用户加入的群组数量
    pub async fn count_groups(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_members
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 更新用户最后活跃时间
    pub async fn update_last_active(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET last_active_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
