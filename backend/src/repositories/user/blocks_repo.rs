use std::collections::HashSet;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated, TimeCursor};
use chrono::{DateTime, Utc};
use sqlx::PgPool;

#[derive(Debug, Clone)]
pub struct UserBlocksRepository {
    pool: PgPool,
}

#[derive(Debug, Clone)]
pub struct BlockEdgeRow {
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}

impl UserBlocksRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn block(&self, blocker_id: i64, blocked_id: i64) -> Result<()> {
        sqlx::query!(
            "INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            blocker_id,
            blocked_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn unblock(&self, blocker_id: i64, blocked_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2",
            blocker_id,
            blocked_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)",
            blocker_id,
            blocked_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(row.unwrap_or(false))
    }

    /// 批量检查 blocker_id 是否拉黑了 blocked_ids 中的用户
    /// 返回被拉黑的 blocked_ids 子集
    pub async fn batch_check_blocked(
        &self,
        blocker_id: i64,
        blocked_ids: &[i64],
    ) -> Result<HashSet<i64>> {
        if blocked_ids.is_empty() {
            return Ok(HashSet::new());
        }
        let rows = sqlx::query_scalar!(
            "SELECT blocked_id as \"id!\" FROM blocks WHERE blocker_id = $1 AND blocked_id = ANY($2)",
            blocker_id,
            blocked_ids
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.into_iter().collect())
    }

    pub async fn list_blocked(
        &self,
        blocker_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<BlockEdgeRow> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT blocked_id as "user_id!", created_at as "created_at!"
            FROM blocks
            WHERE blocker_id = $1
              AND (created_at, blocked_id) < ($2, $3)
            ORDER BY created_at DESC, blocked_id DESC
            LIMIT $4
            "#,
            blocker_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<BlockEdgeRow> = rows
            .into_iter()
            .take(limit as usize)
            .map(|r| BlockEdgeRow {
                user_id: r.user_id,
                created_at: r.created_at,
            })
            .collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
