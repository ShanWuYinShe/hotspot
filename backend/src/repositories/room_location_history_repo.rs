use crate::error::{AppError, Result};
use crate::models::room_location_history::RoomLocationHistory;
use sqlx::PgPool;
use uuid::Uuid;

pub struct RoomLocationHistoryRepository {
    db: PgPool,
}

impl RoomLocationHistoryRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 插入新的位置记录
    pub async fn create(
        &self,
        room_id: Uuid,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> Result<RoomLocationHistory> {
        let record = sqlx::query_as!(
            RoomLocationHistory,
            r#"
            INSERT INTO room_location_history (room_id, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4)
            RETURNING id, room_id, latitude, longitude, location_name, created_at
            "#,
            room_id,
            latitude,
            longitude,
            location_name
        )
        .fetch_one(&self.db)
        .await
        .map_err(AppError::from)?;

        Ok(record)
    }

    /// 获取聊天室最新的位置记录
    pub async fn get_latest_by_room_id(
        &self,
        room_id: Uuid,
    ) -> Result<Option<RoomLocationHistory>> {
        let record = sqlx::query_as!(
            RoomLocationHistory,
            r#"
            SELECT id, room_id, latitude, longitude, location_name, created_at
            FROM room_location_history
            WHERE room_id = $1
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            room_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(AppError::from)?;

        Ok(record)
    }

    /// 获取聊天室的位置历史列表（分页）
    pub async fn list_by_room_id(
        &self,
        room_id: Uuid,
        limit: i64,
        before_id: Option<i64>,
    ) -> Result<Vec<RoomLocationHistory>> {
        let records = if let Some(before_id) = before_id {
            sqlx::query_as!(
                RoomLocationHistory,
                r#"
                SELECT id, room_id, latitude, longitude, location_name, created_at
                FROM room_location_history
                WHERE room_id = $1 AND id < $2
                ORDER BY id DESC
                LIMIT $3
                "#,
                room_id,
                before_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(AppError::from)?
        } else {
            sqlx::query_as!(
                RoomLocationHistory,
                r#"
                SELECT id, room_id, latitude, longitude, location_name, created_at
                FROM room_location_history
                WHERE room_id = $1
                ORDER BY id DESC
                LIMIT $2
                "#,
                room_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(AppError::from)?
        };

        Ok(records)
    }

    /// 按位置搜索附近的聊天室（返回room_id和距离）
    pub async fn search_nearby_rooms(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i64,
    ) -> Result<Vec<(Uuid, f64)>> {
        let records = sqlx::query_as::<_, (Uuid, f64)>(
            r#"
            WITH latest_locations AS (
                SELECT DISTINCT ON (room_id) 
                    room_id,
                    location
                FROM room_location_history
                ORDER BY room_id, created_at DESC
            )
            SELECT 
                room_id,
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


