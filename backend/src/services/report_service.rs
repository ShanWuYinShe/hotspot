use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateReportRequest, GetMyReportsRequest, GetReportsRequest, HandleReportRequest,
        },
        response::{PaginatedData, PaginationMeta, ReportItem, CreateReportData},
    },
    error::AppError,
    repositories::report_repo::{ReportRepository, ReportRow},
    types::enums::{ReportStatus, ReportTargetType},
    utils::validation::Validator,
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
    ) -> Result<CreateReportData, AppError> {
        // 验证target_type
        let target_type_str = req.target_type.to_lowercase();
        let target_type = match target_type_str.as_str() {
            "user" => ReportTargetType::User,
            "moment" => ReportTargetType::Moment,
            "comment" => ReportTargetType::Comment,
            "message" => ReportTargetType::Message,
            "group" => ReportTargetType::Group,
            _ => return Err(AppError::InvalidInput("无效的举报类型".to_string())),
        };

        // 验证reason
        if req.reason.trim().is_empty() {
            return Err(AppError::InvalidInput("举报原因不能为空".to_string()));
        }

        let created_report = self.report_repo.create(
            reporter_id,
            target_type,
            req.target_id,
            req.reason,
            req.description
        ).await?;

        Ok(CreateReportData {
            id: created_report.report_id.to_string(),
        })
    }

    /// 获取用户的举报历史
    pub async fn get_my_reports(
        &self,
        reporter_id: i64,
        req: GetMyReportsRequest,
    ) -> Result<PaginatedData<ReportItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        // 使用游标分页：多取一条判断has_more
        let reports = self
            .report_repo
            .find_reports_by_reporter(reporter_id, limit + 1, cursor)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<ReportRow> = reports.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的cursor_id
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| r.cursor_id.to_string())
        } else {
            None
        };

        let response_items: Vec<ReportItem> = reports_to_return
            .into_iter()
            .map(Self::to_report_item)
            .collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 获取所有举报（管理员）
    pub async fn get_list(
        &self,
        _admin_id: i64, // 验证在Handler层
        req: GetReportsRequest,
    ) -> Result<PaginatedData<ReportItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        
        let status = if let Some(s) = req.status {
             match s.to_lowercase().as_str() {
                "pending" => Some(ReportStatus::Pending),
                "reviewing" => Some(ReportStatus::Reviewing),
                "resolved" => Some(ReportStatus::Resolved),
                "dismissed" => Some(ReportStatus::Dismissed),
                _ => return Err(AppError::InvalidInput("无效的举报状态".to_string())),
             }
        } else {
            None
        };

        // 使用游标分页：多取一条判断has_more
        let reports = self
            .report_repo
            .find_all_reports(status, limit + 1, cursor)
            .await?;

        let has_more = reports.len() > limit as usize;
        let reports_to_return: Vec<ReportRow> = reports.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的cursor_id
        let next_cursor = if has_more {
            reports_to_return.last().map(|r| r.cursor_id.to_string())
        } else {
            None
        };

        let response_items: Vec<ReportItem> = reports_to_return
            .into_iter()
            .map(Self::to_report_item)
            .collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 审核举报（管理员）
    pub async fn handle(
        &self,
        admin_id: i64,
        req: HandleReportRequest,
    ) -> Result<(), AppError> {
        // 验证在Handler层

        // 验证action
        if !["resolve", "dismiss", "escalate"].contains(&req.action.as_str()) {
            return Err(AppError::InvalidInput("无效的操作类型".to_string()));
        }

        // 解析report_id
        let report_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的举报ID".to_string()))?;

        // 查找举报
        let report = self
            .report_repo
            .find_by_id(report_id)
            .await?
            .ok_or_else(|| AppError::NotFound("举报不存在".to_string()))?;

        // 根据action映射status
        let status = match req.action.as_str() {
            "resolve" => ReportStatus::Resolved,
            "dismiss" => ReportStatus::Dismissed,
            "escalate" => ReportStatus::Reviewing,
            _ => return Err(AppError::InvalidInput("无效的操作类型".to_string())),
        };

        // 更新状态
        self.report_repo
            .update_status(report.report_id, status, admin_id, req.note.clone())
            .await?;

        Ok(())
    }

    /// 辅助方法：将ReportRow转换为ReportItem
    fn to_report_item(report: ReportRow) -> ReportItem {
        ReportItem {
            id: report.report_id.to_string(),
            target_type: report.target_type.to_string(),
            target_id: report.target_id,
            reason: report.reason,
            description: report.description,
            status: report.status.to_string(),
            created_at: report.created_at.to_rfc3339(),
            handled_at: report.handled_at.map(|t| t.to_rfc3339()),
        }
    }
}
