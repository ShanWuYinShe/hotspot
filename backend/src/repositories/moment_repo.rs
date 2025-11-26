use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 生活瞬间Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentRow {
    pub id: i64,
    pub moment_id: Uuid,
    pub cursor_id: Uuid,
    pub user_id: i64,
    pub description: String,
    pub images: Vec<String>,
    pub tags: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location: Option<String>,  // PostGIS geography type stored as WKT string
    pub location_name: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct MomentRepository {
    pool: PgPool,
}

impl MomentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 将 cursor UUID 转换为自增 ID
    /// - None: 第一次查询，返回 i64::MAX（配合 ORDER BY id DESC 获取最新记录）
    /// - Some: 查询数据库获取对应的自增 ID，不存在则返回错误
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(cursor_uuid) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM moments WHERE cursor_id = $1"#,
                    cursor_uuid
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 将点赞 cursor UUID 转换为自增 ID
    async fn resolve_like_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(cursor_uuid) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM moment_likes WHERE cursor_id = $1"#,
                    cursor_uuid
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 创建生活瞬间
    pub async fn create(
        &self,
        user_id: i64,
        description: String,
        images: Option<Vec<String>>,
        tags: Vec<String>,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Result<MomentRow> {
        let row = sqlx::query_as!(
            MomentRow,
            r#"
            INSERT INTO moments (user_id, description, images, tags, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id, moment_id, cursor_id, user_id, description, 
                      COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                      COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                      latitude, longitude, 
                      ST_AsText(location) as location,
                      location_name,
                      created_at, updated_at, is_deleted, deleted_at
            "#,
            user_id,
            description,
            images.as_deref(),
            &tags,
            latitude,
            longitude,
            location_name
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找生活瞬间
    pub async fn find_by_id(&self, moment_id: Uuid) -> Result<Option<MomentRow>> {
        let row = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, 
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                   COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                   latitude, longitude, 
                   ST_AsText(location) as location,
                   location_name,
                   created_at, updated_at, is_deleted, deleted_at
            FROM moments
            WHERE moment_id = $1 AND is_deleted = FALSE
            "#,
            moment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 批量获取瞬间
    pub async fn find_by_ids(&self, moment_ids: &[Uuid]) -> Result<Vec<MomentRow>> {
        if moment_ids.is_empty() {
            return Ok(Vec::new());
        }

        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, latitude, longitude,
                   ST_AsText(location) as location,
                   location_name, 
                   COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!",
                   created_at, updated_at, is_deleted, deleted_at
            FROM moments
            WHERE moment_id = ANY($1) AND is_deleted = FALSE
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(moments)
    }

    /// 更新生活瞬间
    pub async fn update(
        &self,
        moment_id: Uuid,
        description: Option<String>,
        images: Option<Vec<String>>,
        tags: Option<Vec<String>>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET description = COALESCE($2, description),
                images = COALESCE($3, images),
                tags = COALESCE($4, tags),
                updated_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id,
            description,
            images.as_deref(),
            tags.as_deref()
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除生活瞬间
    pub async fn soft_delete(&self, moment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO moment_likes (moment_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (moment_id, user_id) DO NOTHING
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 取消点赞瞬间
    pub async fn unlike_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM moment_likes
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 检查是否已点赞
    pub async fn is_liked(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM moment_likes 
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(result.exists)
    }

    /// 获取点赞数
    pub async fn get_like_count(&self, moment_id: Uuid) -> Result<i32> {
        let count = sqlx::query!(
            r#"SELECT COUNT(*) as "count!" FROM moment_likes WHERE moment_id = $1"#,
            moment_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;
        Ok(count)
    }

    /// 收藏瞬间
    pub async fn collect_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO moment_favorites (moment_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (moment_id, user_id) DO NOTHING
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 取消收藏瞬间
    pub async fn uncollect_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM moment_favorites
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 检查是否已收藏
    pub async fn is_collected(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM moment_favorites 
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(result.exists)
    }

    /// 获取评论数
    pub async fn get_comment_count(&self, moment_id: Uuid) -> Result<i32> {
        let count = sqlx::query!(
            r#"SELECT COUNT(*) as "count!" FROM comments WHERE moment_id = $1 AND is_deleted = FALSE"#,
            moment_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;
        Ok(count)
    }

    /// 按用户ID查询瞬间列表
    pub async fn list_by_user(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, 
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                   COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                   latitude, longitude, ST_AsText(location) as location, location_name, created_at, updated_at, is_deleted, deleted_at
            FROM moments
            WHERE user_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(moments)
    }

    /// 按位置搜索瞬间
    pub async fn search_by_location(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        let point = format!("SRID=4326;POINT({} {})", longitude, latitude);
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, 
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                   COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                   latitude, longitude, ST_AsText(location) as location, location_name, created_at, updated_at, is_deleted, deleted_at
            FROM moments
            WHERE ST_DWithin(location, ST_GeogFromText($1), $2)
              AND id < $3 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $4
            "#,
            point,
            radius_meters,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(moments)
    }

    /// 按位置搜索瞬间（带用户信息、统计数据和距离）
    pub async fn search_by_location_with_details(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<MomentWithUserStatsAndDistance>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.cursor_id, m.user_id, m.description,
                m.latitude, m.longitude, ST_AsText(m.location) as location,
                m.location_name,
                COALESCE(m.tags, ARRAY[]::TEXT[]) as "tags!",
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at, m.updated_at, m.is_deleted, m.deleted_at,
                u.nickname, u.avatar_url,
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location, 
                    ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                ) as "distance!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as likes_count
                FROM moment_likes
                GROUP BY moment_id
            ) l ON m.moment_id = l.moment_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as comments_count
                FROM comments
                WHERE is_deleted = FALSE
                GROUP BY moment_id
            ) c ON m.moment_id = c.moment_id
            WHERE m.is_deleted = FALSE 
              AND u.is_deleted = FALSE
              AND ST_DWithin(
                  m.location,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
              AND m.id < $4
            ORDER BY m.id DESC
            LIMIT $5
            "#,
            latitude,
            longitude,
            radius_meters,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentWithUserStatsAndDistance {
            moment: MomentRow {
                id: r.id,
                moment_id: r.moment_id,
                cursor_id: r.cursor_id,
                user_id: r.user_id,
                description: r.description,
                images: r.images,
                tags: r.tags,
                latitude: r.latitude,
                longitude: r.longitude,
                location: r.location,
                location_name: r.location_name,
                created_at: r.created_at,
                updated_at: r.updated_at,
                is_deleted: r.is_deleted,
                deleted_at: r.deleted_at,
            },
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance as i32,
        }).collect())
    }

    /// 按标签搜索瞬间
    pub async fn search_by_tags(&self, tags: &[String], limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, 
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                   COALESCE(tags, ARRAY[]::TEXT[]) as "tags!",
                   latitude, longitude, ST_AsText(location) as location, location_name, created_at, updated_at, is_deleted, deleted_at
            FROM moments
            WHERE tags && $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            tags,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(moments)
    }

    /// 获取用户收藏列表
    pub async fn list_collections(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT m.id, m.moment_id, m.cursor_id, m.user_id, m.description, 
                   COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
                   COALESCE(m.tags, ARRAY[]::TEXT[]) as "tags!",
                   m.latitude, m.longitude, ST_AsText(m.location) as location, m.location_name, m.created_at, m.updated_at, m.is_deleted, m.deleted_at
            FROM moments m
            JOIN moment_favorites mf ON m.moment_id = mf.moment_id
            WHERE mf.user_id = $1 AND m.id < $2 AND m.is_deleted = FALSE
            ORDER BY mf.created_at DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(moments)
    }

    /// 检查用户是否点赞了瞬间
    pub async fn is_liked_by(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let count = sqlx::query_scalar!(
            r#"SELECT COUNT(*) FROM moment_likes WHERE moment_id = $1 AND user_id = $2"#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(count.unwrap_or(0) > 0)
    }

    /// 检查用户是否收藏了瞬间
    pub async fn is_favorited_by(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let count = sqlx::query_scalar!(
            r#"SELECT COUNT(*) FROM moment_favorites WHERE moment_id = $1 AND user_id = $2"#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(count.unwrap_or(0) > 0)
    }

    /// 获取瞬间点赞列表
    pub async fn list_likes(&self, moment_id: Uuid, limit: i64, cursor: Option<Uuid>) -> Result<Vec<MomentLikeRow>> {
        let cursor_pk = self.resolve_like_cursor(cursor).await?;
        
        let likes = sqlx::query_as!(
            MomentLikeRow,
            r#"
            SELECT id, cursor_id, moment_id, user_id, created_at
            FROM moment_likes
            WHERE moment_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            moment_id,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(likes)
    }

    /// 批量获取瞬间的点赞和评论数量
    pub async fn get_stats_batch(&self, moment_ids: &[Uuid]) -> Result<std::collections::HashMap<Uuid, (i64, i64)>> {
        if moment_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id as "moment_id!",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!"
            FROM unnest($1::uuid[]) as m(moment_id)
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as likes_count
                FROM moment_likes
                GROUP BY moment_id
            ) l ON m.moment_id = l.moment_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as comments_count
                FROM comments
                WHERE is_deleted = FALSE
                GROUP BY moment_id
            ) c ON m.moment_id = c.moment_id
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.moment_id, (r.likes_count, r.comments_count))).collect())
    }

    /// 获取最新瞬间列表（带用户信息和统计数据，用于 Feed）
    pub async fn list_latest_with_user_and_stats(
        &self,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<MomentWithUserAndStats>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.cursor_id, m.user_id, m.description,
                m.latitude, m.longitude, ST_AsText(m.location) as location,
                m.location_name,
                COALESCE(m.tags, ARRAY[]::TEXT[]) as "tags!",
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at, m.updated_at, m.is_deleted, m.deleted_at,
                u.nickname, u.avatar_url,
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as likes_count
                FROM moment_likes
                GROUP BY moment_id
            ) l ON m.moment_id = l.moment_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as comments_count
                FROM comments
                WHERE is_deleted = FALSE
                GROUP BY moment_id
            ) c ON m.moment_id = c.moment_id
            WHERE m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY m.created_at DESC
            LIMIT $1 OFFSET $2
            "#,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentWithUserAndStats {
            moment: MomentRow {
                id: r.id,
                moment_id: r.moment_id,
                cursor_id: r.cursor_id,
                user_id: r.user_id,
                description: r.description,
                images: r.images,
                tags: r.tags,
                latitude: r.latitude,
                longitude: r.longitude,
                location: r.location,
                location_name: r.location_name,
                created_at: r.created_at,
                updated_at: r.updated_at,
                is_deleted: r.is_deleted,
                deleted_at: r.deleted_at,
            },
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
        }).collect())
    }
}

/// 瞬间带用户信息和统计数据
#[derive(Debug)]
pub struct MomentWithUserAndStats {
    pub moment: MomentRow,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub likes_count: i64,
    pub comments_count: i64,
}

/// 瞬间带用户信息、统计数据和距离
#[derive(Debug)]
pub struct MomentWithUserStatsAndDistance {
    pub moment: MomentRow,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub likes_count: i64,
    pub comments_count: i64,
    pub distance: i32,
}

#[derive(Debug)]
#[allow(dead_code)]
pub struct MomentLikeRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}
