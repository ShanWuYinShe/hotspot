//! 签到相关数据仓库实现

use chrono::{DateTime, Utc};
use sqlx::PgPool;
use tracing::error;
use uuid::Uuid;
use serde::{Deserialize, Serialize};

use crate::{
    repositories::{RepositoryError, RepositoryResult},
    utils::constants::{DEFAULT_LATITUDE, DEFAULT_LOCATION_NAME, DEFAULT_LONGITUDE},
};

// =============== 查询结果结构体定义 ===============

/// 签到查询结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckinQuery {
    pub id: i64,
    pub checkin_id: String,
    pub user_id: i64,
    pub nickname: String,
    pub description: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: f64,
    pub created_at: DateTime<Utc>,
    pub tags: Vec<String>,
    pub likes_count: i64,
    pub comments_count: i64,
    pub liked_by_me: bool,
}

/// PostgreSQL实现的签到仓库
#[derive(Debug)]
pub struct PgCheckinRepository {
    pool: PgPool,
}

impl PgCheckinRepository {
    /// 创建新的PostgreSQL签到仓库
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 检查签到是否存在
    #[allow(dead_code)]
    pub fn check_checkin_exists(
        &self,
        checkin_id: Uuid,
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send {
        let pool = self.pool.clone();
        async move {
            let exists = sqlx::query_scalar!(
                r#"SELECT EXISTS(SELECT 1 FROM checkins WHERE checkin_id = $1) as "exists!""#,
                checkin_id
            )
            .fetch_one(&pool)
            .await
            .map_err(|e| {
                error!("查询签到失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            Ok(exists)
        }
    }

    /// 检查签到所有权
    #[allow(dead_code)]
    pub fn check_checkin_ownership(
        &self,
        checkin_id: Uuid,
        user_id: i64,
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send {
        let pool = self.pool.clone();
        async move {
            let exists = sqlx::query_scalar!(
                r#"SELECT EXISTS(SELECT 1 FROM checkins WHERE checkin_id = $1 AND user_id = $2) as "exists!""#,
                checkin_id,
                user_id
            )
            .fetch_one(&pool)
            .await
            .map_err(|e| {
                error!("查询签到所有权失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            Ok(exists)
        }
    }

    /// 检查用户是否已点赞签到
    pub fn check_checkin_liked(
        &self,
        checkin_id: Uuid,
        user_id: i64,
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send {
        let pool = self.pool.clone();
        async move {
            let exists = sqlx::query_scalar!(
                r#"SELECT EXISTS(SELECT 1 FROM checkin_likes WHERE checkin_id = $1 AND user_id = $2) as "exists!""#,
                checkin_id,
                user_id
            )
            .fetch_one(&pool)
            .await
            .map_err(|e| {
                error!("查询签到点赞状态失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            Ok(exists)
        }
    }

    /// 创建签到
    pub async fn create_checkin(
        &self,
        user_id: i64,
        description: &str,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        tags: &[String],
    ) -> RepositoryResult<String> {
        // 创建签到
        let checkin_id = Uuid::new_v4();
        let now = Utc::now();

        // 开启事务
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 插入签到记录
        sqlx::query!(
            r#"
            INSERT INTO checkins (checkin_id, user_id, description, latitude, longitude, location_name, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            "#,
            checkin_id,
            user_id,
            description,
            latitude,
            longitude,
            location_name,
            now
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建签到失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 处理标签
        for tag_name in tags {
            let tag_name = tag_name.trim();
            if tag_name.is_empty() {
                continue;
            }

            // 获取或创建标签
            let tag_id = sqlx::query_scalar!(
                r#"
                WITH inserted_tag AS (
                    INSERT INTO tags (tag_id, name, counter)
                    VALUES (uuid_generate_v4(), $1, 1)
                    ON CONFLICT (name) DO 
                    UPDATE SET counter = tags.counter + 1
                    RETURNING tag_id
                )
                SELECT tag_id FROM inserted_tag
                UNION ALL
                SELECT tag_id FROM tags WHERE name = $1
                LIMIT 1
                "#,
                tag_name
            )
            .fetch_one(&mut *tx)
            .await
            .map_err(|e| {
                error!("处理标签失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            // 关联签到和标签
            sqlx::query!(
                r#"
                INSERT INTO checkin_tags (checkin_id, tag_id)
                VALUES ($1, $2)
                ON CONFLICT (checkin_id, tag_id) DO NOTHING
                "#,
                checkin_id,
                tag_id
            )
            .execute(&mut *tx)
            .await
            .map_err(|e| {
                error!("关联标签失败: {:?}", e);
                RepositoryError::Database(e)
            })?;
        }

        // 提交事务
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(checkin_id.to_string())
    }

    /// 删除签到记录
    pub async fn delete_checkin_record(&self, checkin_id: Uuid) -> RepositoryResult<()> {
        sqlx::query!(r#"DELETE FROM checkins WHERE checkin_id = $1"#, checkin_id)
            .execute(&self.pool)
            .await
            .map_err(|e| {
                error!("删除签到失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

        Ok(())
    }

    /// 添加签到点赞
    #[allow(dead_code)]
    pub async fn add_checkin_like(&self, checkin_id: Uuid, user_id: i64) -> RepositoryResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO checkin_likes (checkin_id, user_id, created_at)
            VALUES ($1, $2, $3)
            "#,
            checkin_id,
            user_id,
            Utc::now()
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("添加签到点赞失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(())
    }

    /// 移除签到点赞
    pub async fn remove_checkin_like(&self, checkin_id: Uuid, user_id: i64) -> RepositoryResult<()> {
        sqlx::query!(
            r#"DELETE FROM checkin_likes WHERE checkin_id = $1 AND user_id = $2"#,
            checkin_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("移除签到点赞失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(())
    }

    /// 获取签到详情
    pub async fn search_checkin_by_id(
        &self,
        user_id: i64,
        checkin_id: &str,
    ) -> RepositoryResult<CheckinQuery> {
        let checkin_uuid = Uuid::parse_str(checkin_id).map_err(|e| {
            error!("解析签到ID失败: {:?}", e);
            RepositoryError::InvalidArgument("签到ID格式错误".to_string())
        })?;

        // 查询签到信息
        let checkin = sqlx::query_as!(
            CheckinQuery,
            r#"
            WITH current_user_loc AS (
                SELECT latitude, longitude 
                FROM user_locations 
                WHERE user_id = $1
            ),
            checkin_tags AS (
                SELECT checkin_id, array_agg(t.name) as tag_names
                FROM checkin_tags ct
                JOIN tags t ON ct.tag_id = t.tag_id
                WHERE ct.checkin_id = $2
                GROUP BY ct.checkin_id
            )
            SELECT 
                c.id as "id!",
                c.checkin_id::text as "checkin_id!",
                c.user_id as "user_id!",
                u.nickname as "nickname!",
                c.description,
                COALESCE(c.latitude, $3) as "latitude!",
                COALESCE(c.longitude, $4) as "longitude!",
                COALESCE(c.location_name, $5) as "location_name!",
                c.created_at as "created_at!",
                ST_Distance(
                    ST_MakePoint(COALESCE(c.longitude, $4), COALESCE(c.latitude, $3))::geography,
                    ST_MakePoint(COALESCE(ul.longitude, $4), COALESCE(ul.latitude, $3))::geography
                ) as "distance!",
                COALESCE(ct.tag_names, ARRAY[]::text[]) as "tags!",
                (SELECT COUNT(*) FROM checkin_likes WHERE checkin_id = c.checkin_id) as "likes_count!",
                0 as "comments_count!", -- 如有评论功能请替换
                (SELECT EXISTS(SELECT 1 FROM checkin_likes WHERE checkin_id = c.checkin_id AND user_id = $1)) as "liked_by_me!"
            FROM checkins c
            JOIN users u ON c.user_id = u.user_id
            LEFT JOIN user_locations ul ON ul.user_id = $1
            LEFT JOIN checkin_tags ct ON ct.checkin_id = c.checkin_id
            WHERE c.checkin_id = $2
            "#,
            user_id,
            checkin_uuid,
            DEFAULT_LATITUDE,
            DEFAULT_LONGITUDE,
            DEFAULT_LOCATION_NAME,
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询签到失败: {:?}", e);
            match e {
                sqlx::Error::RowNotFound => RepositoryError::NotFound,
                _ => RepositoryError::Database(e)
            }
        })?;

        Ok(checkin)
    }

    /// 获取用户签到历史
    pub fn search_checkins_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<CheckinQuery>>> + Send {
        let pool = self.pool.clone();
        async move {
        let rows = sqlx::query_as!(
            CheckinQuery,
            r#"
            WITH current_user_loc AS (
                SELECT latitude, longitude 
                FROM user_locations 
                WHERE user_id = $1
            )
            SELECT 
                c.id as "id!",
                c.checkin_id::text as "checkin_id!",
                c.user_id as "user_id!",
                u.nickname as "nickname!",
                c.description,
                c.latitude as "latitude!",
                c.longitude as "longitude!",
                COALESCE(c.location_name, '') as "location_name!",
                c.created_at as "created_at!",
                ST_Distance(
                    ST_MakePoint(c.longitude, c.latitude)::geography,
                    ST_MakePoint(COALESCE(ul.longitude, $2), COALESCE(ul.latitude, $2))::geography
                ) as "distance!",
                COALESCE(ARRAY(SELECT t.name FROM checkin_tags ct JOIN tags t ON ct.tag_id = t.tag_id WHERE ct.checkin_id = c.checkin_id), ARRAY[]::text[]) as "tags!",
                COALESCE((SELECT COUNT(*) FROM checkin_likes WHERE checkin_id = c.checkin_id), 0) as "likes_count!",
                0 as "comments_count!",
                COALESCE((SELECT EXISTS(SELECT 1 FROM checkin_likes WHERE checkin_id = c.checkin_id AND user_id = $1)), false) as "liked_by_me!"
            FROM checkins c
            JOIN users u ON c.user_id = u.user_id
            LEFT JOIN user_locations ul ON ul.user_id = $1
            WHERE c.user_id = $1
            AND ($3::bigint = 0 OR c.id < $3::bigint)
            ORDER BY c.id DESC
            LIMIT $4
            "#,
            user_id,
            DEFAULT_LONGITUDE,
            cursor,
            limit + 1
        )
        .fetch_all(&pool)
        .await
        .map_err(|e| {
            error!("查询用户签到历史失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(rows)
        }
    }

    /// 搜索附近签到
    pub fn search_checkins_by_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<CheckinQuery>>> + Send {
        let pool = self.pool.clone();
        async move {
        let rows = sqlx::query_as!(
            CheckinQuery,
            r#"
            SELECT 
                c.id as "id!",
                c.checkin_id::text as "checkin_id!",
                c.user_id as "user_id!",
                u.nickname as "nickname!",
                c.description,
                c.latitude as "latitude!",
                c.longitude as "longitude!",
                c.location_name as "location_name!",
                c.created_at as "created_at!",
                ST_Distance(
                    ST_MakePoint(c.longitude, c.latitude)::geography,
                    ST_MakePoint($3, $2)::geography
                ) as "distance!",
                ARRAY(SELECT t.name FROM checkin_tags ct JOIN tags t ON ct.tag_id = t.tag_id WHERE ct.checkin_id = c.checkin_id) as "tags!",
                (SELECT COUNT(*) FROM checkin_likes WHERE checkin_id = c.checkin_id) as "likes_count!",
                0 as "comments_count!",
                (SELECT EXISTS(SELECT 1 FROM checkin_likes WHERE checkin_id = c.checkin_id AND user_id = $1)) as "liked_by_me!"
            FROM checkins c
            JOIN users u ON c.user_id = u.user_id
            WHERE ($5::bigint = 0 OR c.id < $5::bigint)
            AND ST_Distance(
                ST_MakePoint(c.longitude, c.latitude)::geography,
                ST_MakePoint($3, $2)::geography
            ) <= $4
            ORDER BY c.id DESC
            LIMIT $6
            "#,
            user_id,
            latitude,
            longitude,
            radius,
            cursor,
            limit + 1
        )
        .fetch_all(&pool)
        .await
        .map_err(|e| {
            error!("查询附近签到失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(rows)
        }
    }

    /// 根据标签搜索签到
    pub fn search_checkins_by_tags(
        &self,
        user_id: i64,
        tags: &[String],
        cursor: i64,
        limit: i64,
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<CheckinQuery>>> + Send {
        let pool = self.pool.clone();
        async move {
        let rows = sqlx::query_as!(
            CheckinQuery,
            r#"
            WITH current_user_loc AS (
                SELECT latitude, longitude 
                FROM user_locations 
                WHERE user_id = $1
            )
            SELECT 
                c.id as "id!",
                c.checkin_id::text as "checkin_id!",
                c.user_id as "user_id!",
                u.nickname as "nickname!",
                c.description,
                c.latitude as "latitude!",
                c.longitude as "longitude!",
                c.location_name as "location_name!",
                c.created_at as "created_at!",
                ST_Distance(
                    ST_MakePoint(c.longitude, c.latitude)::geography,
                    ST_MakePoint(COALESCE(ul.longitude, $2), COALESCE(ul.latitude, $2))::geography
                ) as "distance!",
                ARRAY(SELECT t.name FROM checkin_tags ct JOIN tags t ON ct.tag_id = t.tag_id WHERE ct.checkin_id = c.checkin_id) as "tags!",
                (SELECT COUNT(*) FROM checkin_likes WHERE checkin_id = c.checkin_id) as "likes_count!",
                0 as "comments_count!",
                (SELECT EXISTS(SELECT 1 FROM checkin_likes WHERE checkin_id = c.checkin_id AND user_id = $1)) as "liked_by_me!"
            FROM checkins c
            JOIN users u ON c.user_id = u.user_id
            LEFT JOIN user_locations ul ON ul.user_id = $1
            JOIN checkin_tags ct ON c.checkin_id = ct.checkin_id
            JOIN tags t ON ct.tag_id = t.tag_id
            WHERE t.name = ANY($3)
            AND ($4::bigint = 0 OR c.id < $4::bigint)
            ORDER BY c.id DESC
            LIMIT $5
            "#,
            user_id,
            DEFAULT_LONGITUDE,
            tags,
            cursor,
            limit + 1
        )
        .fetch_all(&pool)
        .await
        .map_err(|e| {
            error!("搜索签到失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(rows)
        }
    }
}
