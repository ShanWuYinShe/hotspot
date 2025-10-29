use sqlx::{PgPool, Postgres, Transaction};
use uuid::Uuid;

use crate::{error::AppError, models::report::Report};

#[derive(Debug, Clone)]
pub struct ReportRepository {
    pool: PgPool,
}

impl ReportRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建举报
    pub async fn create_report(&self, report: &Report) -> Result<Report, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            INSERT INTO reports (id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            "#,
            report.id,
            report.reporter_id,
            report.target_type,
            report.target_id,
            report.reason,
            report.description,
            report.status,
            report.reviewed_by,
            report.review_note,
            report.created_at,
            report.updated_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 通过ID查找举报
    pub async fn find_report_by_id(&self, report_id: Uuid) -> Result<Option<Report>, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            FROM reports
            WHERE id = $1
            "#,
            report_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查询用户提交的举报
    pub async fn find_reports_by_reporter(
        &self,
        reporter_id: i64,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<Report>, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            FROM reports
            WHERE reporter_id = $1
            ORDER BY created_at DESC
            OFFSET $2
            LIMIT $3
            "#,
            reporter_id,
            offset,
            limit
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查询待审核的举报（管理员使用）
    pub async fn find_pending_reports(
        &self,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<Report>, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            FROM reports
            WHERE status = 'pending'
            ORDER BY created_at ASC
            OFFSET $1
            LIMIT $2
            "#,
            offset,
            limit
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查询所有举报（管理员使用）
    pub async fn find_all_reports(
        &self,
        status: Option<String>,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<Report>, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            FROM reports
            WHERE ($1::TEXT IS NULL OR status = $1)
            ORDER BY created_at DESC
            OFFSET $2
            LIMIT $3
            "#,
            status as Option<String>,
            offset,
            limit
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 更新举报状态（审核）
    pub async fn update_report_status(
        &self,
        report_id: Uuid,
        status: String,
        reviewed_by: i64,
        review_note: Option<String>,
    ) -> Result<Report, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            UPDATE reports
            SET status = $2, reviewed_by = $3, review_note = $4, updated_at = NOW()
            WHERE id = $1
            RETURNING id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            "#,
            report_id,
            status,
            reviewed_by,
            review_note
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 使用事务更新举报状态
    pub async fn update_report_status_with_tx(
        &self,
        report_id: Uuid,
        status: String,
        reviewed_by: i64,
        review_note: Option<String>,
        tx: &mut Transaction<'_, Postgres>,
    ) -> Result<Report, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            UPDATE reports
            SET status = $2, reviewed_by = $3, review_note = $4, updated_at = NOW()
            WHERE id = $1
            RETURNING id, reporter_id, target_type, target_id, reason, description, status, reviewed_by, review_note, created_at, updated_at
            "#,
            report_id,
            status,
            reviewed_by,
            review_note
        )
        .fetch_one(tx)
        .await
        .map_err(AppError::from)
    }

    /// 删除举报
    pub async fn delete_report(&self, report_id: Uuid) -> Result<u64, AppError> {
        sqlx::query!(
            r#"
            DELETE FROM reports
            WHERE id = $1
            "#,
            report_id
        )
        .execute(&self.pool)
        .await
        .map(|r| r.rows_affected())
        .map_err(AppError::from)
    }

    /// 查询针对特定目标的举报数量
    pub async fn count_reports_by_target(
        &self,
        target_type: &str,
        target_id: &str,
    ) -> Result<i64, AppError> {
        sqlx::query_scalar!(
            r#"
            SELECT COUNT(id) AS "count!"
            FROM reports
            WHERE target_type = $1 AND target_id = $2
            "#,
            target_type,
            target_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }
}

