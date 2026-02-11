//! 用户在线状态仓库层实现
//!
//! 适配 user_online_status 表：user_id, is_online, last_active_at 等

use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};

use crate::error::Result;

/// 在线状态行数据
#[derive(Debug, Clone, FromRow)]
pub struct UserOnlineStatusRow {
    pub user_id: i64,
    pub is_online: bool,
    pub last_active_at: DateTime<Utc>,
    pub connection_count: i32,
}

#[derive(Debug, Clone)]
pub struct UserOnlineStatusRepository {
    pool: PgPool,
}

impl UserOnlineStatusRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取单个用户在线状态
    pub async fn get_status(&self, user_id: i64) -> Result<Option<UserOnlineStatusRow>> {
        let row = sqlx::query_as!(
            UserOnlineStatusRow,
            r#"
            SELECT user_id as "user_id!", is_online as "is_online!", last_active_at as "last_active_at!",
                   connection_count as "connection_count!"
            FROM user_online_status
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 批量获取用户在线状态
    pub async fn get_batch_status(
        &self,
        user_ids: &[i64],
    ) -> Result<Vec<UserOnlineStatusRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }

        let rows = sqlx::query_as!(
            UserOnlineStatusRow,
            r#"
            SELECT user_id as "user_id!", is_online as "is_online!", last_active_at as "last_active_at!",
                   connection_count as "connection_count!"
            FROM user_online_status
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 更新或插入在线状态
    pub async fn upsert_status(
        &self,
        user_id: i64,
        is_online: bool,
        connection_count: i32,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_online_status (user_id, is_online, last_active_at, connection_count)
            VALUES ($1, $2, NOW(), $3)
            ON CONFLICT (user_id) DO UPDATE SET
                is_online = $2,
                last_active_at = NOW(),
                connection_count = $3
            "#,
            user_id,
            is_online,
            connection_count
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 标记用户上线
    pub async fn set_online(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_online_status (user_id, is_online, last_active_at, connection_count)
            VALUES ($1, TRUE, NOW(), 1)
            ON CONFLICT (user_id) DO UPDATE SET
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

    /// 标记用户离线
    pub async fn set_offline(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET is_online = FALSE,
                last_active_at = NOW(),
                connection_count = GREATEST(0, connection_count - 1)
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
