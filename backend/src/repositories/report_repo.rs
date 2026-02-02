use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{types::Json, FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::{ReportStatus, ReportTargetType};
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// 目标内容快照
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "jsonb")]
pub struct TargetSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<i64>,
}

/// 举报Row - 数据库查询返回类型
#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
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
    pub target_snapshot: Option<Json<TargetSnapshot>>,
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
        let row = sqlx::query_as!(
            ReportRow,
            r#"
            INSERT INTO reports (reporter_id, target_type, target_id, reason, description)
            VALUES ($1, $2::text::report_target_type, $3, $4, $5)
            RETURNING id, report_id, reporter_id,
                      target_type as "target_type!: ReportTargetType",
                      target_id, reason, description,
                      status as "status!: ReportStatus",
                      handled_by, handled_at, handler_note, created_at,
                      is_deleted, deleted_at,
                      target_snapshot as "target_snapshot: Json<TargetSnapshot>"
            "#,
            reporter_id,
            target_type.as_str(),
            target_id,
            reason,
            description
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
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

        let reports = sqlx::query_as!(
            ReportRow,
            r#"
            SELECT id, report_id, reporter_id,
                   target_type as "target_type!: ReportTargetType",
                   target_id, reason, description,
                   status as "status!: ReportStatus",
                   created_at, handled_by, handled_at, handler_note,
                   is_deleted, deleted_at,
                   target_snapshot as "target_snapshot: Json<TargetSnapshot>"
            FROM reports
            WHERE reporter_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            reporter_id,
            last_id,
            (limit + 1) as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        // 判断是否有更多数据
        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<_> = reports.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| encode_cursor(&r.id)).transpose()?
        } else {
            None
        };
        Ok((reports_to_return, next_cursor))
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

        let status_str = status.as_ref().map(|s| s.as_str());
        let target_type_str = target_type.as_ref().map(|t| t.as_str());
        
        let reports = sqlx::query_as!(
            ReportRow,
            r#"
            SELECT id, report_id, reporter_id,
                   target_type as "target_type!: ReportTargetType",
                   target_id, reason, description,
                   status as "status!: ReportStatus",
                   created_at, handled_by, handled_at, handler_note,
                   is_deleted, deleted_at,
                   target_snapshot as "target_snapshot: Json<TargetSnapshot>"
            FROM reports
            WHERE ($1::text IS NULL OR status::text = $1)
              AND ($2::text IS NULL OR target_type::text = $2)
              AND id < $3
              AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $4
            "#,
            status_str,
            target_type_str,
            last_id,
            (limit + 1) as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        // 判断是否有更多数据
        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<_> = reports.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| encode_cursor(&r.id)).transpose()?
        } else {
            None
        };
        
        Ok((reports_to_return, next_cursor))
    }
}
