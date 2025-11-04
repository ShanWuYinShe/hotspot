use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            CreateReportRequest, GetReportHistoryRequest, GetReportsRequest, HandleReportRequest,
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

        Ok(Self::to_response(created_report))
    }

    /// 获取用户的举报历史
    pub async fn get_report_history(
        &self,
        reporter_id: i64,
        req: GetReportHistoryRequest,
    ) -> Result<PaginatedResponse<ReportResponse>, AppError> {
        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let reports = self
            .report_repo
            .find_reports_by_reporter(reporter_id, req.cursor.clone(), limit + 1)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<Report> = reports.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的report_id（业务ID）
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| r.report_id.to_string())
        } else {
            None
        };

        let response_items: Vec<ReportResponse> = reports_to_return
            .into_iter()
            .map(Self::to_response)
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

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let reports = self
            .report_repo
            .find_all_reports(req.status, req.cursor.clone(), limit + 1)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<Report> = reports.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的report_id（业务ID）
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| r.report_id.to_string())
        } else {
            None
        };

        let response_items: Vec<ReportResponse> = reports_to_return
            .into_iter()
            .map(Self::to_response)
            .collect();

        Ok(PaginatedResponse {
            items: response_items,
            has_more,
            next_cursor,
        })
    }

    /// 审核举报（管理员）
    pub async fn handle_report(
        &self,
        admin_id: i64,
        req: HandleReportRequest,
    ) -> Result<ReportResponse, AppError> {
        // TODO: 验证管理员权限

        // 验证action
        if !["resolve", "dismiss", "escalate"].contains(&req.action.as_str()) {
            return Err(AppError::InvalidInput("无效的操作类型".to_string()));
        }

        // 解析report_id
        let report_id = uuid::Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的举报ID".to_string()))?;

        // 查找举报
        let report = self
            .report_repo
            .find_report_by_id(report_id)
            .await?
            .ok_or_else(|| AppError::NotFound("举报不存在".to_string()))?;

        // 根据action映射status
        let status = match req.action.as_str() {
            "resolve" => "resolved",
            "dismiss" => "dismissed",
            "escalate" => "reviewing",
            _ => return Err(AppError::InvalidInput("无效的操作类型".to_string())),
        };

        // 更新状态
        let updated_report = self
            .report_repo
            .update_report_status(report.report_id, status.to_string(), admin_id, req.note)
            .await?;

        // TODO: 根据审核结果采取相应措施
        // 例如：如果resolved，可能需要封禁用户、删除内容等

        Ok(Self::to_response(updated_report))
    }

    /// 辅助方法：将Report转换为ReportResponse
    fn to_response(report: Report) -> ReportResponse {
        ReportResponse {
            id: report.report_id.to_string(),
            reporter_id: report.reporter_id.to_string(),
            reporter_nickname: None,
            target_type: report.target_type,
            target_id: report.target_id,
            reason: report.reason,
            description: report.description,
            status: report.status,
            created_at: report.created_at.to_rfc3339(),
            handled_at: report.handled_at.map(|t| t.to_rfc3339()),
            handler_id: report.handler_id.map(|id| id.to_string()),
            handler_note: report.handler_note,
        }
    }
}

