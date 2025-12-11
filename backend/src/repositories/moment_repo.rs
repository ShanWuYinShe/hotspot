use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 生活瞬间Row - 数据库查询返回类型
/// 注意：tags 已移至单独的 moment_tags 关联表，通过 TagRepository 获取
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentRow {
    pub id: i64,
    pub moment_id: Uuid,
    pub cursor_id: Uuid,
    pub user_id: i64,
    pub description: String,
    pub images: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location: Option<String>,  // PostGIS geography type stored as WKT string
    pub location_name: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 瞬间搜索结果Row - 包含用户信息、统计数据和距离（统一搜索结果结构）
#[derive(Debug, Clone)]
pub struct MomentSearchRow {
    pub moment_id: Uuid,
    pub user_id: i64,
    pub description: String,
    pub images: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
    // 用户信息
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
    // 统计信息
    pub likes_count: i64,
    pub comments_count: i64,
    /// PostGIS 计算的距离（米）
    pub distance: f64,
    /// PostGIS 计算的方位角（0-360度）
    pub degrees: f64,
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
    /// 注意：tags 已移至 moment_tags 关联表，创建后需单独调用 TagRepository.set_moment_tags
    pub async fn create(
        &self,
        user_id: i64,
        description: String,
        images: Option<Vec<String>>,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Result<MomentRow> {
        let row = sqlx::query_as!(
            MomentRow,
            r#"
            INSERT INTO moments (user_id, description, images, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, moment_id, cursor_id, user_id, description, 
                      COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
                      latitude, longitude, 
                      ST_AsText(location) as location,
                      location_name,
                      created_at, updated_at, is_deleted, deleted_at
            "#,
            user_id,
            description,
            images.as_deref(),
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

    /// 按ID列表查找瞬间（带用户信息、统计和距离计算）
    pub async fn find_by_ids_with_distance(
        &self,
        moment_ids: &[Uuid],
        latitude: f64,
        longitude: f64,
    ) -> Result<Vec<MomentSearchRow>> {
        if moment_ids.is_empty() {
            return Ok(Vec::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    m.location
                )), 0) as "degrees!"
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
            WHERE m.moment_id = ANY($1) AND m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY "distance!" ASC
            "#,
            moment_ids,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 更新生活瞬间
    /// 注意：tags 已移至 moment_tags 关联表，更新后需单独调用 TagRepository.set_moment_tags
    pub async fn update(
        &self,
        moment_id: Uuid,
        description: Option<String>,
        images: Option<Vec<String>>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET description = COALESCE($2, description),
                images = COALESCE($3, images),
                updated_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id,
            description,
            images.as_deref()
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

    /// 收藏瞬间
    pub async fn add_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
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
    pub async fn remove_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
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

    /// 按用户ID查询瞬间列表
    pub async fn list_by_user(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT id, moment_id, cursor_id, user_id, description, 
                   COALESCE(images, ARRAY[]::TEXT[]) as "images!", 
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


    /// 按位置搜索瞬间（带用户信息、统计数据和距离）
    /// 按位置搜索瞬间（带 PostGIS 距离计算）
    pub async fn search_by_location(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<MomentSearchRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location::geography, 
                    ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                ) as "distance!",
                COALESCE(
                    degrees(ST_Azimuth(
                        ST_SetSRID(ST_MakePoint($2, $1), 4326),
                        m.location
                    )),
                    0.0
                ) as "degrees!"
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
                  m.location::geography,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
              AND m.id < $4
            ORDER BY "distance!" ASC
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

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 按标签搜索瞬间
    /// 通过 moment_tags 和 tags 关联表进行搜索
    pub async fn search_by_tag(&self, tag_name: &str, limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT m.id, m.moment_id, m.cursor_id, m.user_id, m.description, 
                   COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
                   m.latitude, m.longitude, ST_AsText(m.location) as location, m.location_name, 
                   m.created_at, m.updated_at, m.is_deleted, m.deleted_at
            FROM moments m
            INNER JOIN moment_tags mt ON m.moment_id = mt.moment_id
            INNER JOIN tags t ON mt.tag_id = t.id
            WHERE t.name = $1 AND m.id < $2 AND m.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            tag_name,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(moments)
    }

    /// 获取用户收藏列表
    pub async fn list_favorites(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<MomentRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let moments = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT m.id, m.moment_id, m.cursor_id, m.user_id, m.description, 
                   COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
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

    /// 获取最新瞬间列表（带用户信息、统计数据和距离，用于 Feed）
    pub async fn list_latest_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<MomentSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    m.location
                )), 0) as "degrees!"
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
            LIMIT $3 OFFSET $4
            "#,
            user_lng,
            user_lat,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 获取关注用户发布的瞬间（用于 Feed，带距离计算）
    pub async fn get_following_feed(
        &self,
        user_id: i64,
        user_lat: f64,
        user_lng: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<MomentSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    m.location
                )), 0) as "degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            JOIN user_follows f ON m.user_id = f.following_id AND f.follower_id = $1
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
            LIMIT $4 OFFSET $5
            "#,
            user_id,
            user_lng,
            user_lat,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 获取附近的瞬间（用于 Feed，带距离计算）
    pub async fn get_nearby_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        radius: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<MomentSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(c.comments_count, 0) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    m.location
                )), 0) as "degrees!"
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
                  m.location::geography,
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  $3
              )
            ORDER BY "distance!" ASC, m.created_at DESC
            LIMIT $4 OFFSET $5
            "#,
            user_lng,
            user_lat,
            radius,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }

    /// 获取热门瞬间（用于 Feed trending，距离设为 0）
    pub async fn get_hot_moments(&self, limit: i64, since: DateTime<Utc>) -> Result<Vec<MomentSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
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
            WHERE m.is_deleted = FALSE 
              AND u.is_deleted = FALSE
              AND m.created_at >= $2
            ORDER BY (COALESCE(l.likes_count, 0) + COALESCE(c.comments_count, 0) * 2) DESC
            LIMIT $1
            "#,
            limit,
            since
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| MomentSearchRow {
            moment_id: r.moment_id,
            user_id: r.user_id,
            description: r.description,
            images: r.images,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
            nickname: r.nickname,
            avatar_url: r.avatar_url,
            user_type: r.user_type,
            likes_count: r.likes_count,
            comments_count: r.comments_count,
            distance: 0.0,
            degrees: 0.0,
        }).collect())
    }

    /// 获取用户点赞的瞬间ID列表
    pub async fn get_user_liked_moments(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<Uuid>> {
        let cursor_pk = self.resolve_like_cursor(cursor).await?;

        let rows: Vec<Uuid> = sqlx::query_scalar!(
            r#"
            SELECT moment_id as "moment_id!"
            FROM moment_likes
            WHERE user_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }
}

#[derive(Debug)]
pub struct MomentLikeRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}
