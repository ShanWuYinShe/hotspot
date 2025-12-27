use chrono::{DateTime, Utc};
use sqlx::PgPool;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

#[derive(Debug, Clone)]
pub struct UserBlocksRepository {
    pool: PgPool,
}

impl UserBlocksRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(
        &self,
        blocker_id: i64,
        blocked_id: i64,
        reason: Option<String>,
    ) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            INSERT INTO user_blocks (blocker_id, blocked_id, reason)
            VALUES ($1, $2, $3)
            ON CONFLICT (blocker_id, blocked_id) DO NOTHING
            "#,
            blocker_id,
            blocked_id,
            reason
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    pub async fn delete(&self, blocker_id: i64, blocked_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM user_blocks
            WHERE blocker_id = $1 AND blocked_id = $2
            "#,
            blocker_id,
            blocked_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn exists(&self, blocker_id: i64, blocked_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM user_blocks
                WHERE blocker_id = $1 AND blocked_id = $2
            ) as "exists!"
            "#,
            blocker_id,
            blocked_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 检查 blocker_id 是否拉黑了 blocked_id
    pub async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> Result<bool> {
        self.exists(blocker_id, blocked_id).await
    }

    /// 获取拉黑列表（包含详细信息）
    pub async fn list_blocked_detail(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<BlockedDetailRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows: Vec<BlockedDetailRow> = sqlx::query_as!(
            BlockedDetailRow,
            r#"
            SELECT id, blocked_id, reason, created_at
            FROM user_blocks
            WHERE blocker_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|last| encode_cursor(&last.id))
            .transpose()?;
        Ok((rows, next_cursor))
    }
}

/// 拉黑列表查询结果（包含详细信息）
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct BlockedDetailRow {
    pub id: i64,
    pub blocked_id: i64,
    pub reason: Option<String>,
    pub created_at: DateTime<Utc>,
}
