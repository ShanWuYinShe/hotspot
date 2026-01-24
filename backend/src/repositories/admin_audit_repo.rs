use chrono::{DateTime, Utc};
use serde_json::Value;
use sqlx::{FromRow, PgPool, QueryBuilder};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};

#[derive(Debug, Clone, Default)]
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
    pub details: Option<Value>,
    pub before_state: Option<Value>,
    pub after_state: Option<Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AuditLogRow {
    pub id: i64,
    pub log_id: Uuid,
    pub action: String,
    pub operator_id: i64,
    pub target_type: String,
    pub target_id: String,
    pub target_name: Option<String>,
    pub details: Option<Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AuditLogDetailRow {
    pub log_id: Uuid,
    pub action: String,
    pub operator_id: i64,
    pub target_type: String,
    pub target_id: String,
    pub target_name: Option<String>,
    pub details: Option<Value>,
    pub before_state: Option<Value>,
    pub after_state: Option<Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct AuditExportParams {
    pub status: String,
    pub format: String,
    pub action: Option<String>,
    pub operator_id: Option<i64>,
    pub target_type: Option<String>,
    pub target_id: Option<String>,
    pub start_time: Option<DateTime<Utc>>,
    pub end_time: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AuditExportRow {
    pub export_id: Uuid,
    pub status: String,
    pub download_key: Option<String>,
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct AdminAuditRepository {
    pool: PgPool,
}

impl AdminAuditRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create_log(&self, input: NewAuditLog) -> Result<AuditLogDetailRow> {
        let row = sqlx::query_as::<_, AuditLogDetailRow>(
            r#"
            INSERT INTO admin_audit_logs (
                action, operator_id, target_type, target_id, target_name,
                details, before_state, after_state, ip_address, user_agent
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING
                id, log_id, action, operator_id, target_type, target_id, target_name,
                details, before_state, after_state, ip_address, user_agent, created_at
            "#,
        )
        .bind(input.action)
        .bind(input.operator_id)
        .bind(input.target_type)
        .bind(input.target_id)
        .bind(input.target_name)
        .bind(input.details)
        .bind(input.before_state)
        .bind(input.after_state)
        .bind(input.ip_address)
        .bind(input.user_agent)
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn list_logs(
        &self,
        filter: AuditLogFilter,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<AuditLogRow>, Option<String>)> {
        let (last_created_at, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let mut builder = QueryBuilder::new(
            "SELECT id, log_id, action, operator_id, target_type, target_id, target_name, \
             details, ip_address, user_agent, created_at \
             FROM admin_audit_logs \
             WHERE (created_at, id) < (",
        );
        builder.push_bind(last_created_at);
        builder.push(", ");
        builder.push_bind(last_id);
        builder.push(")");

        if let Some(action) = filter.action {
            builder.push(" AND action = ");
            builder.push_bind(action);
        }
        if let Some(operator_id) = filter.operator_id {
            builder.push(" AND operator_id = ");
            builder.push_bind(operator_id);
        }
        if let Some(target_type) = filter.target_type {
            builder.push(" AND target_type = ");
            builder.push_bind(target_type);
        }
        if let Some(target_id) = filter.target_id {
            builder.push(" AND target_id = ");
            builder.push_bind(target_id);
        }
        if let Some(start_time) = filter.start_time {
            builder.push(" AND created_at >= ");
            builder.push_bind(start_time);
        }
        if let Some(end_time) = filter.end_time {
            builder.push(" AND created_at <= ");
            builder.push_bind(end_time);
        }

        builder.push(" ORDER BY created_at DESC, id DESC LIMIT ");
        builder.push_bind(limit);

        let rows = builder
            .build_query_as::<AuditLogRow>()
            .fetch_all(&self.pool)
            .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.created_at, r.id)))
            .transpose()?;

        Ok((rows, next_cursor))
    }

    pub async fn list_logs_for_export(
        &self,
        filter: AuditLogFilter,
        limit: i64,
    ) -> Result<Vec<AuditLogRow>> {
        let mut builder = QueryBuilder::new(
            "SELECT id, log_id, action, operator_id, target_type, target_id, target_name, \
             details, ip_address, user_agent, created_at \
             FROM admin_audit_logs WHERE 1=1",
        );

        if let Some(action) = filter.action {
            builder.push(" AND action = ");
            builder.push_bind(action);
        }
        if let Some(operator_id) = filter.operator_id {
            builder.push(" AND operator_id = ");
            builder.push_bind(operator_id);
        }
        if let Some(target_type) = filter.target_type {
            builder.push(" AND target_type = ");
            builder.push_bind(target_type);
        }
        if let Some(target_id) = filter.target_id {
            builder.push(" AND target_id = ");
            builder.push_bind(target_id);
        }
        if let Some(start_time) = filter.start_time {
            builder.push(" AND created_at >= ");
            builder.push_bind(start_time);
        }
        if let Some(end_time) = filter.end_time {
            builder.push(" AND created_at <= ");
            builder.push_bind(end_time);
        }

        builder.push(" ORDER BY created_at DESC, id DESC LIMIT ");
        builder.push_bind(limit);

        let rows = builder
            .build_query_as::<AuditLogRow>()
            .fetch_all(&self.pool)
            .await?;

        Ok(rows)
    }

    pub async fn get_detail(&self, log_id: Uuid) -> Result<Option<AuditLogDetailRow>> {
        let row = sqlx::query_as::<_, AuditLogDetailRow>(
            r#"
            SELECT log_id, action, operator_id, target_type, target_id, target_name,
                   details, before_state, after_state, ip_address, user_agent, created_at
            FROM admin_audit_logs
            WHERE log_id = $1
            "#,
        )
        .bind(log_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn create_export(&self, params: AuditExportParams) -> Result<AuditExportRow> {
        let row = sqlx::query_as::<_, AuditExportRow>(
            r#"
            INSERT INTO admin_audit_exports (
                status, format, action, operator_id, target_type, target_id, start_time, end_time
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING
                export_id, status, download_key, expires_at
            "#,
        )
        .bind(params.status)
        .bind(params.format)
        .bind(params.action)
        .bind(params.operator_id)
        .bind(params.target_type)
        .bind(params.target_id)
        .bind(params.start_time)
        .bind(params.end_time)
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn update_export(
        &self,
        export_id: Uuid,
        status: &str,
        download_key: Option<String>,
        expires_at: Option<DateTime<Utc>>,
        error_message: Option<String>,
    ) -> Result<AuditExportRow> {
        let row = sqlx::query_as::<_, AuditExportRow>(
            r#"
            UPDATE admin_audit_exports
            SET status = $2,
                download_key = $3,
                expires_at = $4,
                error_message = $5,
                updated_at = NOW()
            WHERE export_id = $1
            RETURNING
                export_id, status, download_key, expires_at
            "#,
        )
        .bind(export_id)
        .bind(status)
        .bind(download_key)
        .bind(expires_at)
        .bind(error_message)
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn get_export(&self, export_id: Uuid) -> Result<Option<AuditExportRow>> {
        let row = sqlx::query_as::<_, AuditExportRow>(
            r#"
            SELECT export_id, status, download_key, expires_at
            FROM admin_audit_exports
            WHERE export_id = $1
            "#,
        )
        .bind(export_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn list_user_ban_logs(&self, target_id: &str) -> Result<Vec<AuditLogDetailRow>> {
        let rows = sqlx::query_as::<_, AuditLogDetailRow>(
            r#"
            SELECT id, log_id, action, operator_id, target_type, target_id, target_name,
                   details, before_state, after_state, ip_address, user_agent, created_at
            FROM admin_audit_logs
            WHERE target_id = $1
              AND action IN ('user.ban', 'user.unban')
            ORDER BY created_at ASC, id ASC
            "#,
        )
        .bind(target_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn list_group_dismiss_logs(
        &self,
        target_id: &str,
    ) -> Result<Vec<AuditLogDetailRow>> {
        let rows = sqlx::query_as::<_, AuditLogDetailRow>(
            r#"
            SELECT id, log_id, action, operator_id, target_type, target_id, target_name,
                   details, before_state, after_state, ip_address, user_agent, created_at
            FROM admin_audit_logs
            WHERE target_id = $1
              AND action = 'group.dismiss'
            ORDER BY created_at ASC, id ASC
            "#,
        )
        .bind(target_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }
}
