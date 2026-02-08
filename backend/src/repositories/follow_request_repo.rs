use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};
use crate::types::enums::RequestStatus;

/// 关注请求Row - 严格按照数据库 schema
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FollowRequestRow {
    pub request_id: Uuid,
    pub requester_id: i64,
    pub target_id: i64,
    pub status: RequestStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct FollowRequestRepository {
    pool: PgPool,
}

impl FollowRequestRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建关注请求
    pub async fn create(&self, requester_id: i64, target_id: i64) -> Result<FollowRequestRow> {
        let row = sqlx::query_as_unchecked!(
            FollowRequestRow,
            r#"
            INSERT INTO follow_requests (requester_id, target_id)
            VALUES ($1, $2)
            RETURNING request_id, requester_id, target_id, status as "status: RequestStatus", created_at, updated_at
            "#,
            requester_id,
            target_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找请求
    pub async fn find_by_id(&self, request_id: Uuid) -> Result<Option<FollowRequestRow>> {
        let row = sqlx::query_as_unchecked!(
            FollowRequestRow,
            r#"
            SELECT request_id, requester_id, target_id, status as "status: RequestStatus", created_at, updated_at
            FROM follow_requests
            WHERE request_id = $1
            "#,
            request_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查请求是否存在
    pub async fn exists(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM follow_requests
                WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            ) as "exists!"
            "#,
            requester_id,
            target_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 接受请求
    pub async fn accept(&self, request_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE follow_requests SET status = 'accepted', updated_at = NOW() WHERE request_id = $1",
            request_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 拒绝请求
    pub async fn reject(&self, request_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE follow_requests SET status = 'rejected', updated_at = NOW() WHERE request_id = $1",
            request_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 列出收到的请求
    pub async fn list_received(
        &self,
        target_id: i64,
        status: Option<RequestStatus>,
        limit: usize,
        cursor: Option<String>,
    ) -> Result<(Vec<FollowRequestRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query_as_unchecked!(
            FollowRequestRow,
            r#"
            SELECT request_id, requester_id, target_id, status as "status: RequestStatus", created_at, updated_at
            FROM follow_requests
            WHERE target_id = $1 
              AND ($2::text IS NULL OR status = $2::text::request_status)
              AND (created_at, requester_id) < ($3, $4)
            ORDER BY created_at DESC, requester_id DESC
            LIMIT $5
            "#,
            target_id,
            status.map(|s| s.to_string()),
            last_ts,
            last_id,
            (limit + 1) as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit;
        let items: Vec<_> = rows.into_iter().take(limit).collect();
        
        let next_cursor = if has_more {
            items.last().map(|r| encode_cursor(&(r.created_at, r.requester_id))).transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
