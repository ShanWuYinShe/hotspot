use std::str::FromStr;
use uuid::Uuid;

use crate::{
    dto::{
        request::{CreateReportRequest, HandleReportRequest, MyReportsRequest, ReportsRequest},
        response::{CreateReportData, PaginatedData, ReportItem},
    },
    error::AppError,
    repositories::{
        comment_repo::CommentRepository,
        group_message_repo::GroupMessageRepository,
        group_repo::GroupRepository,
        moment_repo::MomentRepository,
        private_message_repo::PrivateMessageRepository,
        report_repo::{ReportRepository, ReportRow},
        user_repo::UserRepository,
    },
    types::enums::{ReportReason, ReportStatus, ReportTargetType},
    utils::validation::Validator,
};

#[derive(Debug, Clone)]
pub struct ReportService {
    report_repo: ReportRepository,
    user_repo: UserRepository,
    moment_repo: MomentRepository,
    comment_repo: CommentRepository,
    group_repo: GroupRepository,
    private_message_repo: PrivateMessageRepository,
    group_message_repo: GroupMessageRepository,
}

impl ReportService {
    pub fn new(
        report_repo: ReportRepository,
        user_repo: UserRepository,
        moment_repo: MomentRepository,
        comment_repo: CommentRepository,
        group_repo: GroupRepository,
        private_message_repo: PrivateMessageRepository,
        group_message_repo: GroupMessageRepository,
    ) -> Self {
        Self {
            report_repo,
            user_repo,
            moment_repo,
            comment_repo,
            group_repo,
            private_message_repo,
            group_message_repo,
        }
    }

    /// 提交举报
    pub async fn create_report(
        &self,
        reporter_id: i64,
        req: CreateReportRequest,
    ) -> Result<CreateReportData, AppError> {
        // target_type 和 reason 已是强类型枚举

        let reason_str = req.reason.to_string();

        let target_type = req.target_type;
        let target_id = normalize_report_target_id(target_type, req.target_id)?;
        self.validate_target_exists(target_type, &target_id).await?;

        let created_report = self
            .report_repo
            .create(
                reporter_id,
                target_type,
                &target_id,
                &reason_str,
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

        let response_items: Vec<ReportItem> = reports
            .into_iter()
            .map(Self::to_report_item)
            .collect::<Result<Vec<_>, _>>()?;

        Ok(PaginatedData {
            items: response_items,
            next_cursor,
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

        let response_items: Vec<ReportItem> = reports
            .into_iter()
            .map(Self::to_report_item)
            .collect::<Result<Vec<_>, _>>()?;

        Ok(PaginatedData {
            items: response_items,
            next_cursor,
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
            .update_status(report_id, status, req.action, req.note.clone(), admin_id)
            .await?;

        Ok(())
    }

    /// 辅助方法：将ReportRow转换为ReportItem
    fn to_report_item(report: ReportRow) -> Result<ReportItem, AppError> {
        use crate::dto::response::ReportTargetSnapshot;

        let target_snapshot = report.target_snapshot.and_then(|value| {
            let content = value
                .get("content")
                .and_then(|v| v.as_str())
                .map(|v| v.to_string());
            let media_urls = value.get("media_urls").and_then(|v| {
                v.as_array().map(|arr| {
                    arr.iter()
                        .filter_map(|item| item.as_str().map(|s| s.to_string()))
                        .collect::<Vec<_>>()
                })
            });
            let owner_id = value.get("owner_id").and_then(|v| {
                v.as_str()
                    .map(|s| s.to_string())
                    .or_else(|| v.as_i64().map(|id| id.to_string()))
            });

            if content.is_none() && media_urls.is_none() && owner_id.is_none() {
                None
            } else {
                Some(ReportTargetSnapshot {
                    content,
                    media_urls,
                    owner_id,
                })
            }
        });

        let reason = ReportReason::from_str(&report.reason).unwrap_or(ReportReason::Other);

        Ok(ReportItem {
            report_id: report.report_id.to_string(),
            target_type: report.target_type,
            target_id: report.target_id.clone(),
            reason,
            description: report.description,
            status: report.status,
            created_at: report.created_at.to_rfc3339(),
            handled_at: report.handled_at.map(|t| t.to_rfc3339()),
            target_snapshot,
        })
    }
}

impl ReportService {
    async fn validate_target_exists(
        &self,
        target_type: ReportTargetType,
        target_id: &str,
    ) -> Result<(), AppError> {
        match target_type {
            ReportTargetType::User => {
                let user_id = target_id
                    .parse::<i64>()
                    .map_err(|_| AppError::InvalidFormat("target_id 无效".to_string()))?;
                if !self.user_repo.exists(user_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Moment => {
                let moment_id = uuid::Uuid::parse_str(target_id)
                    .map_err(|_| AppError::InvalidFormat("target_id 无效".to_string()))?;
                if !self.moment_repo.exists(moment_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Comment => {
                let comment_id = uuid::Uuid::parse_str(target_id)
                    .map_err(|_| AppError::InvalidFormat("target_id 无效".to_string()))?;
                if !self.comment_repo.exists(comment_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Group => {
                let group_id = target_id
                    .parse::<i64>()
                    .map_err(|_| AppError::InvalidFormat("target_id 无效".to_string()))?;
                if !self.group_repo.exists(group_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Message => {
                let message_id = uuid::Uuid::parse_str(target_id)
                    .map_err(|_| AppError::InvalidFormat("target_id 无效".to_string()))?;
                let private_exists = self
                    .private_message_repo
                    .find_by_id(message_id)
                    .await?
                    .is_some();
                if private_exists {
                    return Ok(());
                }
                let group_exists = self
                    .group_message_repo
                    .find_by_id(message_id)
                    .await?
                    .is_some();
                if !group_exists {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
        }
        Ok(())
    }
}

/// 规范化 target_id（统一为 string，验证非空）
fn normalize_report_target_id(
    _target_type: crate::types::enums::ReportTargetType,
    target_id: String,
) -> Result<String, AppError> {
    if target_id.trim().is_empty() {
        return Err(AppError::InvalidFormat("target_id 不能为空".to_string()));
    }
    Ok(target_id)
}
