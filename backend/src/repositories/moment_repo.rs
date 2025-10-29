use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::moment::Moment,
};

pub struct MomentRepository<'a> {
    db: &'a PgPool,
}

impl<'a> MomentRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建瞬间
    pub async fn create(&self, moment: &Moment) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO moments (
                moment_id, user_id, content, images, tags,
                latitude, longitude, visibility
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING moment_id
            "#,
            moment.moment_id,
            moment.user_id,
            moment.content,
            &moment.images,
            &moment.tags,
            moment.latitude,
            moment.longitude,
            moment.visibility,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.moment_id)
    }

    /// 根据ID查找瞬间
    pub async fn find_by_id(&self, moment_id: Uuid) -> Result<Option<Moment>> {
        let moment = sqlx::query_as!(
            Moment,
            r#"
            SELECT * FROM moments WHERE moment_id = $1 AND deleted_at IS NULL
            "#,
            moment_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(moment)
    }

    /// 更新瞬间
    pub async fn update(&self, moment: &Moment) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET content = $2, images = $3, tags = $4,
                latitude = $5, longitude = $6, visibility = $7,
                edited_at = NOW()
            WHERE moment_id = $1
            "#,
            moment.moment_id,
            moment.content,
            &moment.images,
            &moment.tags,
            moment.latitude,
            moment.longitude,
            moment.visibility,
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 软删除瞬间
    pub async fn soft_delete(&self, moment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET deleted_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取用户的瞬间列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Moment>> {
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT * FROM moments
            WHERE user_id = $1 AND deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(moments)
    }

    /// 根据位置搜索瞬间（使用PostGIS）
    pub async fn search_by_location(
        &self,
        lat: f64,
        lon: f64,
        radius_meters: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<(Moment, f64)>> {
        let records = sqlx::query!(
            r#"
            SELECT 
                m.*,
                ST_Distance(
                    ST_MakePoint(m.longitude, m.latitude)::geography,
                    ST_MakePoint($2, $1)::geography
                ) as "distance!"
            FROM moments m
            WHERE 
                m.deleted_at IS NULL
                AND m.visibility = 'public'
                AND ST_DWithin(
                    ST_MakePoint(m.longitude, m.latitude)::geography,
                    ST_MakePoint($2, $1)::geography,
                    $3
                )
            ORDER BY distance ASC
            LIMIT $4 OFFSET $5
            "#,
            lat,
            lon,
            radius_meters,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        let results = records
            .into_iter()
            .map(|rec| {
                let moment = Moment {
                    moment_id: rec.moment_id,
                    user_id: rec.user_id,
                    content: rec.content,
                    images: rec.images,
                    tags: rec.tags,
                    latitude: rec.latitude,
                    longitude: rec.longitude,
                    visibility: rec.visibility,
                    created_at: rec.created_at,
                    edited_at: rec.edited_at,
                    deleted_at: rec.deleted_at,
                };
                (moment, rec.distance)
            })
            .collect();

        Ok(results)
    }

    /// 根据标签搜索瞬间
    pub async fn search_by_tags(
        &self,
        tags: &[String],
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Moment>> {
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT * FROM moments
            WHERE 
                deleted_at IS NULL
                AND visibility = 'public'
                AND tags && $1
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            tags,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }

    /// 获取评论数
    pub async fn get_comment_count(&self, moment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE target_id = $1 AND target_type = 'moment' AND deleted_at IS NULL
            "#,
            moment_id.to_string()
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.exists)
    }

    /// 获取用户的收藏列表
    pub async fn list_collections(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Moment>> {
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT m.* FROM moments m
            INNER JOIN moment_collections mc ON m.moment_id = mc.moment_id
            WHERE mc.user_id = $1 AND m.deleted_at IS NULL
            ORDER BY mc.created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(moments)
    }
}

