use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use std::str::FromStr;
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::{ReportStatus, ReportTargetType};
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// 目标内容快照
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TargetSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<i64>,
}

/// 举报Row - 数据库查询返回类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReportRow {
    pub id: i64,
    pub report_id: Uuid,
    pub reporter_id: i64,
    pub target_type: ReportTargetType,
    pub target_id: String,
    pub reason: String,
    pub description: Option<String>,
    pub status: ReportStatus,
    pub created_at: DateTime<Utc>,
    pub handled_by: Option<i64>,
    pub handled_at: Option<DateTime<Utc>>,
    pub handler_note: Option<String>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    pub target_snapshot: Option<TargetSnapshot>,
}

impl ReportRow {
    /// 从数据库行构建 ReportRow
    fn from_row(row: &sqlx::postgres::PgRow) -> Self {
        use sqlx::Row;

        let target_snapshot_json: Option<serde_json::Value> = row.get("target_snapshot");

        Self {
            id: row.get("id"),
            report_id: row.get("report_id"),
            reporter_id: row.get("reporter_id"),
            target_type: ReportTargetType::from_str(row.get("target_type"))
                .unwrap_or(ReportTargetType::Moment),
            target_id: row.get("target_id"),
            reason: row.get("reason"),
            description: row.get("description"),
            status: ReportStatus::from_str(row.get("status")).unwrap_or(ReportStatus::Pending),
            created_at: row.get("created_at"),
            handled_by: row.get("handled_by"),
            handled_at: row.get("handled_at"),
            handler_note: row.get("handler_note"),
            is_deleted: row.get("is_deleted"),
            deleted_at: row.get("deleted_at"),
            target_snapshot: target_snapshot_json.and_then(|v| serde_json::from_value(v).ok()),
        }
    }
}

#[derive(Debug, Clone)]
pub struct ReportRepository {
    pool: PgPool,
}

impl ReportRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 是否存在（用于仅做存在性校验的场景，避免拉取整行）
    pub async fn exists(&self, report_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM reports
                WHERE report_id = $1
            ) as "exists!"
            "#,
            report_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 创建举报
    pub async fn create(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
        reason: String,
        description: Option<String>,
    ) -> Result<ReportRow> {
        let row = sqlx::query(
            r#"
            INSERT INTO reports (reporter_id, target_type, target_id, reason, description)
            VALUES ($1, $2::report_target_type, $3, $4, $5)
            RETURNING report_id, reporter_id,
                      target_type, target_id, reason, description,
                      status, handled_by, handled_at, handler_note, created_at,
                      id, is_deleted, deleted_at, target_snapshot
            "#,
        )
        .bind(reporter_id)
        .bind(target_type.to_string())
        .bind(&target_id)
        .bind(&reason)
        .bind(&description)
        .fetch_one(&self.pool)
        .await?;

        Ok(ReportRow::from_row(&row))
    }

    /// 更新举报状态
    pub async fn update_status(
        &self,
        report_id: Uuid,
        status: ReportStatus,
        handled_by: i64,
        handler_note: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE reports
            SET status = $2::report_status, handled_by = $3, handled_at = NOW(), handler_note = $4
            WHERE report_id = $1
            "#,
            report_id,
            status as ReportStatus,
            handled_by,
            handler_note
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按举报者ID查询举报列表
    pub async fn find_reports_by_reporter(
        &self,
        reporter_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<ReportRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query(
            r#"
            SELECT id, report_id, reporter_id, target_type,
                   target_id, reason, description, status,
                   created_at, handled_by, handled_at, handler_note,
                   is_deleted, deleted_at, target_snapshot
            FROM reports
            WHERE reporter_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
        )
        .bind(reporter_id)
        .bind(last_id)
        .bind(limit as i64)
        .fetch_all(&self.pool)
        .await?;

        let reports: Vec<ReportRow> = rows.iter().map(ReportRow::from_row).collect();
        let next_cursor = reports.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((reports, next_cursor))
    }

    /// 查询所有举报列表
    pub async fn find_all_reports(
        &self,
        status: Option<ReportStatus>,
        target_type: Option<crate::types::enums::ReportTargetType>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<ReportRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query(
            r#"
            SELECT id, report_id, reporter_id, target_type,
                   target_id, reason, description, status,
                   created_at, handled_by, handled_at, handler_note,
                   is_deleted, deleted_at, target_snapshot
            FROM reports
            WHERE ($1::report_status IS NULL OR status = $1)
              AND ($2::report_target_type IS NULL OR target_type = $2)
              AND id < $3
              AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $4
            "#,
        )
        .bind(status.map(|s| s.to_string()))
        .bind(target_type.map(|t| t.to_string()))
        .bind(last_id)
        .bind(limit as i64)
        .fetch_all(&self.pool)
        .await?;

        let reports: Vec<ReportRow> = rows.iter().map(ReportRow::from_row).collect();
        let next_cursor = reports.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((reports, next_cursor))
    }
}
