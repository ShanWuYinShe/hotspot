//! 动态瞬间仓库层实现
//!
//! 适配新 Schema: moment_id (UUID), user_id (BIGINT), 分区表 moments_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::HashMap;
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::{PrivacyLevel, UserType};
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeIdCursor};

/// 瞬间基础行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentRow {
    pub moment_id: Uuid,
    pub user_id: i64,
    pub description: Option<String>,
    pub mentions: Option<Vec<i64>>,
    pub location_name: Option<String>,
    pub privacy_level: PrivacyLevel,
    pub view_count: i32,
    pub like_count: i32,
    pub comments_count: i32,
    pub share_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 瞬间详情行（带用户信息）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentDetailRow {
    pub moment_id: Uuid,
    pub user_id: i64,
    pub description: Option<String>,
    pub mentions: Option<Vec<i64>>,
    pub location_name: Option<String>,
    pub privacy_level: PrivacyLevel,
    pub view_count: i32,
    pub like_count: i32,
    pub comments_count: i32,
    pub share_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,

    // Joined fields
    pub user_nickname: Option<String>,
    pub user_avatar_url: Option<String>,

    // LBS fields (Result of ST_X/Y or ST_Distance)
    pub longitude: Option<f64>,
    pub latitude: Option<f64>,
    pub distance: Option<f64>,
}

/// Feed 流瞬间行（带用户信息 + 图片）
#[derive(Debug, Clone, FromRow)]
pub struct FeedMomentRow {
    pub moment_id: Uuid,
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub description: String,
    pub images: Vec<String>,
    pub mentions: Option<Vec<i64>>,
    pub created_at: DateTime<Utc>,
    pub likes_count: i32,
    pub comments_count: i32,
    pub location_name: Option<String>,
    pub longitude: Option<f64>,
    pub latitude: Option<f64>,
    pub distance: Option<f64>,
    pub bearing_degrees: Option<f64>,
}

/// 搜索建议瞬间行
#[derive(Debug, Clone, FromRow)]
pub struct MomentSuggestRow {
    pub moment_id: Uuid,
    pub description: String,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub distance: Option<f64>,
}

/// 瞬间列表行（不含用户信息）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MomentListRow {
    pub moment_id: Uuid,
    pub description: Option<String>,
    pub mentions: Option<Vec<i64>>,
    pub location_name: Option<String>,
    pub longitude: Option<f64>,
    pub latitude: Option<f64>,
    pub like_count: i32,
    pub comments_count: i32,
    pub created_at: DateTime<Utc>,
}

/// 瞬间图片行
#[derive(Debug, Clone, FromRow)]
pub struct MomentImageRow {
    pub moment_id: Uuid,
    pub url: String,
    pub width: Option<i32>,
    pub height: Option<i32>,
}

/// 瞬间点赞用户行
#[derive(Debug, Clone, FromRow)]
pub struct MomentLikeRow {
    pub user_id: i64,
    pub user_nickname: Option<String>,
    pub user_avatar_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct MomentRepository {
    pool: PgPool,
}

impl MomentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建瞬间
    pub async fn create(
        &self,
        user_id: i64,
        description: Option<String>,
        mentions: Option<Vec<i64>>,
        lon: Option<f64>,
        lat: Option<f64>,
        location_name: Option<String>,
        privacy_level: String,
    ) -> Result<MomentRow> {
        // 使用 query_as! 进行编译期类型检查
        let row = sqlx::query_as!(
            MomentRow,
            r#"
            INSERT INTO moments_partitioned (user_id, description, mentions, location, location_name, privacy_level)
            VALUES ($1, $2, $3,
                    CASE WHEN $4::float8 IS NOT NULL AND $5::float8 IS NOT NULL 
                         THEN ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography 
                         ELSE NULL END, 
                    $6, $7)
            RETURNING moment_id as "moment_id!", user_id as "user_id!",
                      description, mentions, location_name,
                      privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                      0::int as "view_count!", 0::int as "like_count!", 0::int as "comments_count!", 0::int as "share_count!",
                      created_at as "created_at!", updated_at as "updated_at!", deleted_at
            "#,
            user_id,
            description,
            mentions.as_deref(),
            lon,
            lat,
            location_name,
            privacy_level
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据 ID 查找
    pub async fn find_by_id(&self, moment_id: Uuid) -> Result<Option<MomentRow>> {
        let row = sqlx::query_as!(
            MomentRow,
            r#"
            SELECT moment_id as "moment_id!", user_id as "user_id!",
                   description, mentions, location_name,
                   privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   view_count::int as "view_count!", like_count::int as "like_count!",
                   comments_count::int as "comments_count!", share_count::int as "share_count!",
                   created_at as "created_at!", updated_at as "updated_at!", deleted_at
            FROM moments
            WHERE moment_id = $1 AND deleted_at IS NULL
            "#,
            moment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取动态详情（带用户信息和坐标）
    pub async fn get_detail(
        &self,
        moment_id: Uuid,
        viewer_lon: Option<f64>,
        viewer_lat: Option<f64>,
    ) -> Result<Option<MomentDetailRow>> {
        let row = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   CASE WHEN $2::float8 IS NOT NULL AND $3::float8 IS NOT NULL 
                        THEN ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
                        ELSE NULL END as distance
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.moment_id = $1 AND m.deleted_at IS NULL
            "#,
            moment_id,
            viewer_lon,
            viewer_lat
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 关注流（仅关注 + 自己）
    pub async fn get_following_feed(
        &self,
        user_id: i64,
        viewer_lat: f64,
        viewer_lng: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<FeedMomentRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            FeedMomentRow,
            r#"
            SELECT m.moment_id as "moment_id!",
                   m.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(m.description, '') as "description!",
                   COALESCE(
                       ARRAY_REMOVE(ARRAY_AGG(mi.url ORDER BY mi.sort_order ASC, mi.created_at ASC), NULL),
                       ARRAY[]::text[]
                   ) as "images!",
                   m.mentions,
                   m.created_at as "created_at!",
                   m.like_count::int as "likes_count!",
                   m.comments_count::int as "comments_count!",
                   m.location_name,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   CASE WHEN m.location IS NOT NULL
                        THEN ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
                        ELSE NULL END as distance,
                   CASE WHEN m.location IS NOT NULL
                        THEN degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($2, $3), 4326), m.location::geometry))
                        ELSE NULL END as bearing_degrees
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN moment_images mi ON m.moment_id = mi.moment_id
            LEFT JOIN follows f ON f.following_id = m.user_id AND f.follower_id = $1
            WHERE m.deleted_at IS NULL
              AND (m.user_id = $1 OR f.follower_id IS NOT NULL)
              AND (
                  m.user_id = $1
                  OR m.privacy_level = 'public'
                  OR (
                      m.privacy_level = 'friends'
                      AND EXISTS(
                          SELECT 1 FROM follows f2 WHERE f2.follower_id = m.user_id AND f2.following_id = $1
                      )
                  )
              )
              AND (m.created_at, m.moment_id) < ($4, $5)
            GROUP BY m.moment_id, m.user_id, u.nickname, u.avatar_url, u.user_type,
                     m.description, m.mentions, m.created_at, m.like_count, m.comments_count,
                     m.location_name, m.location
            ORDER BY m.created_at DESC, m.moment_id DESC
            LIMIT $6
            "#,
            user_id,
            viewer_lng,
            viewer_lat,
            last_ts,
            last_id,
            limit as i64 + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 附近流（公共）
    pub async fn get_nearby_feed(
        &self,
        viewer_lat: f64,
        viewer_lng: f64,
        radius: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<FeedMomentRow>, Option<String>)> {
        #[derive(Debug, Serialize, Deserialize)]
        struct DistanceMomentCursor(f64, Uuid);

        let (last_dist, last_id) = if let Some(c) = cursor {
            let DistanceMomentCursor(dist, id) = decode_cursor::<DistanceMomentCursor>(&c)?;
            (dist, id)
        } else {
            (0.0, Uuid::nil())
        };

        let rows = sqlx::query_as!(
            FeedMomentRow,
            r#"
            SELECT m.moment_id as "moment_id!",
                   m.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(m.description, '') as "description!",
                   COALESCE(
                       ARRAY_REMOVE(ARRAY_AGG(mi.url ORDER BY mi.sort_order ASC, mi.created_at ASC), NULL),
                       ARRAY[]::text[]
                   ) as "images!",
                   m.mentions,
                   m.created_at as "created_at!",
                   m.like_count::int as "likes_count!",
                   m.comments_count::int as "comments_count!",
                   m.location_name,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   ST_Distance(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as distance,
                   degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), m.location::geometry)) as bearing_degrees
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN moment_images mi ON m.moment_id = mi.moment_id
            WHERE m.deleted_at IS NULL
              AND m.privacy_level = 'public'
              AND m.location IS NOT NULL
              AND ST_DWithin(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND (
                  ST_Distance(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) > $4
                  OR (ST_Distance(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) = $4 AND m.moment_id > $5)
              )
            GROUP BY m.moment_id, m.user_id, u.nickname, u.avatar_url, u.user_type,
                     m.description, m.mentions, m.created_at, m.like_count, m.comments_count,
                     m.location_name, m.location
            ORDER BY distance ASC, m.moment_id ASC
            LIMIT $6
            "#,
            viewer_lng,
            viewer_lat,
            radius,
            last_dist,
            last_id,
            limit as i64 + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| {
                    encode_cursor(&DistanceMomentCursor(
                        r.distance.unwrap_or(0.0),
                        r.moment_id,
                    ))
                })
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 最新流（公共）
    pub async fn list_latest_feed(
        &self,
        viewer_lat: f64,
        viewer_lng: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<FeedMomentRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            FeedMomentRow,
            r#"
            SELECT m.moment_id as "moment_id!",
                   m.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(m.description, '') as "description!",
                   COALESCE(
                       ARRAY_REMOVE(ARRAY_AGG(mi.url ORDER BY mi.sort_order ASC, mi.created_at ASC), NULL),
                       ARRAY[]::text[]
                   ) as "images!",
                   m.mentions,
                   m.created_at as "created_at!",
                   m.like_count::int as "likes_count!",
                   m.comments_count::int as "comments_count!",
                   m.location_name,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   CASE WHEN m.location IS NOT NULL
                        THEN ST_Distance(m.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography)
                        ELSE NULL END as distance,
                   CASE WHEN m.location IS NOT NULL
                        THEN degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), m.location::geometry))
                        ELSE NULL END as bearing_degrees
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN moment_images mi ON m.moment_id = mi.moment_id
            WHERE m.deleted_at IS NULL
              AND m.privacy_level = 'public'
              AND (m.created_at, m.moment_id) < ($3, $4)
            GROUP BY m.moment_id, m.user_id, u.nickname, u.avatar_url, u.user_type,
                     m.description, m.mentions, m.created_at, m.like_count, m.comments_count,
                     m.location_name, m.location
            ORDER BY m.created_at DESC, m.moment_id DESC
            LIMIT $5
            "#,
            viewer_lng,
            viewer_lat,
            last_ts,
            last_id,
            limit as i64 + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 热门瞬间（按点赞数）
    pub async fn get_hot_moments(
        &self,
        limit: i64,
        since: DateTime<Utc>,
        cursor: Option<String>,
    ) -> Result<(Vec<FeedMomentRow>, Option<String>)> {
        #[derive(Debug, Serialize, Deserialize)]
        struct HotMomentCursor(i32, DateTime<Utc>, Uuid);

        let (last_like, last_ts, last_id) = if let Some(c) = cursor {
            let HotMomentCursor(like_count, ts, id) = decode_cursor::<HotMomentCursor>(&c)?;
            (like_count, ts, id)
        } else {
            (i32::MAX, DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            FeedMomentRow,
            r#"
            SELECT m.moment_id as "moment_id!",
                   m.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(m.description, '') as "description!",
                   COALESCE(
                       ARRAY_REMOVE(ARRAY_AGG(mi.url ORDER BY mi.sort_order ASC, mi.created_at ASC), NULL),
                       ARRAY[]::text[]
                   ) as "images!",
                   m.mentions,
                   m.created_at as "created_at!",
                   m.like_count::int as "likes_count!",
                   m.comments_count::int as "comments_count!",
                   m.location_name,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   NULL::float8 as distance,
                   NULL::float8 as bearing_degrees
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            LEFT JOIN moment_images mi ON m.moment_id = mi.moment_id
            WHERE m.deleted_at IS NULL
              AND m.privacy_level = 'public'
              AND m.created_at >= $1
              AND (
                  m.like_count < $2
                  OR (m.like_count = $2 AND m.created_at < $3)
                  OR (m.like_count = $2 AND m.created_at = $3 AND m.moment_id < $4)
              )
            GROUP BY m.moment_id, m.user_id, u.nickname, u.avatar_url, u.user_type,
                     m.description, m.mentions, m.created_at, m.like_count, m.comments_count,
                     m.location_name, m.location
            ORDER BY m.like_count DESC, m.created_at DESC, m.moment_id DESC
            LIMIT $5
            "#,
            since,
            last_like as i64,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&HotMomentCursor(r.likes_count, r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 搜索建议：按内容关键字匹配
    pub async fn suggest_by_keyword(
        &self,
        keyword: &str,
        viewer_id: Option<i64>,
        limit: i32,
        location: Option<(f64, f64)>,
    ) -> Result<Vec<MomentSuggestRow>> {
        let pattern = format!("%{}%", keyword);
        let (lat, lon) = location
            .map(|(lat, lon)| (Some(lat), Some(lon)))
            .unwrap_or((None, None));

        let rows = sqlx::query_as!(
            MomentSuggestRow,
            r#"
            SELECT
                m.moment_id as "moment_id!",
                COALESCE(m.description, '') as "description!",
                COALESCE(u.nickname, '') as "nickname!",
                u.avatar_url as "avatar_url?",
                CASE
                    WHEN $3::float8 IS NULL OR $4::float8 IS NULL OR m.location IS NULL
                        THEN NULL
                    ELSE ST_Distance(m.location, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)
                END as distance
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.deleted_at IS NULL
              AND COALESCE(m.description, '') ILIKE $1
              AND (
                    ($2::bigint IS NULL AND m.privacy_level = 'public')
                 OR ($2::bigint IS NOT NULL AND (
                        m.user_id = $2
                        OR m.privacy_level = 'public'
                        OR (
                            m.privacy_level = 'friends'
                            AND EXISTS(
                                SELECT 1 FROM follows f
                                WHERE f.follower_id = m.user_id AND f.following_id = $2
                            )
                        )
                    ))
                )
            ORDER BY distance ASC NULLS LAST, m.created_at DESC, m.moment_id DESC
            LIMIT $5
            "#,
            pattern,
            viewer_id,
            lat,
            lon,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 列出指定用户的瞬间
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentListRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            MomentListRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.description, m.mentions, m.location_name,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   m.like_count::int as "like_count!", m.comments_count::int as "comments_count!",
                   m.created_at as "created_at!"
            FROM moments m
            WHERE m.user_id = $1 AND m.deleted_at IS NULL
              AND (m.created_at, m.moment_id) < ($2, $3)
            ORDER BY m.created_at DESC, m.moment_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 按关键字搜索瞬间
    pub async fn search_by_keyword(
        &self,
        kw: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentDetailRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let pattern = format!("%{}%", kw);
        let rows = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   CASE WHEN m.location IS NOT NULL
                        THEN ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
                        ELSE NULL END as distance
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.deleted_at IS NULL
              AND m.description ILIKE $1
              AND (m.created_at, m.moment_id) < ($4, $5)
            ORDER BY m.created_at DESC, m.moment_id DESC
            LIMIT $6
            "#,
            pattern,
            lon,
            lat,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 按标签搜索瞬间
    pub async fn search_by_tag(
        &self,
        tag: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentDetailRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   CASE WHEN m.location IS NOT NULL
                        THEN ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
                        ELSE NULL END as distance
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            JOIN moment_tags mt ON m.moment_id = mt.moment_id
            JOIN tags t ON t.tag_id = mt.tag_id
            WHERE m.deleted_at IS NULL
              AND t.name = $1
              AND (m.created_at, m.moment_id) < ($4, $5)
            ORDER BY m.created_at DESC, m.moment_id DESC
            LIMIT $6
            "#,
            tag,
            lon,
            lat,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 按位置搜索瞬间
    pub async fn search_by_location(
        &self,
        kw: Option<&str>,
        lon: f64,
        lat: f64,
        radius: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentDetailRow>, Option<String>)> {
        #[derive(Debug, Serialize, Deserialize)]
        struct DistanceMomentCursor(f64, Uuid);

        let (last_dist, last_id) = if let Some(c) = cursor {
            let DistanceMomentCursor(dist, id) = decode_cursor::<DistanceMomentCursor>(&c)?;
            (dist, id)
        } else {
            (0.0, Uuid::nil())
        };

        let pattern = kw.map(|v| format!("%{}%", v));

        let rows = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as distance
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.deleted_at IS NULL
              AND m.location IS NOT NULL
              AND ST_DWithin(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4)
              AND ($1::text IS NULL OR m.description ILIKE $1)
              AND (
                  ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) > $5
                  OR (ST_Distance(m.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) = $5 AND m.moment_id > $6)
              )
            ORDER BY distance ASC, m.moment_id ASC
            LIMIT $7
            "#,
            pattern,
            lon,
            lat,
            radius,
            last_dist,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| {
                    encode_cursor(&DistanceMomentCursor(
                        r.distance.unwrap_or(0.0),
                        r.moment_id,
                    ))
                })
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 更新瞬间
    pub async fn update(
        &self,
        moment_id: Uuid,
        description: Option<String>,
        mentions: Option<Vec<i64>>,
        lon: Option<f64>,
        lat: Option<f64>,
        location_name: Option<String>,
        privacy_level: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE moments_partitioned
            SET description = COALESCE($2, description),
                mentions = COALESCE($3, mentions),
                location = CASE WHEN $4::float8 IS NOT NULL AND $5::float8 IS NOT NULL
                                THEN ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography
                                ELSE location END,
                location_name = COALESCE($6, location_name),
                privacy_level = COALESCE($7, privacy_level),
                updated_at = NOW()
            WHERE moment_id = $1
            "#,
            moment_id,
            description,
            mentions.as_deref(),
            lon,
            lat,
            location_name,
            privacy_level
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 批量获取瞬间图片
    pub async fn list_images_batch(
        &self,
        moment_ids: &[Uuid],
    ) -> Result<HashMap<Uuid, Vec<MomentImageRow>>> {
        if moment_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query_as!(
            MomentImageRow,
            r#"
            SELECT moment_id as "moment_id!", url as "url!", width, height
            FROM moment_images
            WHERE moment_id = ANY($1)
            ORDER BY moment_id ASC, sort_order ASC, created_at ASC
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map: HashMap<Uuid, Vec<MomentImageRow>> = HashMap::new();
        for row in rows {
            map.entry(row.moment_id).or_default().push(row);
        }

        Ok(map)
    }

    /// 替换瞬间图片
    pub async fn replace_images(
        &self,
        moment_id: Uuid,
        images: &[(Uuid, String, Option<i32>, Option<i32>, i32)],
    ) -> Result<()> {
        let mut tx = self.pool.begin().await?;

        sqlx::query!("DELETE FROM moment_images WHERE moment_id = $1", moment_id)
            .execute(&mut *tx)
            .await?;

        for (idx, (file_id, url, width, height, sort_order)) in images.iter().enumerate() {
            let order = if *sort_order >= 0 {
                *sort_order
            } else {
                idx as i32
            };
            sqlx::query!(
                r#"
                INSERT INTO moment_images (moment_id, file_id, url, width, height, sort_order)
                VALUES ($1, $2, $3, $4, $5, $6)
                "#,
                moment_id,
                file_id,
                url,
                *width,
                *height,
                order
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 点赞列表
    pub async fn list_likes(
        &self,
        moment_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentLikeRow>, Option<String>)> {
        let last_ts = cursor
            .and_then(|c| decode_cursor::<DateTime<Utc>>(&c).ok())
            .unwrap_or(DateTime::<Utc>::MAX_UTC);

        let rows = sqlx::query_as!(
            MomentLikeRow,
            r#"
            SELECT mr.user_id as "user_id!",
                   u.nickname as "user_nickname?",
                   u.avatar_url as "user_avatar_url?",
                   mr.created_at as "created_at!"
            FROM moment_reactions mr
            JOIN users u ON mr.user_id = u.user_id
            WHERE mr.moment_id = $1
              AND mr.type = 'like'
              AND mr.created_at < $2
            ORDER BY mr.created_at DESC
            LIMIT $3
            "#,
            moment_id,
            last_ts,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn insert_like(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            "INSERT INTO moment_reactions (moment_id, user_id, type) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING",
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    pub async fn delete_like(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM moment_reactions WHERE moment_id = $1 AND user_id = $2 AND type = 'like'",
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn is_liked(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM moment_reactions
                WHERE moment_id = $1 AND user_id = $2 AND type = 'like'
            ) as "exists!"
            "#,
            moment_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn batch_check_liked(
        &self,
        moment_ids: &[Uuid],
        user_id: i64,
    ) -> Result<std::collections::HashSet<Uuid>> {
        if moment_ids.is_empty() {
            return Ok(std::collections::HashSet::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT moment_id as "moment_id!"
            FROM moment_reactions
            WHERE user_id = $1 AND moment_id = ANY($2) AND type = 'like'
            "#,
            user_id,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.moment_id).collect())
    }

    pub async fn is_favorited(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
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

        Ok(row)
    }

    pub async fn insert_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            "INSERT INTO moment_favorites (moment_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    pub async fn delete_favorite(&self, moment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM moment_favorites WHERE moment_id = $1 AND user_id = $2",
            moment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn batch_check_favorited(
        &self,
        moment_ids: &[Uuid],
        user_id: i64,
    ) -> Result<std::collections::HashSet<Uuid>> {
        if moment_ids.is_empty() {
            return Ok(std::collections::HashSet::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT moment_id as "moment_id!"
            FROM moment_favorites
            WHERE user_id = $1 AND moment_id = ANY($2)
            "#,
            user_id,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.moment_id).collect())
    }

    pub async fn list_favorited_moments(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentDetailRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   NULL::float8 as distance
            FROM moment_favorites mf
            JOIN moments m ON m.moment_id = mf.moment_id
            JOIN users u ON m.user_id = u.user_id
            WHERE mf.user_id = $1
              AND (mf.created_at, m.moment_id) < ($2, $3)
              AND m.deleted_at IS NULL
            ORDER BY mf.created_at DESC, m.moment_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn list_user_liked_moments(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<MomentDetailRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            MomentDetailRow,
            r#"
            SELECT m.moment_id as "moment_id!", m.user_id as "user_id!",
                   m.description, m.mentions, m.location_name,
                   m.privacy_level as "privacy_level!: crate::types::enums::PrivacyLevel",
                   m.view_count::int as "view_count!", m.like_count::int as "like_count!",
                   m.comments_count::int as "comments_count!", m.share_count::int as "share_count!",
                   m.created_at as "created_at!", m.updated_at as "updated_at!",
                   u.nickname as user_nickname, u.avatar_url as user_avatar_url,
                   ST_X(m.location::geometry) as longitude,
                   ST_Y(m.location::geometry) as latitude,
                   NULL::float8 as distance
            FROM moment_reactions mr
            JOIN moments m ON m.moment_id = mr.moment_id
            JOIN users u ON m.user_id = u.user_id
            WHERE mr.user_id = $1
              AND mr.type = 'like'
              AND (mr.created_at, m.moment_id) < ($2, $3)
              AND m.deleted_at IS NULL
            ORDER BY mr.created_at DESC, m.moment_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.moment_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 软删除动态
    pub async fn delete(&self, moment_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE moments_partitioned SET deleted_at = NOW() WHERE moment_id = $1",
            moment_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn exists(&self, moment_id: Uuid) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM moments WHERE moment_id = $1 AND deleted_at IS NULL) as \"exists!\"",
            moment_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(exists)
    }

    pub async fn list_moment_ids_after(
        &self,
        last_row_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, Uuid)>> {
        #[derive(Debug, FromRow)]
        struct MomentIdMapRow {
            row_id: i64,
            moment_id: Uuid,
        }

        let rows = sqlx::query_as!(
            MomentIdMapRow,
            r#"
            SELECT (EXTRACT(EPOCH FROM created_at) * 1000)::bigint as "row_id!",
                   moment_id as "moment_id!"
            FROM moment_id_map
            WHERE (EXTRACT(EPOCH FROM created_at) * 1000)::bigint > $1
            ORDER BY created_at ASC
            LIMIT $2
            "#,
            last_row_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.row_id, r.moment_id)).collect())
    }
}
