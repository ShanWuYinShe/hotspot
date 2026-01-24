//! 关注请求 Repository
//!
//! 处理关注请求的数据库操作
//!
//! 注意：此功能依赖用户模块迁移（migrations/002_user_module.sql）

use chrono::{DateTime, Utc};
use sqlx::{PgPool, Row};
use uuid::Uuid;

use crate::error::{AppError, Result};
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

#[derive(Debug, Clone)]
pub struct FollowRequestRepository {
    pool: PgPool,
}

/// 关注请求数据行
#[derive(Debug, Clone)]
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
        let existing: Option<Uuid> = sqlx::query_scalar(
            r#"
            SELECT request_id FROM follow_requests
            WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            "#,
        )
        .bind(requester_id)
        .bind(target_id)
        .fetch_optional(&self.pool)
        .await?;

        if existing.is_some() {
            return Err(AppError::FollowRequestAlreadyExists(
                "已有待处理的关注请求".to_string(),
            ));
        }

        // 创建新请求（如果有之前被拒绝的请求，更新为 pending）
        let request_id: Uuid = sqlx::query_scalar(
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
        )
        .bind(requester_id)
        .bind(target_id)
        .bind(message)
        .fetch_one(&self.pool)
        .await?;

        Ok(request_id)
    }

    /// 检查是否存在待处理的关注请求
    pub async fn has_pending_request(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result: bool = sqlx::query_scalar(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM follow_requests
                WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            )
            "#,
        )
        .bind(requester_id)
        .bind(target_id)
        .fetch_one(&self.pool)
        .await?;

        Ok(result)
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

        let rows = sqlx::query(
            r#"
            SELECT 
                id, request_id, requester_id, target_id, message,
                created_at
            FROM follow_requests
            WHERE target_id = $1 AND status = 'pending' AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
        )
        .bind(user_id)
        .bind(last_id)
        .bind(limit as i64)
        .fetch_all(&self.pool)
        .await?;

        let rows: Vec<FollowRequestRow> = rows
            .into_iter()
            .map(|row| FollowRequestRow {
                id: row.get("id"),
                request_id: row.get("request_id"),
                requester_id: row.get("requester_id"),
                target_id: row.get("target_id"),
                message: row.get("message"),
                created_at: row.get("created_at"),
            })
            .collect();

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

        let row = sqlx::query(
            r#"
            UPDATE follow_requests
            SET status = $1::follow_request_status, handled_at = NOW()
            WHERE request_id = $2 AND target_id = $3 AND status = 'pending'
            RETURNING 
                id, request_id, requester_id, target_id, message,
                created_at
            "#,
        )
        .bind(status)
        .bind(request_id)
        .bind(target_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.map(|r| FollowRequestRow {
            id: r.get("id"),
            request_id: r.get("request_id"),
            requester_id: r.get("requester_id"),
            target_id: r.get("target_id"),
            message: r.get("message"),
            created_at: r.get("created_at"),
        }))
    }

    /// 删除关注请求（用于取消）
    pub async fn delete(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result = sqlx::query(
            r#"
            DELETE FROM follow_requests
            WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            "#,
        )
        .bind(requester_id)
        .bind(target_id)
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }
}
