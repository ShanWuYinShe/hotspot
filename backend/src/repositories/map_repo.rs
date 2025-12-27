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

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }
}

#[derive(Debug, FromRow)]
pub struct MapClusterRow {
    pub key: String,
    pub latitude: f64,
    pub longitude: f64,
    pub count: i64,
}

impl MapRepository {
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
                AND (u.location::geometry && env.bbox)
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
                AND (g.location::geometry && env.bbox)
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
                AND (m.location::geometry && env.bbox)
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
}

