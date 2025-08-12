//! 动态数据仓库实现 - 使用 sqlx 宏版本

use sqlx::PgPool;
use uuid::Uuid;
use chrono::{DateTime, Utc};

use crate::repositories::{Repository, RepositoryResult};

/// 创建动态所需的数据
#[derive(Clone)]
pub struct NewMomentData {
    pub user_id: i64,
    pub content: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Vec<String>,
}

/// 动态查询结果行
#[derive(Clone)]
pub struct MomentRow {
    pub moment_id: String,
    pub user_id: i64,
    pub user_nickname: String,
    #[allow(dead_code)]
    pub user_avatar: Option<String>,
    pub content: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub like_count: i64,
    pub comment_count: i64,
    pub is_liked: bool,
    pub distance: Option<f64>,
    pub tags: Vec<String>,
}

// 已不再使用的 MomentDetailRow 结构已移除（按“无用即删”原则）

/// PostgreSQL 实现的动态仓库
pub struct PgMomentRepository {
    pool: PgPool,
}

/// 位置搜索查询结果结构体
#[derive(Debug)]
struct MomentLocationRow {
    moment_id: Uuid,
    user_id: i64,
    nickname: String,
    content: Option<String>,
    latitude: f64,
    longitude: f64,
    location_name: String,
    distance: f64,
    created_at: DateTime<Utc>,
}

/// 用户动态查询结果结构体
#[derive(Debug)]
struct UserMomentRow {
    id: i64,
    moment_id: Uuid,
    user_id: i64,
    nickname: String,
    content: Option<String>,
    latitude: f64,
    longitude: f64,
    location_name: String,
    created_at: DateTime<Utc>,
}

impl PgMomentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建动态
    pub async fn create_moment(&self, data: NewMomentData) -> RepositoryResult<String> {
        let moment_id = Uuid::new_v4();
        
        // 创建动态
        sqlx::query!(
            r#"
            INSERT INTO moments (moment_id, user_id, content, latitude, longitude, location_name, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, NOW())
            "#,
            moment_id,
            data.user_id,
            data.content,
            data.latitude,
            data.longitude,
            data.location_name
        )
        .execute(&self.pool)
        .await?;
        
        // 创建标签关联
        if !data.tags.is_empty() {
            for tag in &data.tags {
                // 先确保标签存在或创建标签
                let tag_id = sqlx::query_scalar!(
                    "INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET counter = tags.counter + 1 RETURNING tag_id",
                    tag
                )
                .fetch_one(&self.pool)
                .await?;
                
                // 创建动态和标签的关联
                sqlx::query!(
                    "INSERT INTO moment_tags (moment_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
                    moment_id,
                    tag_id
                )
                .execute(&self.pool)
                .await?;
            }
        }
        
        Ok(moment_id.to_string())
    }

    /// 删除动态记录
    pub async fn delete_moment_record(&self, moment_id: &str) -> RepositoryResult<()> {
        let moment_uuid = Uuid::parse_str(moment_id)?;
        
        // 先删除相关的标签
        sqlx::query!(
            "DELETE FROM moment_tags WHERE moment_id = $1",
            moment_uuid
        )
        .execute(&self.pool)
        .await?;
        
        // 删除点赞记录
        sqlx::query!(
            "DELETE FROM moment_likes WHERE moment_id = $1",
            moment_uuid
        )
        .execute(&self.pool)
        .await?;
        
        // 最后删除动态本身
        sqlx::query!(
            "DELETE FROM moments WHERE moment_id = $1",
            moment_uuid
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }

    

    /// 根据动态ID查询动态详情
    pub async fn search_moment_by_id(&self, moment_id: &str, user_id: i64) -> RepositoryResult<Option<MomentRow>> {
        let moment_uuid = Uuid::parse_str(moment_id)?;
        
        // 查询动态基本信息
        #[derive(sqlx::FromRow)]
        struct MomentBasicRow {
            moment_id: Uuid,
            user_id: i64,
            nickname: String,
            content: Option<String>,
            latitude: f64,
            longitude: f64,
            location_name: String,
            created_at: DateTime<Utc>,
        }
        
        let moment_result = sqlx::query_as!(
            MomentBasicRow,
            r#"
            SELECT 
                m.moment_id as "moment_id!",
                m.user_id as "user_id!",
                u.nickname as "nickname!",
                m.content,
                m.latitude as "latitude!",
                m.longitude as "longitude!",
                m.location_name as "location_name!",
                m.created_at as "created_at!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.moment_id = $1
            "#,
            moment_uuid
        )
        .fetch_optional(&self.pool)
        .await?;
        
        if let Some(row) = moment_result {
            // 查询点赞数
            let like_count = sqlx::query_scalar!(
                "SELECT COUNT(*) as count FROM moment_likes WHERE moment_id = $1",
                moment_uuid
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(0);

            // 查询评论数 (暂时设为0，因为没有评论表)
            let comment_count = 0i64;

            // 检查当前用户是否已点赞
            let is_liked = sqlx::query_scalar!(
                "SELECT EXISTS(SELECT 1 FROM moment_likes WHERE moment_id = $1 AND user_id = $2)",
                moment_uuid,
                user_id
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(false);

            // 查询标签
            let tags: Vec<String> = sqlx::query_scalar!(
                "SELECT t.name FROM moment_tags mt JOIN tags t ON mt.tag_id = t.tag_id WHERE mt.moment_id = $1",
                moment_uuid
            )
            .fetch_all(&self.pool)
            .await
            .unwrap_or_default();

            Ok(Some(MomentRow {
                moment_id: row.moment_id.to_string(),
                user_id: row.user_id,
                user_nickname: row.nickname,
                user_avatar: None,
                content: row.content.unwrap_or_default(),
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name,
                tags,
                like_count,
                comment_count,
                is_liked,
                distance: None,
                created_at: row.created_at,
            }))
        } else {
            Ok(None)
        }
    }

    /// 点赞动态
    pub async fn like_moment(&self, moment_id: &str, user_id: i64) -> RepositoryResult<bool> {
        let moment_uuid = Uuid::parse_str(moment_id)?;
        let result = sqlx::query!(
            "INSERT INTO moment_likes (moment_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            moment_uuid,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    /// 取消点赞动态
    pub async fn unlike_moment(&self, moment_id: &str, user_id: i64) -> RepositoryResult<bool> {
        let moment_uuid = Uuid::parse_str(moment_id)?;
        let result = sqlx::query!(
            "DELETE FROM moment_likes WHERE moment_id = $1 AND user_id = $2",
            moment_uuid,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    

    /// 根据位置搜索附近动态
    pub async fn search_moments_by_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        radius: f64,
        cursor: i64,
        limit: i32,
    ) -> RepositoryResult<Vec<MomentRow>> {
        let results = sqlx::query_as!(
            MomentLocationRow,
            r#"
            SELECT 
                m.moment_id as "moment_id!",
                m.user_id as "user_id!",
                u.nickname as "nickname!",
                m.content,
                m.latitude as "latitude!",
                m.longitude as "longitude!",
                m.location_name as "location_name!",
                ST_Distance(
                    ST_Point($1, $2)::geography,
                    ST_Point(m.longitude, m.latitude)::geography
                ) as "distance!",
                m.created_at as "created_at!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE ST_Distance(
                ST_Point($1, $2)::geography,
                ST_Point(m.longitude, m.latitude)::geography
            ) <= $3
            AND m.id < $4
            ORDER BY m.id DESC
            LIMIT $5
            "#,
            latitude,
            longitude,
            radius,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        self.build_moment_rows(results, user_id).await
    }

    /// 根据标签搜索动态
    pub async fn search_moments_by_tags(
        &self,
        user_id: i64,
        tags: Vec<String>,
        cursor: i64,
        limit: i32,
    ) -> RepositoryResult<Vec<MomentRow>> {
        if tags.is_empty() {
            return Ok(Vec::new());
        }

        // 结构体 + 宏查询，非空字段使用 !
        let results = sqlx::query_as!(
            UserMomentRow,
            r#"
            SELECT DISTINCT
                m.id         as "id!: i64",
                m.moment_id as "moment_id!",
                m.user_id    as "user_id!",
                u.nickname   as "nickname!",
                m.content,
                m.latitude   as "latitude!",
                m.longitude  as "longitude!",
                m.location_name as "location_name!",
                m.created_at as "created_at!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            JOIN moment_tags mt ON m.moment_id = mt.moment_id
            JOIN tags t ON mt.tag_id = t.tag_id
            WHERE t.name = ANY($3) AND m.id < $1
            ORDER BY m.id DESC
            LIMIT $2
            "#,
            cursor,
            limit as i64,
            &tags[..]
        )
        .fetch_all(&self.pool)
        .await?;

        self.build_moment_rows_simple(results, user_id).await
    }

    /// 根据用户ID获取动态列表
    pub async fn search_moments_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> RepositoryResult<Vec<MomentRow>> {
        let results = sqlx::query_as!(
            UserMomentRow,
            r#"
            SELECT 
                m.id        as "id!: i64",
                m.moment_id as "moment_id!",
                m.user_id as "user_id!",
                u.nickname as "nickname!",
                m.content,
                m.latitude as "latitude!",
                m.longitude as "longitude!",
                m.location_name as "location_name!",
                m.created_at as "created_at!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.user_id = $1 AND m.id < $2
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        self.build_moment_rows_simple(results, user_id).await
    }

    /// 构建MomentRow列表（带距离信息）
    async fn build_moment_rows(&self, rows: Vec<MomentLocationRow>, user_id: i64) -> RepositoryResult<Vec<MomentRow>> {
        let mut moments = Vec::new();
        
        for row in rows {
            // 这里如后续返回分页元信息，可将 row.id 同步用于 next_cursor（当前结构无 id）
            // 为每个动态查询额外信息
            let like_count = sqlx::query_scalar!(
                "SELECT COUNT(*) as count FROM moment_likes WHERE moment_id = $1",
                row.moment_id
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(0);

            let comment_count = 0i64; // 暂时设为0

            let is_liked = sqlx::query_scalar!(
                "SELECT EXISTS(SELECT 1 FROM moment_likes WHERE moment_id = $1 AND user_id = $2)",
                row.moment_id,
                user_id
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(false);

            let tags: Vec<String> = sqlx::query_scalar!(
                "SELECT t.name FROM moment_tags mt JOIN tags t ON mt.tag_id = t.tag_id WHERE mt.moment_id = $1",
                row.moment_id
            )
            .fetch_all(&self.pool)
            .await
            .unwrap_or_default();

            moments.push(MomentRow {
                moment_id: row.moment_id.to_string(),
                user_id: row.user_id,
                user_nickname: row.nickname,
                user_avatar: None,
                content: row.content.unwrap_or_default(),
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name,
                tags,
                like_count,
                comment_count,
                is_liked,
                distance: Some(row.distance),
                created_at: row.created_at,
            });
        }

        Ok(moments)
    }

    /// 构建MomentRow列表（不带距离信息）
    async fn build_moment_rows_simple(&self, rows: Vec<UserMomentRow>, user_id: i64) -> RepositoryResult<Vec<MomentRow>> {
        let mut moments = Vec::new();
        
        for row in rows {
            // 使用游标字段以避免未读字段告警，并强调其分页语义（当前接口未返回分页元数据）
            let _cursor_id = row.id;
            // 为每个动态查询额外信息
            let like_count = sqlx::query_scalar!(
                "SELECT COUNT(*) as count FROM moment_likes WHERE moment_id = $1",
                row.moment_id
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(0);

            let comment_count = 0i64; // 暂时设为0

            let is_liked = sqlx::query_scalar!(
                "SELECT EXISTS(SELECT 1 FROM moment_likes WHERE moment_id = $1 AND user_id = $2)",
                row.moment_id,
                user_id
            )
            .fetch_one(&self.pool)
            .await?
            .unwrap_or(false);

            let tags: Vec<String> = sqlx::query_scalar!(
                "SELECT t.name FROM moment_tags mt JOIN tags t ON mt.tag_id = t.tag_id WHERE mt.moment_id = $1",
                row.moment_id
            )
            .fetch_all(&self.pool)
            .await
            .unwrap_or_default();

            moments.push(MomentRow {
                moment_id: row.moment_id.to_string(),
                user_id: row.user_id,
                user_nickname: row.nickname,
                user_avatar: None,
                content: row.content.unwrap_or_default(),
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name,
                tags,
                like_count,
                comment_count,
                is_liked,
                distance: None,
                created_at: row.created_at,
            });
        }

        Ok(moments)
    }
}

impl Repository for PgMomentRepository {
    fn name(&self) -> &str {
        "moments"
    }
}
