use std::str::FromStr;

use crate::{
    dto::{
        request::{
            AdminReportHandleRequest, AdminReportSearchRequest, CreateReportRequest,
            MyReportsRequest,
        },
        response::{AdminReportItem, CreateReportData, PaginatedData, ReportItem, UserInfo},
    },
    error::AppError,
    repositories::{
        comment_repo::CommentRepository,
        group::message_repo::GroupMessageRepository,
        group_repo::GroupRepository,
        moment_repo::MomentRepository,
        private_message_repo::PrivateMessageRepository,
        report_repo::{ReportRepository, ReportRow},
        user_repo::UserRepository,
    },
    types::enums::{ReportReason, ReportStatus, ReportTargetType},
    utils::validation::Validator,
};
use tracing::{info, warn};

#[derive(Clone)]
pub struct ReportService {
    report_repo: ReportRepository,
    user_repo: UserRepository,
    moment_repo: MomentRepository,
    comment_repo: CommentRepository,
    group_repo: GroupRepository,
    private_message_repo: PrivateMessageRepository,
    group_message_repo: GroupMessageRepository,
    media_service: crate::services::media_service::MediaService,
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
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            report_repo,
            user_repo,
            moment_repo,
            comment_repo,
            group_repo,
            private_message_repo,
            group_message_repo,
            media_service,
        }
    }

    /// 提交举报
    pub async fn create_report(
        &self,
        reporter_id: i64,
        req: CreateReportRequest,
    ) -> Result<CreateReportData, AppError> {
        let started_at = std::time::Instant::now();
        info!(
            reporter_id,
            target_type = ?req.target_type,
            reason = ?req.reason,
            has_description = req.description.is_some(),
            "report.create started"
        );
        // target_type 和 reason 已是强类型枚举

        let reason_str = req.reason.to_string();

        let target_type = req.target_type;
        let target_id = normalize_report_target_id(target_type, req.target_id)?;
        self.validate_target_exists(target_type, &target_id).await?;
        self.check_self_report(reporter_id, target_type, &target_id)
            .await?;
        if self
            .report_repo
            .has_pending_or_reviewing_for_target(target_type, &target_id)
            .await?
        {
            warn!(
                reporter_id,
                target_type = ?target_type,
                target_id = %target_id,
                "report.create rejected: target already has pending/reviewing report"
            );
            return Err(AppError::BusinessRuleViolation(
                "该目标已有待处理举报，请勿重复提交".to_string(),
            ));
        }

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
        info!(
            reporter_id,
            report_id = %created_report.report_id,
            target_type = ?target_type,
            target_id = %target_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "report.create completed"
        );

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
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )?;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        let (reports, next_cursor) = self
            .report_repo
            .find_reports_by_reporter(
                reporter_id,
                req.status,
                req.target_type,
                limit as i32,
                cursor,
            )
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
        req: &AdminReportSearchRequest,
    ) -> Result<PaginatedData<AdminReportItem>, AppError> {
        let pagination = req.pagination.as_ref();
        let limit_value = pagination.and_then(|p| p.limit).unwrap_or(20);
        let limit = Validator::validate_limit(Some(limit_value), crate::constants::MAX_PAGE_LIMIT)?;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        let (reports, next_cursor) = self
            .report_repo
            .find_all_reports(req.status, req.target_type, limit as i32, cursor)
            .await?;

        let mut all_user_ids: Vec<i64> = reports.iter().map(|r| r.reporter_id).collect();
        for r in &reports {
            if let Some(uid) = r.handled_by {
                all_user_ids.push(uid);
            }
        }
        all_user_ids.sort_unstable();
        all_user_ids.dedup();

        let users = self
            .user_repo
            .find_profiles(&all_user_ids)
            .await
            .unwrap_or_default();
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let mut response_items = Vec::with_capacity(reports.len());
        for report in reports {
            let report_item = Self::to_report_item(report.clone())?;
            let target_snapshot = report.target_snapshot.and_then(|value| {
                // 严格按最新结构化快照读取，不再兼容历史扁平格式
                if value.get("moment").is_some()
                    || value.get("comment").is_some()
                    || value.get("message").is_some()
                {
                    return serde_json::from_value(value).ok();
                }
                None
            });
            let reporter = if let Some(u) = user_map.get(&report.reporter_id) {
                let avatar_url = self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await;
                UserInfo::new(
                    u.user_id,
                    u.nickname
                        .clone()
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                )
            } else {
                UserInfo::new(
                    report.reporter_id,
                    crate::constants::UNKNOWN_USER_NICKNAME,
                    None,
                )
            };
            let handled_by = if let Some(uid) = report.handled_by {
                if let Some(u) = user_map.get(&uid) {
                    let avatar_url = self
                        .media_service
                        .sign_url_if_path(u.avatar_url.as_deref())
                        .await;
                    Some(UserInfo::new(
                        u.user_id,
                        u.nickname
                            .clone()
                            .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                        avatar_url,
                    ))
                } else {
                    None
                }
            } else {
                None
            };
            response_items.push(AdminReportItem {
                report: report_item,
                reporter_user_id: report.reporter_id,
                reporter,
                handled_by,
                handled_action: report.handled_action,
                handled_note: report.handled_note,
                target_snapshot,
            });
        }

        Ok(PaginatedData {
            items: response_items,
            next_cursor,
        })
    }

    /// 审核举报（管理员）
    pub async fn handle(
        &self,
        admin_id: i64,
        req: AdminReportHandleRequest,
    ) -> Result<(), AppError> {
        use crate::types::enums::ReportHandleAction;
        let started_at = std::time::Instant::now();
        info!(
            admin_id,
            report_id = %req.report_id,
            action = ?req.action,
            has_note = req.note.is_some(),
            "report.handle started"
        );

        // action 已是强类型枚举

        // 解析report_id
        let report_id = Validator::parse_uuid(&req.report_id, "举报ID")?;

        // 仅做存在性校验（避免拉整行）
        if !self.report_repo.exists(report_id).await? {
            warn!(admin_id, report_id = %report_id, "report.handle rejected: report not found");
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
        info!(
            admin_id,
            report_id = %report_id,
            status = ?status,
            elapsed_ms = started_at.elapsed().as_millis(),
            "report.handle completed"
        );

        Ok(())
    }

    /// 辅助方法：将ReportRow转换为ReportItem
    fn to_report_item(report: ReportRow) -> Result<ReportItem, AppError> {
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
        })
    }
}

impl ReportService {
    async fn check_self_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: &str,
    ) -> Result<(), AppError> {
        let is_self = match target_type {
            ReportTargetType::User => {
                let uid = crate::utils::validation::Validator::parse_user_id(target_id)?;
                uid == reporter_id
            }
            ReportTargetType::Moment => {
                let mid = Validator::parse_uuid(target_id, "target_id")?;
                self.moment_repo
                    .find_by_id(mid)
                    .await?
                    .map(|m| m.user_id == reporter_id)
                    .unwrap_or(false)
            }
            ReportTargetType::Comment => {
                let cid = Validator::parse_uuid(target_id, "target_id")?;
                self.comment_repo
                    .find_by_id(cid)
                    .await?
                    .map(|c| c.user_id == reporter_id)
                    .unwrap_or(false)
            }
            ReportTargetType::Group => {
                let gid = crate::utils::validation::Validator::parse_group_id(target_id)?;
                self.group_repo
                    .find_by_id(gid)
                    .await?
                    .map(|g| g.owner_id == reporter_id)
                    .unwrap_or(false)
            }
            ReportTargetType::Message => {
                let mid = Validator::parse_uuid(target_id, "target_id")?;
                let is_private_self = self
                    .private_message_repo
                    .find_by_id(mid)
                    .await?
                    .map(|m| m.sender_id == reporter_id)
                    .unwrap_or(false);
                if is_private_self {
                    true
                } else {
                    self.group_message_repo
                        .find_by_id(mid)
                        .await?
                        .map(|m| m.sender_id == reporter_id)
                        .unwrap_or(false)
                }
            }
        };
        if is_self {
            return Err(AppError::BusinessRuleViolation(
                "不能举报自己的内容".to_string(),
            ));
        }
        Ok(())
    }

    async fn validate_target_exists(
        &self,
        target_type: ReportTargetType,
        target_id: &str,
    ) -> Result<(), AppError> {
        match target_type {
            ReportTargetType::User => {
                let user_id = crate::utils::validation::Validator::parse_user_id(target_id)?;
                if !self.user_repo.exists(user_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Moment => {
                let moment_id = Validator::parse_uuid(target_id, "target_id")?;
                if !self.moment_repo.exists(moment_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Comment => {
                let comment_id = Validator::parse_uuid(target_id, "target_id")?;
                if !self.comment_repo.exists(comment_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Group => {
                let group_id = crate::utils::validation::Validator::parse_group_id(target_id)?;
                if !self.group_repo.exists(group_id).await? {
                    return Err(AppError::NotFound("举报目标不存在".to_string()));
                }
            }
            ReportTargetType::Message => {
                let message_id = Validator::parse_uuid(target_id, "target_id")?;
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
