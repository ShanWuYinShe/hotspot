//! 地图仓库层实现
//!
//! 基于 PostGIS 提供高性能的 LBS 聚类和实体检索

use crate::error::Result;
use sqlx::{FromRow, PgPool};

#[derive(Debug, Clone)]
pub struct MapRepository {
    pool: PgPool,
}

impl MapRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

/// 地图聚类结果
#[derive(Debug, FromRow)]
pub struct MapClusterRow {
    pub key: String,
    pub latitude: f64,
    pub longitude: f64,
    pub count: i64,
    pub preview_urls: Vec<String>,
}

/// 地图实体结果
#[derive(Debug, FromRow)]
pub struct MapEntityRow {
    pub id: String,
    pub title: String,
    pub subtitle: Option<String>,
    pub cover_url: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
}

impl MapRepository {
    /// 聚类查询附近的用户
    pub async fn cluster_users(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as!(
            MapClusterRow,
            r#"
            WITH env AS (SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox),
            snapped AS (
              SELECT ST_SnapToGrid(ST_Transform(u.location::geometry, 3857), $5) AS p,
                     u.avatar_url
              FROM users u, env
              WHERE u.status = 'normal' AND u.deleted_at IS NULL AND ST_Intersects(u.location, env.bbox::geography)
            )
            SELECT 
              CONCAT(FLOOR(ST_X(p)/$5), ':', FLOOR(ST_Y(p)/$5)) as "key!",
              ST_Y(ST_Transform(p, 4326)) as "latitude!",
              ST_X(ST_Transform(p, 4326)) as "longitude!",
              COUNT(*) as "count!",
              COALESCE(ARRAY_REMOVE(ARRAY_AGG(avatar_url), NULL), ARRAY[]::text[]) as "preview_urls!"
            FROM snapped GROUP BY p
            "#,
            west, south, east, north, grid_meters
        ).fetch_all(&self.pool).await?;
        Ok(rows)
    }

    /// 与 map_service 对齐的接口名称
    pub async fn cluster_users_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        self.cluster_users(west, south, east, north, grid_meters)
            .await
    }

    /// 搜索矩形区域内的用户
    pub async fn search_users(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as!(
            MapEntityRow,
            r#"
            SELECT 
                u.user_id::text as "id!",
                u.nickname as "title!",
                u.bio as subtitle,
                u.avatar_url as cover_url,
                ST_Y(u.location::geometry) as "latitude!",
                ST_X(u.location::geometry) as "longitude!"
            FROM users u
            WHERE u.status = 'normal' AND u.deleted_at IS NULL AND ST_Intersects(u.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
            LIMIT $5
            "#,
            west, south, east, north, limit as i64
        ).fetch_all(&self.pool).await?;
        Ok(rows)
    }

    pub async fn search_users_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        self.search_users(west, south, east, north, limit).await
    }

    /// 聚类查询群组
    pub async fn cluster_groups_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as!(
            MapClusterRow,
            r#"
            WITH env AS (SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox),
            snapped AS (
              SELECT ST_SnapToGrid(ST_Transform(g.location::geometry, 3857), $5) AS p,
                     g.avatar_url
              FROM groups g, env
              WHERE g.dismissed_at IS NULL AND g.location IS NOT NULL
                AND ST_Intersects(g.location, env.bbox::geography)
            )
            SELECT 
              CONCAT(FLOOR(ST_X(p)/$5), ':', FLOOR(ST_Y(p)/$5)) as "key!",
              ST_Y(ST_Transform(p, 4326)) as "latitude!",
              ST_X(ST_Transform(p, 4326)) as "longitude!",
              COUNT(*) as "count!",
              COALESCE(ARRAY_REMOVE(ARRAY_AGG(avatar_url), NULL), ARRAY[]::text[]) as "preview_urls!"
            FROM snapped GROUP BY p
            "#,
            west,
            south,
            east,
            north,
            grid_meters
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 搜索矩形区域内的群组
    pub async fn search_groups_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as!(
            MapEntityRow,
            r#"
            SELECT 
                g.group_id::text as "id!",
                g.name as "title!",
                g.description as subtitle,
                g.avatar_url as cover_url,
                ST_Y(g.location::geometry) as "latitude!",
                ST_X(g.location::geometry) as "longitude!"
            FROM groups g
            WHERE g.dismissed_at IS NULL
              AND g.location IS NOT NULL
              AND ST_Intersects(g.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
            LIMIT $5
            "#,
            west,
            south,
            east,
            north,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 聚类查询瞬间
    pub async fn cluster_moments_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as!(
            MapClusterRow,
            r#"
            WITH env AS (SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox),
            snapped AS (
              SELECT ST_SnapToGrid(ST_Transform(m.location::geometry, 3857), $5) AS p,
                     m.moment_id,
                     mi.url as preview_url
              FROM moments m
              LEFT JOIN LATERAL (
                SELECT url
                FROM moment_images
                WHERE moment_id = m.moment_id
                ORDER BY sort_order ASC, created_at ASC
                LIMIT 1
              ) mi ON TRUE
              WHERE m.deleted_at IS NULL
                AND m.location IS NOT NULL
                AND ST_Intersects(m.location, (SELECT bbox FROM env)::geography)
                AND ($6::text IS NULL OR m.created_at >= ($6::text)::timestamptz)
                AND ($7::text IS NULL OR m.created_at <= ($7::text)::timestamptz)
            )
            SELECT 
              CONCAT(FLOOR(ST_X(p)/$5), ':', FLOOR(ST_Y(p)/$5)) as "key!",
              ST_Y(ST_Transform(p, 4326)) as "latitude!",
              ST_X(ST_Transform(p, 4326)) as "longitude!",
              COUNT(DISTINCT moment_id) as "count!",
              COALESCE(ARRAY_REMOVE(ARRAY_AGG(preview_url), NULL), ARRAY[]::text[]) as "preview_urls!"
            FROM snapped GROUP BY p
            "#,
            west,
            south,
            east,
            north,
            grid_meters,
            start_time,
            end_time
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 搜索矩形区域内的瞬间
    pub async fn search_moments_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as!(
            MapEntityRow,
            r#"
            SELECT 
                m.moment_id::text as "id!",
                COALESCE(m.description, '') as "title!",
                m.location_name as subtitle,
                mi.url as cover_url,
                ST_Y(m.location::geometry) as "latitude!",
                ST_X(m.location::geometry) as "longitude!"
            FROM moments m
            LEFT JOIN LATERAL (
                SELECT url
                FROM moment_images
                WHERE moment_id = m.moment_id
                ORDER BY sort_order ASC, created_at ASC
                LIMIT 1
            ) mi ON TRUE
            WHERE m.deleted_at IS NULL
              AND m.location IS NOT NULL
              AND ST_Intersects(m.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
              AND ($6::text IS NULL OR m.created_at >= ($6::text)::timestamptz)
              AND ($7::text IS NULL OR m.created_at <= ($7::text)::timestamptz)
            LIMIT $5
            "#,
            west,
            south,
            east,
            north,
            limit as i64,
            start_time,
            end_time
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 仅好友用户聚类
    pub async fn cluster_friend_users_by_bounds(
        &self,
        user_id: i64,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as!(
            MapClusterRow,
            r#"
            WITH env AS (SELECT ST_MakeEnvelope($2, $3, $4, $5, 4326) AS bbox),
            snapped AS (
              SELECT ST_SnapToGrid(ST_Transform(u.location::geometry, 3857), $6) AS p,
                     u.avatar_url
              FROM users u, env
              WHERE u.status = 'normal' AND u.deleted_at IS NULL
                AND u.location IS NOT NULL
                AND ST_Intersects(u.location, env.bbox::geography)
                AND (
                    u.user_id = $1
                    OR (
                        EXISTS (SELECT 1 FROM follows f1 WHERE f1.follower_id = $1 AND f1.following_id = u.user_id)
                        AND EXISTS (SELECT 1 FROM follows f2 WHERE f2.follower_id = u.user_id AND f2.following_id = $1)
                    )
                )
            )
            SELECT 
              CONCAT(FLOOR(ST_X(p)/$6), ':', FLOOR(ST_Y(p)/$6)) as "key!",
              ST_Y(ST_Transform(p, 4326)) as "latitude!",
              ST_X(ST_Transform(p, 4326)) as "longitude!",
              COUNT(*) as "count!",
              COALESCE(ARRAY_REMOVE(ARRAY_AGG(avatar_url), NULL), ARRAY[]::text[]) as "preview_urls!"
            FROM snapped GROUP BY p
            "#,
            user_id,
            west,
            south,
            east,
            north,
            grid_meters
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 仅好友用户搜索
    pub async fn search_friend_users_by_bounds(
        &self,
        user_id: i64,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as!(
            MapEntityRow,
            r#"
            SELECT 
                u.user_id::text as "id!",
                COALESCE(u.nickname, '') as "title!",
                u.bio as subtitle,
                u.avatar_url as cover_url,
                ST_Y(u.location::geometry) as "latitude!",
                ST_X(u.location::geometry) as "longitude!"
            FROM users u
            WHERE u.status = 'normal' AND u.deleted_at IS NULL
              AND u.location IS NOT NULL
              AND ST_Intersects(u.location, ST_MakeEnvelope($2, $3, $4, $5, 4326)::geography)
              AND (
                  u.user_id = $1
                  OR (
                      EXISTS (SELECT 1 FROM follows f1 WHERE f1.follower_id = $1 AND f1.following_id = u.user_id)
                      AND EXISTS (SELECT 1 FROM follows f2 WHERE f2.follower_id = u.user_id AND f2.following_id = $1)
                  )
              )
            LIMIT $6
            "#,
            user_id,
            west,
            south,
            east,
            north,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 仅好友瞬间聚类
    pub async fn cluster_friend_moments_by_bounds(
        &self,
        user_id: i64,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as!(
            MapClusterRow,
            r#"
            WITH env AS (SELECT ST_MakeEnvelope($2, $3, $4, $5, 4326) AS bbox),
            snapped AS (
              SELECT ST_SnapToGrid(ST_Transform(m.location::geometry, 3857), $6) AS p,
                     m.moment_id,
                     mi.url as preview_url
              FROM moments m
              LEFT JOIN LATERAL (
                SELECT url
                FROM moment_images
                WHERE moment_id = m.moment_id
                ORDER BY sort_order ASC, created_at ASC
                LIMIT 1
              ) mi ON TRUE
              WHERE m.deleted_at IS NULL
                AND m.location IS NOT NULL
                AND ST_Intersects(m.location, (SELECT bbox FROM env)::geography)
                AND (
                    m.user_id = $1
                    OR (
                        EXISTS (SELECT 1 FROM follows f1 WHERE f1.follower_id = $1 AND f1.following_id = m.user_id)
                        AND EXISTS (SELECT 1 FROM follows f2 WHERE f2.follower_id = m.user_id AND f2.following_id = $1)
                    )
                )
                AND ($7::text IS NULL OR m.created_at >= ($7::text)::timestamptz)
                AND ($8::text IS NULL OR m.created_at <= ($8::text)::timestamptz)
            )
            SELECT 
              CONCAT(FLOOR(ST_X(p)/$6), ':', FLOOR(ST_Y(p)/$6)) as "key!",
              ST_Y(ST_Transform(p, 4326)) as "latitude!",
              ST_X(ST_Transform(p, 4326)) as "longitude!",
              COUNT(DISTINCT moment_id) as "count!",
              COALESCE(ARRAY_REMOVE(ARRAY_AGG(preview_url), NULL), ARRAY[]::text[]) as "preview_urls!"
            FROM snapped GROUP BY p
            "#,
            user_id,
            west,
            south,
            east,
            north,
            grid_meters,
            start_time,
            end_time
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 仅好友瞬间搜索
    pub async fn search_friend_moments_by_bounds(
        &self,
        user_id: i64,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as!(
            MapEntityRow,
            r#"
            SELECT 
                m.moment_id::text as "id!",
                COALESCE(m.description, '') as "title!",
                m.location_name as subtitle,
                mi.url as cover_url,
                ST_Y(m.location::geometry) as "latitude!",
                ST_X(m.location::geometry) as "longitude!"
            FROM moments m
            LEFT JOIN LATERAL (
                SELECT url
                FROM moment_images
                WHERE moment_id = m.moment_id
                ORDER BY sort_order ASC, created_at ASC
                LIMIT 1
            ) mi ON TRUE
            WHERE m.deleted_at IS NULL
              AND m.location IS NOT NULL
              AND ST_Intersects(m.location, ST_MakeEnvelope($2, $3, $4, $5, 4326)::geography)
              AND (
                  m.user_id = $1
                  OR (
                      EXISTS (SELECT 1 FROM follows f1 WHERE f1.follower_id = $1 AND f1.following_id = m.user_id)
                      AND EXISTS (SELECT 1 FROM follows f2 WHERE f2.follower_id = m.user_id AND f2.following_id = $1)
                  )
              )
              AND ($7::text IS NULL OR m.created_at >= ($7::text)::timestamptz)
              AND ($8::text IS NULL OR m.created_at <= ($8::text)::timestamptz)
            LIMIT $6
            "#,
            user_id,
            west,
            south,
            east,
            north,
            limit as i64,
            start_time,
            end_time
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }
}
