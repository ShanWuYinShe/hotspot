use sqlx::{PgPool, Postgres, Transaction};
use uuid::Uuid;

use crate::{
    error::AppError,
    models::report::Report,
    types::enums::ReportStatus,
};

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
            INSERT INTO reports (
                report_id, reporter_id, target_type, target_id, 
                reason, description, status, handler_id, handler_note
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7::report_status, $8, $9)
            RETURNING id, report_id, reporter_id, target_type, target_id, 
                      reason, description, status as "status: ReportStatus", created_at, handled_at, 
                      handler_id, handler_note
            "#,
            report.report_id,
            report.reporter_id,
            report.target_type,
            report.target_id,
            report.reason,
            report.description,
            report.status as ReportStatus,
            report.handler_id,
            report.handler_note
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 通过report_id查找举报
    pub async fn find_report_by_id(&self, report_id: Uuid) -> Result<Option<Report>, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, report_id, reporter_id, target_type, target_id, 
                   reason, description, status as "status: ReportStatus", created_at, handled_at, 
                   handler_id, handler_note
            FROM reports
            WHERE report_id = $1
            "#,
            report_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查询用户提交的举报
    /// 按举报者查询举报记录（游标分页）
    /// cursor: report_id (UUID字符串)
    pub async fn find_reports_by_reporter(
        &self,
        reporter_id: i64,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Report>, AppError> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let report_uuid = uuid::Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM reports WHERE report_id = $1",
                report_uuid
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(AppError::from)?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, report_id, reporter_id, target_type, target_id, 
                   reason, description, status as "status: ReportStatus", created_at, handled_at, 
                   handler_id, handler_note
            FROM reports
            WHERE reporter_id = $1
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            reporter_id,
            cursor_id,
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
            SELECT id, report_id, reporter_id, target_type, target_id, 
                   reason, description, status as "status: ReportStatus", created_at, handled_at, 
                   handler_id, handler_note
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
    /// 查询所有举报（管理员）（游标分页）
    /// cursor: report_id (UUID字符串)
    pub async fn find_all_reports(
        &self,
        status: Option<ReportStatus>,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Report>, AppError> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let report_uuid = uuid::Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM reports WHERE report_id = $1",
                report_uuid
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(AppError::from)?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        sqlx::query_as!(
            Report,
            r#"
            SELECT id, report_id, reporter_id, target_type, target_id, 
                   reason, description, status as "status: ReportStatus", created_at, handled_at, 
                   handler_id, handler_note
            FROM reports
            WHERE ($1::report_status IS NULL OR status = $1::report_status OR $1::report_status = 'any'::report_status)
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            status as Option<ReportStatus>,
            cursor_id,
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
        status: ReportStatus,
        handler_id: i64,
        handler_note: Option<String>,
    ) -> Result<Report, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            UPDATE reports
            SET status = $2::report_status, handler_id = $3, handler_note = $4, handled_at = NOW()
            WHERE report_id = $1
            RETURNING id, report_id, reporter_id, target_type, target_id, 
                      reason, description, status as "status: ReportStatus", created_at, handled_at, 
                      handler_id, handler_note
            "#,
            report_id,
            status as ReportStatus,
            handler_id,
            handler_note
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 使用事务更新举报状态
    pub async fn update_report_status_with_tx(
        &self,
        report_id: Uuid,
        status: ReportStatus,
        handler_id: i64,
        handler_note: Option<String>,
        tx: &mut Transaction<'_, Postgres>,
    ) -> Result<Report, AppError> {
        sqlx::query_as!(
            Report,
            r#"
            UPDATE reports
            SET status = $2::report_status, handler_id = $3, handler_note = $4, handled_at = NOW()
            WHERE report_id = $1
            RETURNING id, report_id, reporter_id, target_type, target_id, 
                      reason, description, status as "status: ReportStatus", created_at, handled_at, 
                      handler_id, handler_note
            "#,
            report_id,
            status as ReportStatus,
            handler_id,
            handler_note
        )
        .fetch_one(&mut **tx)
        .await
        .map_err(AppError::from)
    }

    /// 删除举报
    pub async fn delete_report(&self, report_id: Uuid) -> Result<u64, AppError> {
        sqlx::query!(
            r#"
            DELETE FROM reports
            WHERE report_id = $1
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
