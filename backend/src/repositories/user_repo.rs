use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::user::User,
};

#[derive(Clone)]
pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建用户
    pub async fn create(&self, user: &User) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO users (user_id, login_id, nickname, password_hash, avatar_url, bio, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, NOW())
            "#,
            user.user_id,
            user.login_id,
            user.nickname,
            user.password_hash,
            user.avatar_url,
            user.bio,
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 根据登录ID查找用户
    pub async fn find_by_login_id(&self, login_id: &str) -> Result<Option<User>> {
        let user = sqlx::query_as!(
            User,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   created_at, last_active_at, is_deleted, deleted_at
            FROM users
            WHERE login_id = $1 AND is_deleted = FALSE
            "#,
            login_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(user)
    }

    /// 根据用户ID查找用户
    pub async fn find_by_id(&self, user_id: i64) -> Result<Option<User>> {
        let user = sqlx::query_as!(
            User,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   created_at, last_active_at, is_deleted, deleted_at
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(user)
    }

    /// 检查登录ID是否存在
    pub async fn exists_by_login_id(&self, login_id: &str) -> Result<bool> {
        let result = sqlx::query!(
            "SELECT EXISTS(SELECT 1 FROM users WHERE login_id = $1) as \"exists!\"",
            login_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 更新最后活跃时间
    pub async fn update_last_active(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET last_active_at = NOW() WHERE user_id = $1",
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按昵称搜索用户（双ID游标分页）
    pub async fn search_by_name(
        &self,
        name: &str,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<User>, Option<String>)> {
        let search_pattern = format!("%{}%", name);
        
        // 1. 根据业务ID(user_id)查询对应的自增主键id (如果提供了cursor)
        let cursor_id = if let Some(user_id_str) = cursor {
            if let Ok(user_id) = user_id_str.parse::<i64>() {
                sqlx::query!("SELECT id FROM users WHERE user_id = $1", user_id)
                    .fetch_optional(&self.pool)
                    .await?
                    .map(|r| r.id)
            } else {
                None
            }
        } else {
            None
        };

        // 2. 多查询一条以判断has_more
        let users = sqlx::query_as!(
            User,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   created_at, last_active_at, is_deleted, deleted_at
            FROM users
            WHERE nickname ILIKE $1
              AND is_deleted = FALSE
              AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id ASC
            LIMIT $3
            "#,
            search_pattern,
            cursor_id,
            (limit + 1) as i64,
        )
        .fetch_all(&self.pool)
        .await?;

        // 3. 判断has_more并构造next_cursor
        let has_more = users.len() > limit as usize;
        let result_users = if has_more {
            users[..limit as usize].to_vec()
        } else {
            users
        };
        
        let next_cursor = if has_more {
            result_users.last().map(|u| u.user_id.to_string())
        } else {
            None
        };

        Ok((result_users, next_cursor))
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
    pub async fn update_password(&self, user_id: i64, password_hash: &str) -> Result<()> {
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
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 按位置搜索用户（PostGIS + 双ID游标分页）
    pub async fn search_by_location(
        &self,
        latitude: f64,
        longitude: f64,
        radius: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<(User, f64)>, Option<String>)> {
        // 1. 根据业务ID(user_id)查询对应的自增主键id (如果提供了cursor)
        let cursor_id = if let Some(user_id_str) = cursor {
            if let Ok(user_id) = user_id_str.parse::<i64>() {
                sqlx::query!("SELECT id FROM users WHERE user_id = $1", user_id)
                    .fetch_optional(&self.pool)
                    .await?
                    .map(|r| r.id)
            } else {
                None
            }
        } else {
            None
        };

        // 2. 使用PostGIS查询附近用户（多查询一条以判断has_more）
        let results = sqlx::query!(
            r#"
            SELECT 
                u.user_id, u.login_id, u.nickname, u.password_hash, u.avatar_url, u.bio,
                u.created_at, u.last_active_at, u.is_deleted, u.deleted_at,
                ST_Distance(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                    ST_SetSRID(ST_MakePoint(ulh.longitude, ulh.latitude), 4326)::geography
                ) as "distance!"
            FROM users u
            INNER JOIN (
                SELECT DISTINCT ON (user_id) user_id, latitude, longitude
                FROM user_location_history
                ORDER BY user_id, created_at DESC
            ) ulh ON u.user_id = ulh.user_id
            WHERE u.is_deleted = FALSE
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  ST_SetSRID(ST_MakePoint(ulh.longitude, ulh.latitude), 4326)::geography,
                  $3
              )
              AND ($4::BIGINT IS NULL OR u.id > $4::BIGINT)
            ORDER BY u.id ASC
            LIMIT $5
            "#,
            longitude,
            latitude,
            radius,
            cursor_id,
            (limit + 1) as i64,
        )
        .fetch_all(&self.pool)
        .await?;

        // 3. 构造结果
        let has_more = results.len() > limit as usize;
        let result_items: Vec<(User, f64)> = results
            .iter()
            .take(if has_more { limit as usize } else { results.len() })
            .map(|r| {
                (
                    User {
                        user_id: r.user_id,
                        login_id: r.login_id.clone(),
                        nickname: r.nickname.clone(),
                        password_hash: r.password_hash.clone(),
                        avatar_url: r.avatar_url.clone(),
                        bio: r.bio.clone(),
                        created_at: r.created_at,
                        last_active_at: r.last_active_at,
                        is_deleted: r.is_deleted,
                        deleted_at: r.deleted_at,
                    },
                    r.distance,
                )
            })
            .collect();

        let next_cursor = if has_more {
            result_items.last().map(|(u, _)| u.user_id.to_string())
        } else {
            None
        };

        Ok((result_items, next_cursor))
    }

    /// 获取用户统计数据
    pub async fn get_user_stats(&self, user_id: i64) -> Result<(i32, i32, i32, i32, i32)> {
        let moments_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM moments
            WHERE user_id = $1 AND deleted_at IS NULL
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let friends_count = sqlx::query!(
            r#"
            SELECT COUNT(DISTINCT 
                CASE 
                    WHEN from_user_id = $1 THEN to_user_id
                    ELSE from_user_id
                END
            ) as "count!"
            FROM user_relationships
            WHERE (from_user_id = $1 OR to_user_id = $1)
              AND relationship_type = 'friend'
              AND status = 'accepted'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let followers_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_relationships
            WHERE to_user_id = $1
              AND relationship_type = 'following'
              AND status = 'accepted'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let following_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_relationships
            WHERE from_user_id = $1
              AND relationship_type = 'following'
              AND status = 'accepted'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let groups_count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM room_members
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        Ok((moments_count, friends_count, followers_count, following_count, groups_count))
    }

    /// 获取用户未读统计
    pub async fn get_unread_stats(&self, user_id: i64) -> Result<(i32, i32)> {
        let friend_requests = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_relationships
            WHERE to_user_id = $1
              AND relationship_type = 'friend'
              AND status = 'pending'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        let group_invites = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM room_invites
            WHERE invitee_id = $1 AND status = 'pending'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;

        Ok((friend_requests, group_invites))
    }

    /// 获取用户设置
    pub async fn get_settings(&self, user_id: i64) -> Result<(String, bool, bool)> {
        let result = sqlx::query!(
            r#"
            SELECT privacy_level, location_sharing, allow_follow
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok((
            result.privacy_level,
            result.location_sharing,
            result.allow_follow,
        ))
    }

    /// 更新用户设置
    pub async fn update_settings(
        &self,
        user_id: i64,
        privacy_level: Option<&str>,
        location_sharing: Option<bool>,
        allow_follow: Option<bool>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET privacy_level = COALESCE($2, privacy_level),
                location_sharing = COALESCE($3, location_sharing),
                allow_follow = COALESCE($4, allow_follow)
            WHERE user_id = $1
            "#,
            user_id,
            privacy_level,
            location_sharing,
            allow_follow
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}

