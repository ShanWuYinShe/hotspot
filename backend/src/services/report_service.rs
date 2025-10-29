use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateReportRequest, GetReportHistoryRequest, GetReportsRequest, ReviewReportRequest,
        },
        response::{PaginatedResponse, ReportResponse},
    },
    error::AppError,
    models::report::Report,
    repositories::report_repo::ReportRepository,
};

#[derive(Debug, Clone)]
pub struct ReportService {
    report_repo: ReportRepository,
}

impl ReportService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            report_repo: ReportRepository::new(pool),
        }
    }

    /// 提交举报
    pub async fn create_report(
        &self,
        reporter_id: i64,
        req: CreateReportRequest,
    ) -> Result<ReportResponse, AppError> {
        // 验证target_type
        if !["user", "moment", "comment", "message"]
            .contains(&req.target_type.as_str())
        {
            return Err(AppError::InvalidInput(
                "无效的举报类型".to_string(),
            ));
        }

        // 验证reason
        if req.reason.trim().is_empty() {
            return Err(AppError::InvalidInput("举报原因不能为空".to_string()));
        }

        let report = Report::new(
            reporter_id,
            req.target_type,
            req.target_id,
            req.reason,
            req.description,
        );

        let created_report = self.report_repo.create_report(&report).await?;

        Ok(ReportResponse {
            id: created_report.id,
            reporter_id: created_report.reporter_id,
            target_type: created_report.target_type,
            target_id: created_report.target_id,
            reason: created_report.reason,
            description: created_report.description,
            status: created_report.status,
            reviewed_by: created_report.reviewed_by,
            review_note: created_report.review_note,
            created_at: created_report.created_at,
            updated_at: created_report.updated_at,
        })
    }

    /// 获取用户的举报历史
    pub async fn get_report_history(
        &self,
        reporter_id: i64,
        req: GetReportHistoryRequest,
    ) -> Result<PaginatedResponse<ReportResponse>, AppError> {
        let offset = req.cursor.unwrap_or(0);
        let limit = req.limit.min(100);

        let reports = self
            .report_repo
            .find_reports_by_reporter(reporter_id, offset, limit + 1)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports: Vec<Report> = if has_more {
            reports.into_iter().take(limit as usize).collect()
        } else {
            reports
        };

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        let response_items: Vec<ReportResponse> = reports
            .into_iter()
            .map(|r| ReportResponse {
                id: r.id,
                reporter_id: r.reporter_id,
                target_type: r.target_type,
                target_id: r.target_id,
                reason: r.reason,
                description: r.description,
                status: r.status,
                reviewed_by: r.reviewed_by,
                review_note: r.review_note,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect();

        Ok(PaginatedResponse {
            items: response_items,
            has_more,
            next_cursor,
        })
    }

    /// 获取所有举报（管理员）
    pub async fn get_all_reports(
        &self,
        _admin_id: i64, // 这里应该验证是否为管理员
        req: GetReportsRequest,
    ) -> Result<PaginatedResponse<ReportResponse>, AppError> {
        // TODO: 验证管理员权限

        let offset = req.cursor.unwrap_or(0);
        let limit = req.limit.min(100);

        let reports = self
            .report_repo
            .find_all_reports(req.status, offset, limit + 1)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports: Vec<Report> = if has_more {
            reports.into_iter().take(limit as usize).collect()
        } else {
            reports
        };

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        let response_items: Vec<ReportResponse> = reports
            .into_iter()
            .map(|r| ReportResponse {
                id: r.id,
                reporter_id: r.reporter_id,
                target_type: r.target_type,
                target_id: r.target_id,
                reason: r.reason,
                description: r.description,
                status: r.status,
                reviewed_by: r.reviewed_by,
                review_note: r.review_note,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect();

        Ok(PaginatedResponse {
            items: response_items,
            has_more,
            next_cursor,
        })
    }

    /// 审核举报（管理员）
    pub async fn review_report(
        &self,
        admin_id: i64,
        req: ReviewReportRequest,
    ) -> Result<ReportResponse, AppError> {
        // TODO: 验证管理员权限

        // 验证status
        if !["reviewing", "resolved", "rejected"].contains(&req.status.as_str()) {
            return Err(AppError::InvalidInput("无效的审核状态".to_string()));
        }

        // 查找举报
        let report = self
            .report_repo
            .find_report_by_id(req.report_id)
            .await?
            .ok_or_else(|| AppError::NotFound("举报不存在".to_string()))?;

        // 更新状态
        let updated_report = self
            .report_repo
            .update_report_status(report.id, req.status, admin_id, req.review_note)
            .await?;

        // TODO: 根据审核结果采取相应措施
        // 例如：如果resolved，可能需要封禁用户、删除内容等

        Ok(ReportResponse {
            id: updated_report.id,
            reporter_id: updated_report.reporter_id,
            target_type: updated_report.target_type,
            target_id: updated_report.target_id,
            reason: updated_report.reason,
            description: updated_report.description,
            status: updated_report.status,
            reviewed_by: updated_report.reviewed_by,
            review_note: updated_report.review_note,
            created_at: updated_report.created_at,
            updated_at: updated_report.updated_at,
        })
    }
}

