use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

use crate::utils::pagination::{decode_cursor, encode_cursor, DistanceCursor, IdCursor};

/// 生活瞬间Row - 数据库查询返回类型
/// 注意：tags 已移至单独的 moment_tags 关联表，通过 TagRepository 获取
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentRow {
    pub id: i64,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub description: String,
    pub images: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location: Option<String>, // PostGIS geography type stored as WKT string
    pub location_name: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 瞬间搜索结果Row - 包含用户信息、统计数据和距离（统一搜索结果结构）
#[derive(Debug, Clone)]
pub struct MomentSearchRow {
    // id removed as it is only used for cursor generation in repo
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
    pub bearing_degrees: f64,
}

#[derive(Debug, Clone)]
pub struct MomentRepository {
    pool: PgPool,
}

impl MomentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取瞬间摘要快照（用于事件通知）
    pub async fn get_summary_snapshot(
        &self,
        moment_id: Uuid,
    ) -> Result<Option<crate::events::event_types::MomentSummarySnapshot>> {
        let row = sqlx::query!(
            r#"
            SELECT moment_id, description, images
            FROM moments
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.map(|r| {
            let cover_url = r.images.and_then(|imgs| imgs.first().cloned());
            let text_preview = if r.description.chars().count() > 50 {
                format!("{}...", r.description.chars().take(50).collect::<String>())
            } else {
                r.description
            };

            crate::events::event_types::MomentSummarySnapshot {
                moment_id,
                cover_url,
                text_preview,
            }
        }))
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
            RETURNING id, moment_id, user_id, description, 
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

    /// 检查瞬间是否存在
    pub async fn exists(&self, moment_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"SELECT EXISTS(SELECT 1 FROM moments WHERE moment_id = $1 AND is_deleted = FALSE) as "exists!""#,
            moment_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 获取瞬间所有者ID
    pub async fn get_owner(&self, moment_id: Uuid) -> Result<Option<i64>> {
        let row = sqlx::query!(
            r#"SELECT user_id FROM moments WHERE moment_id = $1 AND is_deleted = FALSE"#,
            moment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.map(|r| r.user_id))
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
                m.created_at, m.id,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    m.location
                )), 0) as "bearing_degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.moment_id = ANY($1) AND m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY "distance!" ASC
            "#,
            moment_ids,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: r.bearing_degrees,
            })
            .collect())
    }

    /// 按关键字搜索生活瞬间（全局，按时间倒序）
    pub async fn search_by_keyword(
        &self,
        keyword: &str,
        latitude: f64,
        longitude: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let search_pattern = format!("%{}%", keyword);

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    m.location
                )), 0) as "bearing_degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.is_deleted = FALSE 
              AND u.is_deleted = FALSE
              AND m.description ILIKE $1
              AND (m.created_at < $4 OR (m.created_at = $4 AND m.id < $5))
            ORDER BY m.created_at DESC, m.id DESC
            LIMIT $6
            "#,
            search_pattern,
            longitude,
            latitude,
            last_ts,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.created_at, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: r.bearing_degrees,
            })
            .collect(),
            next_cursor,
        ))
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

    /// 更新生活瞬间的位置
    pub async fn update_location(
        &self,
        moment_id: Uuid,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments
            SET latitude = $2,
                longitude = $3,
                location_name = $4,
                updated_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id,
            latitude,
            longitude,
            location_name
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
    pub async fn like_moment(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
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
        Ok(result.rows_affected() > 0)
    }

    /// 取消点赞瞬间
    pub async fn unlike_moment(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            DELETE FROM moment_likes
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected() > 0)
    }

    /// 收藏瞬间
    pub async fn add_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
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
        Ok(result.rows_affected() > 0)
    }

    /// 取消收藏瞬间
    pub async fn remove_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            DELETE FROM moment_favorites
            WHERE moment_id = $1 AND user_id = $2
            "#,
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected() > 0)
    }

    /// 按用户ID查询瞬间列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let moments = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description, 
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
                m.latitude, m.longitude, m.location_name, m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.user_id = $1 AND m.id < $2 AND m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = moments.last().map(|m| encode_cursor(&m.id)).transpose()?;

        Ok((
            moments
                .into_iter()
                .map(|m| MomentSearchRow {
                    moment_id: m.moment_id,
                    user_id: m.user_id,
                    description: m.description,
                    images: m.images,
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    created_at: m.created_at,
                    nickname: m.nickname,
                    avatar_url: m.avatar_url,
                    user_type: m.user_type,
                    likes_count: m.likes_count,
                    comments_count: m.comments_count,
                    distance: 0.0,
                    bearing_degrees: 0.0,
                })
                .collect(),
            next_cursor,
        ))
    }

    /// 按位置搜索瞬间（带 PostGIS 距离计算）
    pub async fn search_by_location(
        &self,
        keyword: Option<&str>,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        // Keyset Pagination: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let search_pattern = keyword.map(|k| format!("%{}%", k));

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
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
                ) as "bearing_degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.is_deleted = FALSE 
              AND u.is_deleted = FALSE
              AND ($7::TEXT IS NULL OR m.description ILIKE $7)
              AND ST_DWithin(
                  m.location::geography,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              AND (
                  ST_Distance(
                      m.location::geography,
                      ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                  ) > $4
                  OR (
                      ST_Distance(
                          m.location::geography,
                          ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                      ) = $4
                      AND m.id > $5
                  )
              )
            ORDER BY "distance!" ASC, m.id ASC
            LIMIT $6
            "#,
            latitude,
            longitude,
            radius_meters,
            last_dist,
            last_id,
            limit as i64,
            search_pattern
        )
        .fetch_all(&self.pool)
        .await?;

        // Construct next cursor from last item
        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: r.bearing_degrees,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 按标签搜索瞬间
    /// 通过 moment_tags 和 tags 关联表进行搜索
    pub async fn search_by_tag(
        &self,
        tag_name: &str,
        latitude: f64,
        longitude: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let moments = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description, 
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
                m.latitude, m.longitude, m.location_name, m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint(m.longitude, m.latitude), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($4, $5), 4326),
                    ST_SetSRID(ST_MakePoint(m.longitude, m.latitude), 4326)
                )), 0) as "bearing_degrees!"
            FROM moments m
            INNER JOIN moment_tags mt ON m.moment_id = mt.moment_id
            INNER JOIN tags t ON mt.tag_id = t.id
            JOIN users u ON m.user_id = u.user_id
            WHERE t.name = $1 AND m.id < $2 AND m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            tag_name,
            last_id,
            limit as i64,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = moments.last().map(|m| encode_cursor(&m.id)).transpose()?;

        Ok((
            moments
                .into_iter()
                .map(|m| MomentSearchRow {
                    moment_id: m.moment_id,
                    user_id: m.user_id,
                    description: m.description,
                    images: m.images,
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    created_at: m.created_at,
                    nickname: m.nickname,
                    avatar_url: m.avatar_url,
                    user_type: m.user_type,
                    likes_count: m.likes_count,
                    comments_count: m.comments_count,
                    distance: m.distance,
                        bearing_degrees: m.bearing_degrees,
                })
                .collect(),
            next_cursor,
        ))
    }

    /// 获取用户收藏列表
    pub async fn list_favorites(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<(MomentSearchRow, DateTime<Utc>)>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description, 
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!", 
                m.latitude, m.longitude, m.location_name, m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                mf.created_at as favorited_at,
                ST_Distance(
                    ST_SetSRID(ST_MakePoint(m.longitude, m.latitude), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($5, $6), 4326),
                    ST_SetSRID(ST_MakePoint(m.longitude, m.latitude), 4326)
                )), 0) as "bearing_degrees!"
            FROM moments m
            JOIN moment_favorites mf ON m.moment_id = mf.moment_id
            JOIN users u ON m.user_id = u.user_id
            WHERE mf.user_id = $1 
              AND (mf.created_at < $2 OR (mf.created_at = $2 AND m.id < $3))
              AND m.is_deleted = FALSE AND u.is_deleted = FALSE
            ORDER BY mf.created_at DESC, m.id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit as i64,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.favorited_at, r.id)))
            .transpose()?;

        let results = rows
            .into_iter()
            .map(|r| {
                (
                    MomentSearchRow {
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
                        bearing_degrees: r.bearing_degrees,
                    },
                    r.favorited_at,
                )
            })
            .collect();

        Ok((results, next_cursor))
    }

    /// 检查用户是否点赞了瞬间
    pub async fn is_liked_by(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM moment_likes
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(exists)
    }

    /// 检查用户是否收藏了瞬间
    pub async fn is_favorited_by(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM moment_favorites
                WHERE moment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(exists)
    }

    /// 获取瞬间点赞列表
    pub async fn list_likes(
        &self,
        moment_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentLikeRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let likes = sqlx::query_as!(
            MomentLikeRow,
            r#"
            SELECT id, moment_id, user_id, created_at
            FROM moment_likes
            WHERE moment_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            moment_id,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = likes.last().map(|l| encode_cursor(&l.id)).transpose()?;
        Ok((likes, next_cursor))
    }

    /// 批量获取瞬间的点赞和评论数量
    pub async fn get_stats_batch(
        &self,
        moment_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, (i64, i64)>> {
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
                WHERE moment_id = ANY($1)
                GROUP BY moment_id
            ) l ON m.moment_id = l.moment_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as comments_count
                FROM comments
                WHERE is_deleted = FALSE AND moment_id = ANY($1)
                GROUP BY moment_id
            ) c ON m.moment_id = c.moment_id
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.moment_id, (r.likes_count, r.comments_count)))
            .collect())
    }

    /// 获取最新瞬间列表（带用户信息、统计数据和距离，用于 Feed）
    pub async fn list_latest_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    m.location
                )), 0) as "bearing_degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.is_deleted = FALSE AND u.is_deleted = FALSE
              AND (m.created_at < $3 OR (m.created_at = $3 AND m.id < $4))
            ORDER BY m.created_at DESC, m.id DESC
            LIMIT $5
            "#,
            user_lng,
            user_lat,
            last_ts,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.created_at, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: r.bearing_degrees,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 获取关注用户发布的瞬间（用于 Feed，带距离计算）
    pub async fn get_following_feed(
        &self,
        user_id: i64,
        user_lat: f64,
        user_lng: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                ST_Distance(
                    m.location::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    m.location
                )), 0) as "bearing_degrees!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            JOIN user_follows f ON m.user_id = f.following_id AND f.follower_id = $1
            WHERE m.is_deleted = FALSE AND u.is_deleted = FALSE
              AND (m.created_at < $4 OR (m.created_at = $4 AND m.id < $5))
            ORDER BY m.created_at DESC, m.id DESC
            LIMIT $6
            "#,
            user_id,
            user_lng,
            user_lat,
            last_ts,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.created_at, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: r.bearing_degrees,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 获取附近的瞬间（用于 Feed，带距离计算）
    pub async fn get_nearby_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        radius: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        // Reuse search_by_location logic without keyword
        self.search_by_location(None, user_lat, user_lng, radius, limit, cursor)
            .await
    }

    /// 获取热门瞬间（用于 Feed trending，距离设为 0）
    pub async fn get_hot_moments(
        &self,
        limit: i32,
        since: DateTime<Utc>,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentSearchRow>, Option<String>)> {
        // Keyset Pagination: (score, id) where score = likes + comments * 2
        // We order by score DESC, id DESC
        let (last_score, last_id) = if let Some(c) = cursor {
            decode_cursor::<(i64, i64)>(&c)?
        } else {
            (i64::MAX, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                m.id, m.moment_id, m.user_id, m.description,
                m.latitude, m.longitude, m.location_name,
                COALESCE(m.images, ARRAY[]::TEXT[]) as "images!",
                m.created_at,
                u.nickname, u.avatar_url, u.user_type as "user_type!: crate::types::enums::UserType",
                (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) as "likes_count!",
                (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) as "comments_count!",
                (
                    (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) + 
                    (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) * 2
                ) as "score!"
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.is_deleted = FALSE 
              AND u.is_deleted = FALSE
              AND m.created_at >= $2
              AND (
                  (
                    (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) + 
                    (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) * 2
                  ) < $3
                  OR (
                      (
                        (SELECT COUNT(*) FROM moment_likes ml WHERE ml.moment_id = m.moment_id) + 
                        (SELECT COUNT(*) FROM comments mc WHERE mc.moment_id = m.moment_id AND mc.is_deleted = FALSE) * 2
                      ) = $3
                      AND m.id < $4
                  )
              )
            ORDER BY "score!" DESC, m.id DESC
            LIMIT $1
            "#,
            limit as i64,
            since,
            last_score,
            last_id
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.score, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| MomentSearchRow {
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
                bearing_degrees: 0.0,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 获取用户点赞的瞬间ID列表 (返回Row以便分页)
    pub async fn get_user_liked_moments(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentLikeRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            MomentLikeRow,
            r#"
            SELECT id, moment_id, user_id, created_at
            FROM moment_likes
            WHERE user_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((rows, next_cursor))
    }
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentLikeRow {
    pub id: i64,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}
