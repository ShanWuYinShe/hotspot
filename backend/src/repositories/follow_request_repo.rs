//! 关注请求 Repository
//!
//! 处理关注请求的数据库操作
//!
//! 注意：此功能依赖用户模块迁移（migrations/002_user_module.sql）

use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

#[derive(Debug, Clone)]
pub struct FollowRequestRepository {
    pool: PgPool,
}

/// 关注请求数据行
#[derive(Debug, Clone, FromRow)]
pub struct FollowRequestRow {
    pub id: i64,
    pub request_id: Uuid,
    pub requester_id: i64,
    pub target_id: i64,
    pub message: Option<String>,
    pub created_at: DateTime<Utc>,
}

impl FollowRequestRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建关注请求
    ///
    /// 返回值：Ok(request_id) 创建成功，Err 如果已存在待处理的请求
    pub async fn create(
        &self,
        requester_id: i64,
        target_id: i64,
        message: Option<String>,
    ) -> Result<Uuid> {
        // 检查是否已有待处理的请求
        let existing = sqlx::query_scalar!(
            r#"
            SELECT request_id FROM follow_requests
            WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            "#,
            requester_id,
            target_id
        )
        .fetch_optional(&self.pool)
        .await?;

        if existing.is_some() {
            return Err(AppError::FollowRequestAlreadyExists(
                "已有待处理的关注请求".to_string(),
            ));
        }

        // 创建新请求（如果有之前被拒绝的请求，更新为 pending）
        let request_id = sqlx::query_scalar!(
            r#"
            INSERT INTO follow_requests (requester_id, target_id, message, status, created_at)
            VALUES ($1, $2, $3, 'pending', NOW())
            ON CONFLICT (requester_id, target_id) DO UPDATE SET
                message = EXCLUDED.message,
                status = 'pending',
                created_at = NOW(),
                handled_at = NULL
            RETURNING request_id
            "#,
            requester_id,
            target_id,
            message
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(request_id)
    }

    /// 检查是否存在待处理的关注请求
    pub async fn has_pending_request(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM follow_requests
                WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            )
            "#,
            requester_id,
            target_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.unwrap_or(false))
    }

    /// 获取用户收到的关注请求列表（待处理）
    pub async fn list_received(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<FollowRequestRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            FollowRequestRow,
            r#"
            SELECT 
                id, request_id, requester_id, target_id, message,
                created_at
            FROM follow_requests
            WHERE target_id = $1 AND status = 'pending' AND id < $2
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

    /// 处理关注请求（接受/拒绝）
    pub async fn handle(
        &self,
        request_id: Uuid,
        target_id: i64,
        accept: bool,
    ) -> Result<Option<FollowRequestRow>> {
        let status = if accept { "accepted" } else { "rejected" };

        let row = sqlx::query_as!(
            FollowRequestRow,
            r#"
            UPDATE follow_requests
            SET status = $1::TEXT::follow_request_status, handled_at = NOW()
            WHERE request_id = $2 AND target_id = $3 AND status = 'pending'
            RETURNING 
                id, request_id, requester_id, target_id, message,
                created_at
            "#,
            status,
            request_id,
            target_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 删除关注请求（用于取消）
    pub async fn delete(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            DELETE FROM follow_requests
            WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            "#,
            requester_id,
            target_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }
}
