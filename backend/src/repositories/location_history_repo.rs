use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    constants::LOCATION_CHANGE_THRESHOLD,
    error::Result,
    models::location_history::{GroupLocationHistory, UserLocationHistory},
    utils::geo::calculate_distance,
};

pub struct LocationHistoryRepository {
    db: PgPool,
}

impl LocationHistoryRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 记录用户位置（如果变化超过阈值）
    pub async fn record_user_location_if_changed(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> Result<bool> {
        // 获取最后一次位置
        let last_location = sqlx::query_as!(
            UserLocationHistory,
            "SELECT id, user_id, latitude, longitude, location_name, created_at
             FROM user_location_history
             WHERE user_id = $1
             ORDER BY created_at DESC
             LIMIT 1",
            user_id
        )
        .fetch_optional(&self.db)
        .await?;

        let should_record = if let Some(last) = last_location {
            let distance = calculate_distance(last.latitude, last.longitude, latitude, longitude);
            distance >= LOCATION_CHANGE_THRESHOLD
        } else {
            true // 首次记录
        };

        if should_record {
            sqlx::query!(
                "INSERT INTO user_location_history (user_id, latitude, longitude, location_name)
                 VALUES ($1, $2, $3, $4)",
                user_id,
                latitude,
                longitude,
                location_name
            )
            .execute(&self.db)
            .await?;
        }

        Ok(should_record)
    }

    /// 记录群组位置（如果变化超过阈值）
    pub async fn record_group_location_if_changed(
        &self,
        group_id: Uuid,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> Result<bool> {
        // 获取最后一次位置
        let last_location = sqlx::query_as!(
            GroupLocationHistory,
            "SELECT id, group_id, latitude, longitude, location_name, created_at
             FROM group_location_history
             WHERE group_id = $1
             ORDER BY created_at DESC
             LIMIT 1",
            group_id
        )
        .fetch_optional(&self.db)
        .await?;

        let should_record = if let Some(last) = last_location {
            let distance = calculate_distance(last.latitude, last.longitude, latitude, longitude);
            distance >= LOCATION_CHANGE_THRESHOLD
        } else {
            true // 首次记录
        };

        if should_record {
            sqlx::query!(
                "INSERT INTO group_location_history (group_id, latitude, longitude, location_name)
                 VALUES ($1, $2, $3, $4)",
                group_id,
                latitude,
                longitude,
                location_name
            )
            .execute(&self.db)
            .await?;
        }

        Ok(should_record)
    }

    /// 获取用户位置历史
    pub async fn get_user_location_history(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<UserLocationHistory>> {
        let history = sqlx::query_as!(
            UserLocationHistory,
            "SELECT id, user_id, latitude, longitude, location_name, created_at
             FROM user_location_history
             WHERE user_id = $1
             ORDER BY created_at DESC
             LIMIT $2 OFFSET $3",
            user_id,
            limit,
            offset
        )
        .fetch_all(&self.db)
        .await?;

        Ok(history)
    }

    /// 获取群组位置历史
    pub async fn get_group_location_history(
        &self,
        group_id: Uuid,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<GroupLocationHistory>> {
        let history = sqlx::query_as!(
            GroupLocationHistory,
            "SELECT id, group_id, latitude, longitude, location_name, created_at
             FROM group_location_history
             WHERE group_id = $1
             ORDER BY created_at DESC
             LIMIT $2 OFFSET $3",
            group_id,
            limit,
            offset
        )
        .fetch_all(&self.db)
        .await?;

        Ok(history)
    }
}











