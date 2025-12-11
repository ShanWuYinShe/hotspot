use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::UserType;

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
    pub user_type: UserType,
    pub current_latitude: Option<f64>,
    pub current_longitude: Option<f64>,
    pub current_location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 用户搜索结果Row - 包含距离和方位角（统一搜索结果结构）
#[derive(Debug, Clone)]
pub struct UserSearchRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub user_type: UserType,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Vec<String>,
    /// PostGIS 计算的距离（米）
    pub distance: f64,
    /// PostGIS 计算的方位角（0-360度）
    pub degrees: f64,
}

#[derive(Debug, Clone)]
pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取数据库连接池引用
    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    /// 创建用户
    pub async fn create(
        &self,
        user_id: i64,
        user_type: UserType,
        login_id: Option<String>,
        nickname: String,
        password_hash: Option<String>,
    ) -> Result<UserRow> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            INSERT INTO users (user_id, user_type, login_id, nickname, password_hash)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                      email, phone, user_type as "user_type: UserType",
                      current_latitude, current_longitude, current_location_name,
                      created_at, is_deleted, deleted_at
            "#,
            user_id,
            user_type as UserType,
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
                   email, phone, user_type as "user_type: UserType",
                   current_latitude, current_longitude, current_location_name,
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
                   email, phone, user_type as "user_type: UserType",
                   current_latitude, current_longitude, current_location_name,
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

    /// 按ID列表查找用户（带距离计算，使用 PostGIS）
    /// 按 ID 列表查找用户（带距离计算）
    pub async fn find_by_ids_with_distance(
        &self,
        user_ids: &[i64],
        latitude: f64,
        longitude: f64,
    ) -> Result<Vec<UserSearchRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query!(
            r#"
            SELECT 
                user_id,
                nickname,
                avatar_url,
                bio,
                user_type as "user_type!: UserType",
                COALESCE(current_latitude, 35.154922) as "latitude!",
                COALESCE(current_longitude, 112.682524) as "longitude!",
                COALESCE(current_location_name, '南天门') as "location_name!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
                    ST_SetSRID(ST_MakePoint(COALESCE(current_longitude, 112.682524), COALESCE(current_latitude, 35.154922)), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    ST_SetSRID(ST_MakePoint(COALESCE(current_longitude, 112.682524), COALESCE(current_latitude, 35.154922)), 4326)
                )), 0) as "degrees!"
            FROM users
            WHERE user_id = ANY($1) AND is_deleted = FALSE
            ORDER BY "distance!" ASC
            "#,
            user_ids,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| UserSearchRow {
            user_id: r.user_id,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            bio: r.bio,
            user_type: r.user_type,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            tags: vec![], // TODO: 从 user_tags 获取
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 按登录ID查找用户
    pub async fn find_by_login_id(&self, login_id: &str) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, cursor_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, user_type as "user_type: UserType",
                   current_latitude, current_longitude, current_location_name,
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

    /// 更新用户当前位置
    pub async fn update_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET current_latitude = $2,
                current_longitude = $3,
                current_location_name = COALESCE($4, current_location_name)
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

    /// 按昵称搜索用户（带距离计算）
    pub async fn search_by_keyword(
        &self,
        keyword: &str,
        latitude: f64,
        longitude: f64,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<UserSearchRow>> {
        let search_pattern = format!("%{}%", keyword);
        let cursor_pk = self.resolve_cursor(cursor).await?;

        let rows = sqlx::query!(
            r#"
            SELECT 
                u.user_id,
                u.nickname,
                u.avatar_url,
                u.bio,
                u.user_type as "user_type!: UserType",
                COALESCE(u.current_latitude, 35.154922) as "latitude!",
                COALESCE(u.current_longitude, 112.682524) as "longitude!",
                COALESCE(u.current_location_name, '南天门') as "location_name!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
                    ST_SetSRID(ST_MakePoint(COALESCE(u.current_longitude, 112.682524), COALESCE(u.current_latitude, 35.154922)), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    ST_SetSRID(ST_MakePoint(COALESCE(u.current_longitude, 112.682524), COALESCE(u.current_latitude, 35.154922)), 4326)
                )), 0) as "degrees!"
            FROM users u
            WHERE u.nickname ILIKE $1 AND u.is_deleted = FALSE AND u.user_id < $4
            ORDER BY u.user_id DESC
            LIMIT $5
            "#,
            search_pattern,
            longitude,
            latitude,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| UserSearchRow {
            user_id: r.user_id,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            bio: r.bio,
            user_type: r.user_type,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            tags: vec![], // TODO: 从 user_tags 获取
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 按位置搜索附近用户（PostGIS 计算距离）
    pub async fn search_by_location(
        &self,
        latitude: f64,
        longitude: f64,
        radius: f64,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<UserSearchRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query!(
            r#"
            SELECT 
                user_id, 
                nickname, 
                avatar_url, 
                bio,
                user_type as "user_type!: UserType",
                current_latitude as "latitude!",
                current_longitude as "longitude!",
                COALESCE(current_location_name, '南天门') as "location_name!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                    ST_SetSRID(ST_MakePoint(current_longitude, current_latitude), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    ST_SetSRID(ST_MakePoint(current_longitude, current_latitude), 4326)
                )), 0) as "degrees!"
            FROM users
            WHERE is_deleted = FALSE
              AND user_id < $4
              AND current_latitude IS NOT NULL
              AND current_longitude IS NOT NULL
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  ST_SetSRID(ST_MakePoint(current_longitude, current_latitude), 4326)::geography,
                  $3
              )
            ORDER BY "distance!" ASC, user_id DESC
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

        Ok(rows.into_iter().map(|r| UserSearchRow {
            user_id: r.user_id,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            bio: r.bio,
            user_type: r.user_type,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            tags: vec![], // TODO: 从 user_tags 获取
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
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

    /// 获取用户当前位置
    pub async fn get_location(&self, user_id: i64) -> Result<Option<(f64, f64)>> {
        let row = sqlx::query!(
            r#"
            SELECT current_latitude, current_longitude
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.and_then(|r| {
            match (r.current_latitude, r.current_longitude) {
                (Some(lat), Some(lng)) => Some((lat, lng)),
                _ => None,
            }
        }))
    }

    /// 按位置搜索用户（用于 Feed，使用 offset 分页，可排除特定用户）
    pub async fn search_for_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        radius: f64,
        limit: i64,
        offset: i64,
        exclude_user_id: Option<i64>,
    ) -> Result<Vec<UserSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                u.user_id,
                u.nickname,
                u.avatar_url,
                u.bio,
                u.current_latitude as "latitude!",
                u.current_longitude as "longitude!",
                COALESCE(u.current_location_name, '南天门') as "location_name!",
                u.user_type as "user_type!: UserType",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint(u.current_longitude, u.current_latitude), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    ST_SetSRID(ST_MakePoint(u.current_longitude, u.current_latitude), 4326)
                )), 0) as "degrees!"
            FROM users u
            LEFT JOIN user_settings us ON u.user_id = us.user_id
            WHERE u.is_deleted = FALSE
              AND (us.searchable IS NULL OR us.searchable = TRUE)
              AND (us.location_searchable IS NULL OR us.location_searchable = TRUE)
              AND ($5::BIGINT IS NULL OR u.user_id != $5)
              AND u.current_latitude IS NOT NULL
              AND u.current_longitude IS NOT NULL
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint(u.current_longitude, u.current_latitude), 4326)::geography,
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  $3
              )
            ORDER BY "distance!" ASC
            LIMIT $4 OFFSET $6
            "#,
            user_lng,
            user_lat,
            radius,
            limit,
            exclude_user_id,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| UserSearchRow {
            user_id: r.user_id,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            bio: r.bio,
            user_type: r.user_type,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            tags: vec![], // TODO: 从 user_tags 获取
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }
}
