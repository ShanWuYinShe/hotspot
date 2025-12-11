use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserBlockRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub blocker_id: i64,
    pub blocked_id: i64,
    pub reason: Option<String>,
    pub created_at: DateTime<Utc>,
}

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
    ) -> Result<UserBlockRow> {
        let row = sqlx::query_as!(
            UserBlockRow,
            r#"
            INSERT INTO user_blocks (blocker_id, blocked_id, reason)
            VALUES ($1, $2, $3)
            ON CONFLICT (blocker_id, blocked_id) DO NOTHING
            RETURNING id, cursor_id, blocker_id, blocked_id, reason, created_at
            "#,
            blocker_id,
            blocked_id,
            reason
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
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

    /// 解析 UUID cursor 为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM user_blocks WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 获取拉黑列表（包含详细信息）
    pub async fn list_blocked_detail(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<BlockedDetailRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<BlockedDetailRow> = sqlx::query_as!(
            BlockedDetailRow,
            r#"
            SELECT blocked_id, cursor_id, reason, created_at
            FROM user_blocks
            WHERE blocker_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((rows, next_cursor))
    }
}

/// 拉黑列表查询结果（包含详细信息）
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct BlockedDetailRow {
    pub blocked_id: i64,
    pub cursor_id: Uuid,
    pub reason: Option<String>,
    pub created_at: DateTime<Utc>,
}

