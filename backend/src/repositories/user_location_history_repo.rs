//! 用户位置历史仓库
//!
//! 记录用户历史位置变化

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;

/// 用户位置历史Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserLocationHistoryRow {
    pub id: i64,
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
    pub recorded_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct UserLocationHistoryRepository {
    pool: PgPool,
}

impl UserLocationHistoryRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 记录用户位置
    /// location 字段由数据库 trigger 自动从 latitude/longitude 生成
    pub async fn record(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> Result<UserLocationHistoryRow> {
        let row = sqlx::query_as!(
            UserLocationHistoryRow,
            r#"
            INSERT INTO user_location_history (user_id, latitude, longitude, location_name)
            VALUES ($1, $2, $3, $4)
            RETURNING id, user_id, latitude, longitude, location_name, recorded_at
            "#,
            user_id,
            latitude,
            longitude,
            location_name
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取用户最新位置
    pub async fn get_latest(&self, user_id: i64) -> Result<Option<UserLocationHistoryRow>> {
        let row = sqlx::query_as!(
            UserLocationHistoryRow,
            r#"
            SELECT id, user_id, latitude, longitude, location_name, recorded_at
            FROM user_location_history
            WHERE user_id = $1
            ORDER BY recorded_at DESC
            LIMIT 1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 批量获取用户最新位置
    /// 使用 PostGIS 计算用户最新位置与指定坐标的距离（米）
    /// 返回 None 表示没有位置记录
    pub async fn get_distance_from_latest(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
    ) -> Result<Option<f64>> {
        let row = sqlx::query_scalar!(
            r#"
            SELECT ST_Distance(
                COALESCE(location, ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography),
                ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
            ) as "distance!"
            FROM user_location_history
            WHERE user_id = $1
            ORDER BY recorded_at DESC
            LIMIT 1
            "#,
            user_id,
            longitude,
            latitude
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }
}
