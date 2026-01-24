use uuid::Uuid;

use crate::{
    dto::{
        request::{CreateReportRequest, HandleReportRequest, MyReportsRequest, ReportsRequest},
        response::{CreateReportData, PaginatedData, PaginationMeta, ReportItem},
    },
    error::AppError,
    repositories::report_repo::{ReportRepository, ReportRow},
    types::enums::ReportStatus,
    utils::validation::Validator,
};

#[derive(Debug, Clone)]
pub struct ReportService {
    report_repo: ReportRepository,
}

impl ReportService {
    pub fn new(report_repo: ReportRepository) -> Self {
        Self { report_repo }
    }

    /// 提交举报
    pub async fn create_report(
        &self,
        reporter_id: i64,
        req: CreateReportRequest,
    ) -> Result<CreateReportData, AppError> {
        // target_type 已是强类型枚举

        // 验证reason
        if req.reason.trim().is_empty() {
            return Err(AppError::InvalidFormat("举报原因不能为空".to_string()));
        }

        let created_report = self
            .report_repo
            .create(
                reporter_id,
                req.target_type,
                req.target_id,
                req.reason,
                req.description,
            )
            .await?;

        Ok(CreateReportData {
            report_id: created_report.report_id.to_string(),
        })
    }

    /// 获取用户的举报历史
    pub async fn get_my_reports(
        &self,
        reporter_id: i64,
        req: MyReportsRequest,
    ) -> Result<PaginatedData<ReportItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (reports, next_cursor) = self
            .report_repo
            .find_reports_by_reporter(reporter_id, limit as i32, cursor)
            .await?;

        let response_items: Vec<ReportItem> =
            reports.into_iter().map(Self::to_report_item).collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 获取所有举报（管理员）
    pub async fn get_list(
        &self,
        _admin_id: i64, // 验证在Handler层
        req: ReportsRequest,
    ) -> Result<PaginatedData<ReportItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        // status 和 target_type 已是强类型枚举，直接使用
        let (reports, next_cursor) = self
            .report_repo
            .find_all_reports(req.status, req.target_type, limit as i32, cursor)
            .await?;

        let response_items: Vec<ReportItem> =
            reports.into_iter().map(Self::to_report_item).collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 审核举报（管理员）
    pub async fn handle(&self, admin_id: i64, req: HandleReportRequest) -> Result<(), AppError> {
        use crate::types::enums::ReportHandleAction;

        // action 已是强类型枚举

        // 解析report_id
        let report_id = Uuid::parse_str(&req.report_id)
            .map_err(|_| AppError::BadRequest("无效的举报ID".to_string()))?;

        // 仅做存在性校验（避免拉整行）
        if !self.report_repo.exists(report_id).await? {
            return Err(AppError::NotFound("举报不存在".to_string()));
        }

        // 根据action映射status
        let status = match req.action {
            ReportHandleAction::Resolve => ReportStatus::Resolved,
            ReportHandleAction::Dismiss => ReportStatus::Dismissed,
            ReportHandleAction::Escalate => ReportStatus::Reviewing,
        };

        // 更新状态
        self.report_repo
            .update_status(report_id, status, admin_id, req.note.clone())
            .await?;

        Ok(())
    }

    /// 辅助方法：将ReportRow转换为ReportItem
    fn to_report_item(report: ReportRow) -> ReportItem {
        use crate::dto::response::ReportTargetSnapshot;

        ReportItem {
            report_id: report.report_id.to_string(),
            target_type: report.target_type,
            target_id: report.target_id,
            reason: report.reason,
            description: report.description,
            status: report.status,
            created_at: report.created_at.to_rfc3339(),
            updated_at: None,
            handled_at: report.handled_at.map(|t| t.to_rfc3339()),
            resolution_note: report.handler_note,
            reporter: None,
            target_snapshot: report.target_snapshot.map(|s| ReportTargetSnapshot {
                content: s.content,
                media_urls: s.media_urls,
                owner_id: s.owner_id.map(|id| id.to_string()),
            }),
        }
    }
}
