//! 管理员审计日志仓库
//! 基于 audit_logs 分区表

use chrono::{DateTime, Utc};
use serde_json::Value as JsonValue;
use sqlx::types::ipnetwork::IpNetwork;
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeIdCursor};

/// 审计日志行
#[derive(Debug, Clone, FromRow)]
pub struct AuditLogRow {
    pub log_id: Uuid,
    pub action: String,
    pub operator_id: i64,
    pub target_type: Option<String>,
    pub target_id: Option<String>,
    pub target_name: Option<String>,
    pub details: Option<JsonValue>,
    pub before_state: Option<JsonValue>,
    pub after_state: Option<JsonValue>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct AuditLogFilter {
    pub action: Option<String>,
    pub operator_id: Option<i64>,
    pub target_type: Option<String>,
    pub target_id: Option<String>,
    pub start_time: Option<DateTime<Utc>>,
    pub end_time: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct NewAuditLog {
    pub action: String,
    pub operator_id: i64,
    pub target_type: String,
    pub target_id: String,
    pub target_name: Option<String>,
    pub details: Option<JsonValue>,
    pub before_state: Option<JsonValue>,
    pub after_state: Option<JsonValue>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

#[derive(Clone)]
pub struct AdminAuditRepository {
    pool: PgPool,
}

impl AdminAuditRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建审计日志
    pub async fn create_log(&self, log: NewAuditLog) -> Result<Uuid> {
        let ip_network = log
            .ip_address
            .as_deref()
            .and_then(|ip| ip.parse::<IpNetwork>().ok());
        let log_id = sqlx::query_scalar!(
            r#"
            INSERT INTO audit_logs_partitioned
                (action, operator_id, target_type, target_id, target_name, details, before_state, after_state, ip_address, user_agent)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING log_id as "log_id!"
            "#,
            log.action,
            log.operator_id,
            log.target_type,
            log.target_id,
            log.target_name,
            log.details,
            log.before_state,
            log.after_state,
            ip_network,
            log.user_agent
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(log_id)
    }

    /// 审计日志列表
    pub async fn list_logs(
        &self,
        filter: AuditLogFilter,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<AuditLogRow>, Option<String>)> {
        let (cursor_time, cursor_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            AuditLogRow,
            r#"
            SELECT log_id as "log_id!",
                   action as "action!",
                   operator_id as "operator_id!",
                   target_type as "target_type?",
                   target_id as "target_id?",
                   target_name as "target_name?",
                   details as "details?",
                   before_state as "before_state?",
                   after_state as "after_state?",
                   ip_address::text as "ip_address?",
                   user_agent as "user_agent?",
                   created_at as "created_at!"
            FROM audit_logs
            WHERE ($1::text IS NULL OR action = $1)
              AND ($2::bigint IS NULL OR operator_id = $2)
              AND ($3::text IS NULL OR target_type = $3)
              AND ($4::text IS NULL OR target_id = $4)
              AND ($5::timestamptz IS NULL OR created_at >= $5)
              AND ($6::timestamptz IS NULL OR created_at <= $6)
              AND (created_at, log_id) < ($7, $8)
            ORDER BY created_at DESC, log_id DESC
            LIMIT $9
            "#,
            filter.action,
            filter.operator_id,
            filter.target_type,
            filter.target_id,
            filter.start_time,
            filter.end_time,
            cursor_time,
            cursor_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.log_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 审计日志详情
    pub async fn get_detail(&self, log_id: Uuid) -> Result<Option<AuditLogRow>> {
        let row = sqlx::query_as!(
            AuditLogRow,
            r#"
            SELECT log_id as "log_id!",
                   action as "action!",
                   operator_id as "operator_id!",
                   target_type as "target_type?",
                   target_id as "target_id?",
                   target_name as "target_name?",
                   details as "details?",
                   before_state as "before_state?",
                   after_state as "after_state?",
                   ip_address::text as "ip_address?",
                   user_agent as "user_agent?",
                   created_at as "created_at!"
            FROM audit_logs
            WHERE log_id = $1
            "#,
            log_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 用户封禁/解封日志
    pub async fn list_user_ban_logs(&self, user_id: &str) -> Result<Vec<AuditLogRow>> {
        let rows = sqlx::query_as!(
            AuditLogRow,
            r#"
            SELECT log_id as "log_id!",
                   action as "action!",
                   operator_id as "operator_id!",
                   target_type as "target_type?",
                   target_id as "target_id?",
                   target_name as "target_name?",
                   details as "details?",
                   before_state as "before_state?",
                   after_state as "after_state?",
                   ip_address::text as "ip_address?",
                   user_agent as "user_agent?",
                   created_at as "created_at!"
            FROM audit_logs
            WHERE target_type = 'user'
              AND target_id = $1
              AND action IN ('user.ban', 'user.unban')
            ORDER BY created_at ASC, log_id ASC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 群组解散日志
    pub async fn list_group_dismiss_logs(&self, group_id: &str) -> Result<Vec<AuditLogRow>> {
        let rows = sqlx::query_as!(
            AuditLogRow,
            r#"
            SELECT log_id as "log_id!",
                   action as "action!",
                   operator_id as "operator_id!",
                   target_type as "target_type?",
                   target_id as "target_id?",
                   target_name as "target_name?",
                   details as "details?",
                   before_state as "before_state?",
                   after_state as "after_state?",
                   ip_address::text as "ip_address?",
                   user_agent as "user_agent?",
                   created_at as "created_at!"
            FROM audit_logs
            WHERE target_type = 'group'
              AND target_id = $1
              AND action = 'group.dismiss'
            ORDER BY created_at ASC, log_id ASC
            "#,
            group_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }
}
