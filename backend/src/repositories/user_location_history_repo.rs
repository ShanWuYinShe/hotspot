use crate::error::{AppError, Result};
use crate::models::user_location_history::UserLocationHistory;
use sqlx::PgPool;

pub struct UserLocationHistoryRepository {
    db: PgPool,
}

impl UserLocationHistoryRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 插入新的位置记录
    pub async fn create(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> Result<UserLocationHistory> {
        let record = sqlx::query_as!(
            UserLocationHistory,
            r#"
            INSERT INTO user_location_history (user_id, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4)
            RETURNING id, user_id, latitude, longitude, location_name, created_at
            "#,
            user_id,
            latitude,
            longitude,
            location_name
        )
        .fetch_one(&self.db)
        .await
        .map_err(AppError::from)?;

        Ok(record)
    }

    /// 获取用户最新的位置记录
    pub async fn get_latest_by_user_id(
        &self,
        user_id: i64,
    ) -> Result<Option<UserLocationHistory>> {
        let record = sqlx::query_as!(
            UserLocationHistory,
            r#"
            SELECT id, user_id, latitude, longitude, location_name, created_at
            FROM user_location_history
            WHERE user_id = $1
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            user_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(AppError::from)?;

        Ok(record)
    }

    /// 获取用户的位置历史列表（分页）
    pub async fn list_by_user_id(
        &self,
        user_id: i64,
        limit: i64,
        before_id: Option<i64>,
    ) -> Result<Vec<UserLocationHistory>> {
        let records = if let Some(before_id) = before_id {
            sqlx::query_as!(
                UserLocationHistory,
                r#"
                SELECT id, user_id, latitude, longitude, location_name, created_at
                FROM user_location_history
                WHERE user_id = $1 AND id < $2
                ORDER BY id DESC
                LIMIT $3
                "#,
                user_id,
                before_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(AppError::from)?
        } else {
            sqlx::query_as!(
                UserLocationHistory,
                r#"
                SELECT id, user_id, latitude, longitude, location_name, created_at
                FROM user_location_history
                WHERE user_id = $1
                ORDER BY id DESC
                LIMIT $2
                "#,
                user_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(AppError::from)?
        };

        Ok(records)
    }

    /// 按位置搜索附近的用户（返回用户ID和距离）
    pub async fn search_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i64,
    ) -> Result<Vec<(i64, f64)>> {
        let records = sqlx::query_as::<_, (i64, f64)>(
            r#"
            WITH latest_locations AS (
                SELECT DISTINCT ON (user_id) 
                    user_id,
                    location
                FROM user_location_history
                ORDER BY user_id, created_at DESC
            )
            SELECT 
                user_id,
                ST_Distance(
                    location,
                    ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                ) as distance
            FROM latest_locations
            WHERE ST_DWithin(
                location,
                ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                $3
            )
            ORDER BY distance
            LIMIT $4
            "#
        )
        .bind(latitude)
        .bind(longitude)
        .bind(radius_meters)
        .bind(limit)
        .fetch_all(&self.db)
        .await
        .map_err(AppError::from)?;

        Ok(records)
    }
}


