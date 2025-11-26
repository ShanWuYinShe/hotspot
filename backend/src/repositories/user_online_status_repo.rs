use chrono::{DateTime, Utc};

use sqlx::PgPool;

use crate::error::Result;

pub struct UserOnlineStatusRepository {
    pool: PgPool,
}

impl UserOnlineStatusRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取用户在线状态
    pub async fn get_status(&self, user_id: i64) -> Result<Option<UserOnlineStatusRow>> {
        let status = sqlx::query_as!(
            UserOnlineStatusRow,
            r#"
            SELECT user_id, is_online, last_active_at, last_seen_at, connection_count, updated_at
            FROM user_online_status
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(status)
    }


    /// 批量获取用户在线状态
    pub async fn get_batch_status(&self, user_ids: &[i64]) -> Result<Vec<UserOnlineStatusRow>> {
        let statuses = sqlx::query_as!(
            UserOnlineStatusRow,
            r#"
            SELECT user_id, is_online, last_active_at, last_seen_at, connection_count, updated_at
            FROM user_online_status
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(statuses)
    }

    /// 设置用户在线
    pub async fn set_online(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_online_status (user_id, is_online, last_active_at, connection_count, updated_at)
            VALUES ($1, TRUE, NOW(), 1, NOW())
            ON CONFLICT (user_id)
            DO UPDATE SET
                is_online = TRUE,
                last_active_at = NOW(),
                connection_count = user_online_status.connection_count + 1,
                updated_at = NOW()
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 设置用户离线（减少连接计数，当连接数为0时标记为离线）
    pub async fn set_offline(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET 
                connection_count = GREATEST(connection_count - 1, 0),
                is_online = CASE WHEN connection_count - 1 <= 0 THEN FALSE ELSE TRUE END,
                last_seen_at = CASE WHEN connection_count - 1 <= 0 THEN NOW() ELSE last_seen_at END,
                updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新最后活跃时间
    pub async fn update_last_active(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET last_active_at = NOW(), updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取在线用户数量
    pub async fn count_online_users(&self) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_online_status
            WHERE is_online = TRUE
            "#
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 清理长时间未活跃的在线状态（定期任务用）
    pub async fn cleanup_stale_online_status(&self, timeout_minutes: i32) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            UPDATE user_online_status
            SET 
                is_online = FALSE,
                last_seen_at = last_active_at,
                updated_at = NOW()
            WHERE is_online = TRUE
              AND last_active_at < NOW() - INTERVAL '1 minute' * $1
            "#,
            timeout_minutes as i64
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() as i64)
    }

    /// 批量获取用户在线状态
    pub async fn batch_get_status(
        &self,
        user_ids: &[i64],
    ) -> Result<Vec<(i64, UserOnlineStatusRow)>> {
        let rows = sqlx::query_as!(
            UserOnlineStatusRow,
            r#"
            SELECT user_id, is_online, last_active_at, last_seen_at, connection_count, updated_at
            FROM user_online_status
            WHERE user_id = ANY($1)
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        // 将结果转换为Vec<(i64, UserOnlineStatusRow)>格式
        let result = rows.into_iter().map(|row| (row.user_id, row)).collect();

        Ok(result)
    }
}

// ==================== 私有查询结果类型 ====================

/// 用户在线状态行
#[derive(Debug)]
#[allow(dead_code)]
pub struct UserOnlineStatusRow {
    pub user_id: i64,
    pub is_online: bool,
    pub last_active_at: Option<DateTime<Utc>>,
    pub last_seen_at: Option<DateTime<Utc>>,
    pub connection_count: i32,
    pub updated_at: DateTime<Utc>,
}
