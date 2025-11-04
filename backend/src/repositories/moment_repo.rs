use sqlx::{PgPool, Row};
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::moment::Moment,
};

#[derive(Debug, Clone)]
pub struct MomentRepository {
    db: PgPool,
}

impl MomentRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 创建瞬间
    pub async fn create(&self, moment: &Moment) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO moments (
                moment_id, user_id, description, images, tags,
                latitude, longitude, location_name
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING moment_id
            "#,
            moment.moment_id,
            moment.user_id,
            moment.description,
            moment.images.as_deref(),
            moment.tags.as_deref(),
            moment.latitude,
            moment.longitude,
            moment.location_name,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.moment_id)
    }

    /// 根据ID查找瞬间
    pub async fn find_by_id(&self, moment_id: Uuid) -> Result<Option<Moment>> {
        let moment = sqlx::query_as!(
            Moment,
            r#"
            SELECT moment_id, user_id, description, latitude, longitude, 
                   location_name, tags, images, created_at, updated_at,
                   is_deleted, deleted_at
            FROM moments 
            WHERE moment_id = $1 AND is_deleted = FALSE
            "#,
            moment_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(moment)
    }

    /// 更新瞬间
    pub async fn update(&self, moment: &Moment) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET description = $2, images = $3, tags = $4,
                latitude = $5, longitude = $6, location_name = $7,
                updated_at = NOW()
            WHERE moment_id = $1
            "#,
            moment.moment_id,
            moment.description,
            moment.images.as_deref(),
            moment.tags.as_deref(),
            moment.latitude,
            moment.longitude,
            moment.location_name,
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 删除瞬间（软删除）
    pub async fn soft_delete(&self, moment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取用户的瞬间列表
    /// 列出用户的瞬间（游标分页）
    /// cursor: moment_id (UUID字符串)
    pub async fn list_by_user(
        &self,
        user_id: i64,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Moment>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let moment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM moments WHERE moment_id = $1 AND is_deleted = FALSE",
                moment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT moment_id, user_id, description, latitude, longitude, 
                   location_name, tags, images, created_at, updated_at,
                   is_deleted, deleted_at
            FROM moments
            WHERE user_id = $1 
            AND is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(moments)
    }

    /// 根据位置搜索瞬间（使用PostGIS）
    /// 按位置搜索瞬间（游标分页）
    /// cursor: moment_id (UUID字符串)
    pub async fn search_by_location(
        &self,
        lat: f64,
        lon: f64,
        radius_meters: f64,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<(Moment, f64)>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let moment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM moments WHERE moment_id = $1 AND is_deleted = FALSE",
                moment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页，按distance排序，id作为第二排序键确保稳定性
        let records = sqlx::query(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description, m.latitude, m.longitude,
                m.location_name, m.tags, m.images, m.created_at, m.updated_at,
                m.is_deleted, m.deleted_at,
                ST_Distance(
                    ST_MakePoint(m.longitude, m.latitude)::geography,
                    ST_MakePoint($2, $1)::geography
                ) as distance
            FROM moments m
            WHERE 
                m.is_deleted = FALSE
                AND ST_DWithin(
                    ST_MakePoint(m.longitude, m.latitude)::geography,
                    ST_MakePoint($2, $1)::geography,
                    $3
                )
                AND ($4::BIGINT IS NULL OR m.id > $4::BIGINT)
            ORDER BY distance ASC, m.id ASC
            LIMIT $5
            "#
        )
        .bind(lat)
        .bind(lon)
        .bind(radius_meters)
        .bind(cursor_id)
        .bind(limit)
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        let results = records
            .iter()
            .map(|rec| {
                let moment = Moment {
                    moment_id: rec.get("moment_id"),
                    user_id: rec.get("user_id"),
                    description: rec.get("description"),
                    latitude: rec.get("latitude"),
                    longitude: rec.get("longitude"),
                    location_name: rec.get("location_name"),
                    tags: rec.get("tags"),
                    images: rec.get("images"),
                    created_at: rec.get("created_at"),
                    updated_at: rec.get("updated_at"),
                    is_deleted: rec.get("is_deleted"),
                    deleted_at: rec.get("deleted_at"),
                };
                let distance: f64 = rec.get("distance");
                (moment, distance)
            })
            .collect();

        Ok(results)
    }

    /// 根据标签搜索瞬间
    /// 按标签搜索瞬间（游标分页）
    /// cursor: moment_id (UUID字符串)
    pub async fn search_by_tags(
        &self,
        tags: &[String],
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Moment>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let moment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM moments WHERE moment_id = $1 AND is_deleted = FALSE",
                moment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT moment_id, user_id, description, latitude, longitude, 
                   location_name, tags, images, created_at, updated_at,
                   is_deleted, deleted_at
            FROM moments
            WHERE tags && $1 
            AND is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            tags,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(moments)
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
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM moment_likes
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 收藏瞬间
    pub async fn collect_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO moment_collections (moment_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (moment_id, user_id) DO NOTHING
            "#,
            moment_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 取消收藏
    pub async fn uncollect_moment(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM moment_collections
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取点赞数
    pub async fn get_like_count(&self, moment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM moment_likes
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count)
    }

    /// 获取评论数
    pub async fn get_comment_count(&self, moment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count)
    }

    /// 检查是否已点赞
    pub async fn is_liked(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM moment_likes
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.exists)
    }

    /// 检查是否已收藏
    pub async fn is_collected(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM moment_collections
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.exists)
    }

    /// 获取用户的收藏列表
    /// 获取用户收藏的瞬间列表（游标分页）
    /// cursor: moment_id (UUID字符串)
    pub async fn list_collections(
        &self,
        user_id: i64,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Moment>> {
        // 1. 解析cursor获取对应的moment_collections.id
        let cursor_id = if let Some(cursor_str) = cursor {
            let moment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                r#"
                SELECT mc.id
                FROM moment_collections mc
                WHERE mc.user_id = $1 AND mc.moment_id = $2
                "#,
                user_id,
                moment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用moment_collections.id进行游标分页
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT m.moment_id, m.user_id, m.description, m.latitude, m.longitude,
                   m.location_name, m.tags, m.images, m.created_at, m.updated_at,
                   m.is_deleted, m.deleted_at
            FROM moments m
            INNER JOIN moment_collections mc ON m.moment_id = mc.moment_id
            WHERE mc.user_id = $1 
            AND m.is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR mc.id > $2::BIGINT)
            ORDER BY mc.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(moments)
    }
}

