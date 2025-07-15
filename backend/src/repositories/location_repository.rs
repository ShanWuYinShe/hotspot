//! 位置相关数据仓库实现

use sqlx::PgPool;
use chrono::DateTime;

use crate::repositories::{Repository, RepositoryError, RepositoryResult};

/// PostgreSQL实现的位置仓库
#[derive(Debug)]
pub struct PgLocationRepository {
    /// 数据库连接池
    pool: PgPool,
}

impl PgLocationRepository {
    /// 创建新的PostgreSQL位置仓库
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 保存用户位置
    #[allow(dead_code)]
    pub async fn save_user_location(&self, user_id: i64, latitude: f64, longitude: f64, timestamp: i64) -> RepositoryResult<i64> {
        let dt = DateTime::from_timestamp(timestamp, 0)
            .ok_or_else(|| RepositoryError::InvalidArgument("无效的时间戳".to_string()))?;

        let result = sqlx::query!(
            r#"
            INSERT INTO user_locations (user_id, latitude, longitude, updated_at)
            VALUES ($1, $2, $3, $4)
            RETURNING id
            "#,
            user_id,
            latitude,
            longitude,
            dt
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(result.id)
    }
    
    /// 保存签到信息
    #[allow(dead_code)]
    pub async fn save_checkin(&self, user_id: i64, latitude: f64, longitude: f64, description: &str, timestamp: i64) -> RepositoryResult<i64> {
        let dt = DateTime::from_timestamp(timestamp, 0)
            .ok_or_else(|| RepositoryError::InvalidArgument("无效的时间戳".to_string()))?;

        let result = sqlx::query!(
            r#"
            INSERT INTO checkins (user_id, latitude, longitude, description, created_at)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id
            "#,
            user_id,
            latitude,
            longitude,
            description,
            dt
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(result.id)
    }
    
    /// 查找附近的签到
    #[allow(dead_code)]
    pub async fn find_nearby_checkins(&self, latitude: f64, longitude: f64, radius_meters: f64, limit: i32) -> RepositoryResult<Vec<(i64, i64, String, f64, f64, String, i64)>> {
        let results = sqlx::query!(
            r#"
            SELECT c.id, c.user_id, c.description, c.latitude, c.longitude, u.nickname,
                   EXTRACT(EPOCH FROM c.created_at)::bigint as timestamp
            FROM checkins c
            JOIN users u ON c.user_id = u.user_id
            WHERE ST_DWithin(
                ST_MakePoint(c.longitude, c.latitude)::geography,
                ST_MakePoint($1, $2)::geography,
                $3
            )
            ORDER BY ST_Distance(
                ST_MakePoint(c.longitude, c.latitude)::geography,
                ST_MakePoint($1, $2)::geography
            )
            LIMIT $4
            "#,
            longitude,
            latitude,
            radius_meters,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        let checkins = results
            .into_iter()
            .map(|row| {
                (
                    row.id,
                    row.user_id,
                    row.description.unwrap_or_default(),
                    row.latitude,
                    row.longitude,
                    row.nickname,
                    row.timestamp.unwrap_or(0),
                )
            })
            .collect();

        Ok(checkins)
    }
    
    /// 查找附近的用户
    #[allow(dead_code)]
    pub async fn find_nearby_users(&self, latitude: f64, longitude: f64, radius_meters: f64, exclude_user_id: i64, limit: i32) -> RepositoryResult<Vec<(i64, String, f64, f64)>> {
        let results = sqlx::query!(
            r#"
            SELECT ul.user_id, u.nickname, ul.latitude, ul.longitude
            FROM user_locations ul
            JOIN users u ON ul.user_id = u.user_id
            WHERE ul.user_id != $1
            AND ST_DWithin(
                ST_MakePoint(ul.longitude, ul.latitude)::geography,
                ST_MakePoint($2, $3)::geography,
                $4
            )
            ORDER BY ST_Distance(
                ST_MakePoint(ul.longitude, ul.latitude)::geography,
                ST_MakePoint($2, $3)::geography
            )
            LIMIT $5
            "#,
            exclude_user_id,
            longitude,
            latitude,
            radius_meters,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        let users = results
            .into_iter()
            .map(|row| (row.user_id, row.nickname, row.latitude, row.longitude))
            .collect();

        Ok(users)
    }
}

impl Repository for PgLocationRepository {
    fn name(&self) -> &str {
        "PgLocationRepository"
    }
}
