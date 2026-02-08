//! 用户在线状态仓库
//!
//! 管理用户的在线状态记录（数据库持久化）

use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use crate::error::Result;

#[derive(Debug, FromRow)]
pub struct UserOnlineStatusRow {
    pub user_id: i64,
    pub is_online: bool,
    pub last_active_at: Option<DateTime<Utc>>,
}

#[derive(Clone)]
pub struct UserOnlineStatusRepository {
    pool: PgPool,
}

impl UserOnlineStatusRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取用户在线状态
    pub async fn get_status(&self, user_id: i64) -> Result<Option<UserOnlineStatusRow>> {
        let row = sqlx::query_as_unchecked!(
            UserOnlineStatusRow,
            "SELECT user_id, is_online, last_active_at FROM user_online_status WHERE user_id = $1",
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;
        Ok(row)
    }

    pub async fn get_batch_status(&self, user_ids: &[i64]) -> Result<Vec<UserOnlineStatusRow>> {
        let rows = sqlx::query_as_unchecked!(
            UserOnlineStatusRow,
            "SELECT user_id, is_online, last_active_at FROM user_online_status WHERE user_id = ANY($1)",
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 设置用户在线
    pub async fn set_online(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_online_status (user_id, is_online, last_active_at, connection_count)
            VALUES ($1, TRUE, NOW(), 1)
            ON CONFLICT (user_id)
            DO UPDATE SET
                is_online = TRUE,
                last_active_at = NOW(),
                connection_count = user_online_status.connection_count + 1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 设置用户离线
    pub async fn set_offline(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET 
                connection_count = GREATEST(connection_count - 1, 0),
                is_online = CASE WHEN connection_count - 1 <= 0 THEN FALSE ELSE TRUE END,
                last_active_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
