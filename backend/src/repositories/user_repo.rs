use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::user::User,
};

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
                   latitude, longitude, location_name, created_at, last_active_at
            FROM users
            WHERE login_id = $1
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
                   latitude, longitude, location_name, created_at, last_active_at
            FROM users
            WHERE user_id = $1
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

    /// 更新用户位置
    pub async fn update_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users 
            SET latitude = $2, longitude = $3, location_name = $4
            WHERE user_id = $1
            "#,
            user_id,
            latitude,
            longitude,
            location_name
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按位置搜索用户
    pub async fn search_by_location(
        &self,
        latitude: f64,
        longitude: f64,
        radius: i32,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<User>, Option<String>)> {
        let users = sqlx::query_as!(
            User,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   latitude, longitude, location_name, created_at, last_active_at
            FROM users
            WHERE location IS NOT NULL
              AND ST_DWithin(
                  location,
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  $3
              )
              AND ($4::BIGINT IS NULL OR user_id > $4::BIGINT)
            ORDER BY ST_Distance(
                location,
                ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
            ), user_id
            LIMIT $5
            "#,
            longitude,
            latitude,
            radius as f64,
            cursor,
            limit as i64,
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if users.len() == limit as usize {
            users.last().map(|u| u.user_id.to_string())
        } else {
            None
        };

        Ok((users, next_cursor))
    }

    /// 按昵称搜索用户
    pub async fn search_by_name(
        &self,
        name: &str,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<User>, Option<String>)> {
        let search_pattern = format!("%{}%", name);
        
        let users = sqlx::query_as!(
            User,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   latitude, longitude, location_name, created_at, last_active_at
            FROM users
            WHERE nickname ILIKE $1
              AND ($2::BIGINT IS NULL OR user_id > $2::BIGINT)
            ORDER BY user_id
            LIMIT $3
            "#,
            search_pattern,
            cursor,
            limit as i64,
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if users.len() == limit as usize {
            users.last().map(|u| u.user_id.to_string())
        } else {
            None
        };

        Ok((users, next_cursor))
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
}

