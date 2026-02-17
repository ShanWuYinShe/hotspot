use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::{ReportHandleAction, ReportStatus, ReportTargetType};
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated, TimeIdCursor};
use std::collections::HashMap;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ReportRow {
    pub report_id: Uuid,
    pub reporter_id: i64,
    pub target_type: ReportTargetType,
    pub target_id: String,
    pub reason: String,
    pub description: Option<String>,
    pub status: ReportStatus,
    pub handled_at: Option<chrono::DateTime<chrono::Utc>>,
    pub handled_by: Option<i64>,
    pub handled_action: Option<ReportHandleAction>,
    pub handled_note: Option<String>,
    pub target_snapshot: Option<serde_json::Value>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone)]
pub struct ReportRepository {
    pool: PgPool,
}

impl ReportRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: &str,
        reason: &str,
        description: Option<String>,
    ) -> Result<ReportRow> {
        let row = sqlx::query_as!(
            ReportRow,
            r#"
            INSERT INTO reports (reporter_id, target_type, target_id, reason, description)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING report_id,
                      reporter_id,
                      target_type as "target_type: ReportTargetType",
                      target_id,
                      reason,
                      description,
                      status as "status!: ReportStatus",
                      handled_at,
                      handled_by,
                      handled_action as "handled_action: ReportHandleAction",
                      handled_note,
                      target_snapshot,
                      created_at,
                      updated_at
            "#,
            reporter_id,
            target_type as ReportTargetType,
            target_id,
            reason,
            description
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn update_status(
        &self,
        report_id: Uuid,
        status: ReportStatus,
        action: ReportHandleAction,
        note: Option<String>,
        handler_id: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE reports
            SET status = $2,
                handled_action = $3,
                handled_note = $4,
                handled_by = $5,
                handled_at = NOW(),
                updated_at = NOW()
            WHERE report_id = $1
            "#,
            report_id,
            status as ReportStatus,
            action as ReportHandleAction,
            note,
            handler_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn exists(&self, report_id: Uuid) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(SELECT 1 FROM reports WHERE report_id = $1) as "exists!"
            "#,
            report_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(exists)
    }

    /// 用户举报记录
    pub async fn find_reports_by_reporter(
        &self,
        reporter_id: i64,
        status: Option<ReportStatus>,
        target_type: Option<ReportTargetType>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<ReportRow> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (chrono::DateTime::<chrono::Utc>::MAX_UTC, Uuid::max())
        };

        let status_str = status.map(|s| s.as_str().to_string());
        let target_type_str = target_type.map(|t| t.as_str().to_string());
        let fetch_limit = limit as i64 + 1;
        let rows = sqlx::query_as!(
            ReportRow,
            r#"
            SELECT report_id,
                   reporter_id,
                   target_type as "target_type: ReportTargetType",
                   target_id,
                   reason,
                   description,
                   status as "status!: ReportStatus",
                   handled_at,
                   handled_by,
                   handled_action as "handled_action: ReportHandleAction",
                   handled_note,
                   target_snapshot,
                   created_at,
                   updated_at
            FROM reports
            WHERE reporter_id = $1
              AND ($2::text IS NULL OR status::text = $2)
              AND ($3::text IS NULL OR target_type::text = $3)
              AND (created_at, report_id) < ($4, $5)
            ORDER BY created_at DESC, report_id DESC
            LIMIT $6
            "#,
            reporter_id,
            status_str,
            target_type_str,
            last_ts,
            last_id,
            fetch_limit
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.report_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 管理员举报列表
    pub async fn find_all_reports(
        &self,
        status: Option<ReportStatus>,
        target_type: Option<ReportTargetType>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<ReportRow> {
        let status_str = status.map(|s| s.as_str().to_string());
        let target_type_str = target_type.map(|t| t.as_str().to_string());
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (chrono::DateTime::<chrono::Utc>::MAX_UTC, Uuid::max())
        };

        let fetch_limit = limit as i64 + 1;
        let rows = sqlx::query_as!(
            ReportRow,
            r#"
            SELECT report_id,
                   reporter_id,
                   target_type as "target_type: ReportTargetType",
                   target_id,
                   reason,
                   description,
                   status as "status!: ReportStatus",
                   handled_at,
                   handled_by,
                   handled_action as "handled_action: ReportHandleAction",
                   handled_note,
                   target_snapshot,
                   created_at,
                   updated_at
            FROM reports
            WHERE ($1::text IS NULL OR status::text = $1)
              AND ($2::text IS NULL OR target_type::text = $2)
              AND (created_at, report_id) < ($3, $4)
            ORDER BY created_at DESC, report_id DESC
            LIMIT $5
            "#,
            status_str,
            target_type_str,
            last_ts,
            last_id,
            fetch_limit
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.report_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn count_group_reports(
        &self,
        group_ids: &[i64],
    ) -> Result<HashMap<i64, (i64, Option<DateTime<Utc>>)>> {
        if group_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let group_id_strings: Vec<String> = group_ids.iter().map(|id| id.to_string()).collect();
        let rows = sqlx::query!(
            r#"
            SELECT target_id as "target_id!",
                   COUNT(*)::bigint as "reports_count!",
                   MAX(created_at) as "last_reported_at?"
            FROM reports
            WHERE target_type = 'group'
              AND target_id = ANY($1)
            GROUP BY target_id
            "#,
            &group_id_strings
        )
        .fetch_all(&self.pool)
        .await?;

        let mut result = HashMap::new();
        for row in rows {
            if let Ok(id) = row.target_id.parse::<i64>() {
                result.insert(id, (row.reports_count, row.last_reported_at));
            }
        }

        Ok(result)
    }
}
