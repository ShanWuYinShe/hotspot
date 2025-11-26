use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

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
    pub async fn get_latest_batch(&self, user_ids: &[i64]) -> Result<std::collections::HashMap<i64, UserLocationHistoryRow>> {
        if user_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        // 使用 DISTINCT ON 获取每个用户的最新位置
        let rows = sqlx::query_as!(
            UserLocationHistoryRow,
            r#"
            SELECT DISTINCT ON (user_id) id, user_id, latitude, longitude, location_name, recorded_at
            FROM user_location_history
            WHERE user_id = ANY($1)
            ORDER BY user_id, recorded_at DESC
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.user_id, r)).collect())
    }
}
