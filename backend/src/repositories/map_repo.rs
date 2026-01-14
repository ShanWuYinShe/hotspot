use sqlx::{FromRow, PgPool};

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct MapRepository {
    pool: PgPool,
}

impl MapRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[derive(Debug, FromRow)]
pub struct MapClusterRow {
    pub key: String,
    pub latitude: f64,
    pub longitude: f64,
    pub count: i64,
}

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
    // ==================== Cluster Queries ====================

    pub async fn cluster_users_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as::<_, MapClusterRow>(
            r#"
            WITH env AS (
              SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox
            ),
            snapped AS (
              SELECT
                ST_SnapToGrid(ST_Transform(u.location::geometry, 3857), $5) AS p
              FROM users u, env
              WHERE u.is_deleted = FALSE
                AND u.is_banned = FALSE
                AND u.location IS NOT NULL
                AND ST_Intersects(u.location, env.bbox::geography)
            )
            SELECT
              CONCAT(
                FLOOR(ST_X(p) / $5)::bigint,
                ':',
                FLOOR(ST_Y(p) / $5)::bigint,
                ':',
                $5::bigint
              ) AS key,
              ST_Y(ST_Transform(p, 4326)) AS latitude,
              ST_X(ST_Transform(p, 4326)) AS longitude,
              COUNT(*)::bigint AS count
            FROM snapped
            GROUP BY p
            ORDER BY count DESC
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(grid_meters)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn cluster_groups_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        grid_meters: f64,
    ) -> Result<Vec<MapClusterRow>> {
        let rows = sqlx::query_as::<_, MapClusterRow>(
            r#"
            WITH env AS (
              SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox
            ),
            snapped AS (
              SELECT
                ST_SnapToGrid(ST_Transform(g.location::geometry, 3857), $5) AS p
              FROM group_chats g, env
              WHERE g.is_deleted = FALSE
                AND g.is_dismissed = FALSE
                AND g.location IS NOT NULL
                AND ST_Intersects(g.location, env.bbox::geography)
            )
            SELECT
              CONCAT(
                FLOOR(ST_X(p) / $5)::bigint,
                ':',
                FLOOR(ST_Y(p) / $5)::bigint,
                ':',
                $5::bigint
              ) AS key,
              ST_Y(ST_Transform(p, 4326)) AS latitude,
              ST_X(ST_Transform(p, 4326)) AS longitude,
              COUNT(*)::bigint AS count
            FROM snapped
            GROUP BY p
            ORDER BY count DESC
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(grid_meters)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

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
        let rows = sqlx::query_as::<_, MapClusterRow>(
            r#"
            WITH env AS (
              SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox
            ),
            snapped AS (
              SELECT
                ST_SnapToGrid(ST_Transform(m.location::geometry, 3857), $5) AS p
              FROM moments m, env
              WHERE m.is_deleted = FALSE
                AND m.location IS NOT NULL
                AND ST_Intersects(m.location, env.bbox::geography)
                AND ($6::TIMESTAMPTZ IS NULL OR m.created_at >= $6::TIMESTAMPTZ)
                AND ($7::TIMESTAMPTZ IS NULL OR m.created_at <= $7::TIMESTAMPTZ)
            )
            SELECT
              CONCAT(
                FLOOR(ST_X(p) / $5)::bigint,
                ':',
                FLOOR(ST_Y(p) / $5)::bigint,
                ':',
                $5::bigint
              ) AS key,
              ST_Y(ST_Transform(p, 4326)) AS latitude,
              ST_X(ST_Transform(p, 4326)) AS longitude,
              COUNT(*)::bigint AS count
            FROM snapped
            GROUP BY p
            ORDER BY count DESC
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(grid_meters)
        .bind(start_time)
        .bind(end_time)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    // ==================== Entity Queries ====================

    pub async fn search_users_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as::<_, MapEntityRow>(
            r#"
            SELECT
                u.user_id::text AS id,
                u.nickname AS title,
                u.bio AS subtitle,
                u.avatar_url AS cover_url,
                ST_Y(u.location::geometry) AS latitude,
                ST_X(u.location::geometry) AS longitude
            FROM users u
            WHERE u.is_deleted = FALSE
                AND u.is_banned = FALSE
                AND u.location IS NOT NULL
                AND ST_Intersects(u.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
            LIMIT $5
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(limit)
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    pub async fn search_groups_by_bounds(
        &self,
        west: f64,
        south: f64,
        east: f64,
        north: f64,
        limit: i32,
    ) -> Result<Vec<MapEntityRow>> {
        let rows = sqlx::query_as::<_, MapEntityRow>(
            r#"
            SELECT
                g.group_chat_id::text AS id,
                g.name AS title,
                g.description AS subtitle,
                g.avatar_url AS cover_url,
                ST_Y(g.location::geometry) AS latitude,
                ST_X(g.location::geometry) AS longitude
            FROM group_chats g
            WHERE g.is_deleted = FALSE
                AND g.is_dismissed = FALSE
                AND g.location IS NOT NULL
                AND ST_Intersects(g.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
            LIMIT $5
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(limit)
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

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
        // Moments 需要 join user 获取 nickname?
        // 简单起见，title=desc, subtitle=user_id or nickname if joined.
        // MapEntityRow struct fields match.
        // cover_url: images[1] (Postgres 1-based array index)
        let rows = sqlx::query_as::<_, MapEntityRow>(
            r#"
            SELECT
                m.moment_id::text AS id,
                m.description AS title,
                u.nickname AS subtitle,
                m.images[1] AS cover_url,
                ST_Y(m.location::geometry) AS latitude,
                ST_X(m.location::geometry) AS longitude
            FROM moments m
            JOIN users u ON m.user_id = u.user_id
            WHERE m.is_deleted = FALSE
                AND m.location IS NOT NULL
                AND ST_Intersects(m.location, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
                AND ($6::TIMESTAMPTZ IS NULL OR m.created_at >= $6::TIMESTAMPTZ)
                AND ($7::TIMESTAMPTZ IS NULL OR m.created_at <= $7::TIMESTAMPTZ)
            LIMIT $5
            "#,
        )
        .bind(west)
        .bind(south)
        .bind(east)
        .bind(north)
        .bind(limit)
        .bind(start_time)
        .bind(end_time)
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }
}
