use std::collections::HashMap;

use tokio::try_join;

use super::GroupService;
use crate::{
    dto::{
        request::{
            AcceptInviteRequest, ApproveGroupApplicationRequest, CancelApplicationRequest,
            GroupApplicationsRequest, GroupSentInvitesRequest, JoinGroupRequest,
            MyApplicationsRequest, MyInvitesRequest, RejectGroupApplicationRequest,
            RejectInviteRequest, RevokeSentInviteRequest,
        },
        response::{
            GroupApplicationItem, GroupInfo, GroupInviteItem, GroupSentInviteItem,
            JoinGroupApplication, JoinGroupResult, JoinGroupStatus, PaginatedData, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    types::enums::{
        ApplicationStatus, ApplicationStatusFilter, GroupJoinMethod, GroupRole, ResponseStatus,
    },
    utils::{password::PasswordUtil, validation::Validator},
};

impl GroupService {
    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<JoinGroupResult> {
        let started_at = std::time::Instant::now();
        if let Some(message) = req.message.as_deref() {
            Validator::validate_invite_or_apply_message(message)?;
        }
        if let Some(password) = req.password.as_deref() {
            Validator::validate_group_password(password)?;
        }
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            user_id,
            group_id,
            has_message = req.message.is_some(),
            has_password = req.password.is_some(),
            "group.join started"
        );
        let _group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        if self.bans_repo.is_banned(group_id, user_id).await? {
            return Err(AppError::Forbidden("已被封禁".to_string()));
        }

        let settings = self.settings_repo.find_by_group(group_id).await?;
        let join_method = settings
            .as_ref()
            .map(|s| Self::join_method_from_str(&s.join_method))
            .unwrap_or(GroupJoinMethod::Open);

        if join_method == GroupJoinMethod::InviteOnly {
            return Err(AppError::BusinessRuleViolation(
                "该群组仅限邀请加入".to_string(),
            ));
        }

        if join_method == GroupJoinMethod::Approval {
            if let Some(max) = settings.as_ref().and_then(|s| s.max_members) {
                let count = self.member_repo.count_members(group_id).await? as i32;
                if count >= max {
                    return Err(AppError::GroupFull);
                }
            }
            let app = self
                .application_repo
                .create(group_id, user_id, req.message.clone())
                .await?;
            tracing::info!(
                user_id,
                group_id,
                application_id = %app.application_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "group.join completed (pending_approval)"
            );
            return Ok(JoinGroupResult {
                status: JoinGroupStatus::PendingApproval,
                application: Some(JoinGroupApplication {
                    application_id: app.application_id.to_string(),
                }),
            });
        }

        if let Some(settings) = &settings {
            if let Some(pwd_hash) = settings.password_hash.as_deref() {
                let pwd = req.password.as_deref().unwrap_or("");
                if !PasswordUtil::verify_password(pwd, pwd_hash)? {
                    return Err(AppError::GroupPasswordError("群组密码错误".to_string()));
                }
            }
            if let Some(max) = settings.max_members {
                let count = self.member_repo.count_members(group_id).await? as i32;
                if count >= max {
                    return Err(AppError::GroupFull);
                }
            }
        }

        self.member_repo
            .add_member(group_id, user_id, GroupRole::Member, None)
            .await?;
        let _ = self
            .member_history_repo
            .record_join(group_id, user_id, Some(user_id))
            .await;

        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id,
            user_id,
            added_by: user_id,
            role: GroupRole::Member,
            member_count,
        });
        tracing::info!(
            user_id,
            group_id,
            member_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.join completed (joined)"
        );

        Ok(JoinGroupResult {
            status: JoinGroupStatus::Joined,
            application: None,
        })
    }

    pub async fn get_applications(
        &self,
        user_id: i64,
        req: GroupApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.read")
            .await?;

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
            ApplicationStatusFilter::Revoked => Some(ApplicationStatus::Revoked),
        };
        let (rows, next_cursor) = self
            .application_repo
            .list_by_group(
                group_id,
                status,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let applicant_ids: Vec<i64> = rows.iter().map(|r| r.applicant_id).collect();
        let handler_ids: Vec<i64> = rows.iter().filter_map(|r| r.handled_by).collect();
        let users = self.user_repo.find_profiles(&applicant_ids).await?;
        let handlers = self.user_repo.find_profiles(&handler_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();
        let handler_map: HashMap<i64, _> = handlers.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let applicant = user_map.get(&r.applicant_id);
            let applicant_avatar_url = self
                .media_service
                .sign_url_if_path(applicant.and_then(|u| u.avatar_url.as_deref()))
                .await;
            let applicant_info = UserInfo::new(
                r.applicant_id,
                applicant
                    .and_then(|u| u.nickname.clone())
                    .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                applicant_avatar_url,
            );
            let handled_by_info = if let Some(id) = r.handled_by {
                if let Some(u) = handler_map.get(&id) {
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
            items.push(GroupApplicationItem {
                application_id: r.application_id.to_string(),
                applicant: applicant_info,
                message: r.message,
                status: r.status,
                created_at: r.created_at.to_rfc3339(),
                handled_at: r.handled_at.map(|t| t.to_rfc3339()),
                handled_by: handled_by_info,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_my_applications(
        &self,
        user_id: i64,
        req: MyApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
            ApplicationStatusFilter::Revoked => Some(ApplicationStatus::Revoked),
        };
        let (rows, next_cursor) = self
            .application_repo
            .list_by_applicant(
                user_id,
                status,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let handler_ids: Vec<i64> = rows.iter().filter_map(|r| r.handled_by).collect();
        let (handlers, applicant) = try_join!(
            self.user_repo.find_profiles(&handler_ids),
            self.build_user_info(user_id)
        )?;
        let handler_map: HashMap<i64, _> = handlers.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let handled_by = if let Some(id) = r.handled_by {
                if let Some(u) = handler_map.get(&id) {
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
            items.push(GroupApplicationItem {
                application_id: r.application_id.to_string(),
                applicant: applicant.clone(),
                message: r.message,
                status: r.status,
                created_at: r.created_at.to_rfc3339(),
                handled_at: r.handled_at.map(|t| t.to_rfc3339()),
                handled_by,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub(super) async fn process_application(
        &self,
        operator_id: i64,
        application_id: &str,
        approved: bool,
        reject_reason: Option<&str>,
    ) -> Result<GroupApplicationItem> {
        use chrono::Utc;
        if let Some(reason) = reject_reason {
            Validator::validate_length(reason, "reason", 0, 500)?;
        }

        let app_id = Validator::parse_uuid(application_id, "申请ID")?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;

        self.require_group_admin_or_system_permission(
            application.group_id,
            operator_id,
            "admin.groups.write",
        )
        .await?;

        let already_accepted = approved
            && application.status == ApplicationStatus::Accepted
            && self
                .member_repo
                .is_member(application.group_id, application.applicant_id)
                .await?;

        let status = if approved {
            ApplicationStatus::Accepted
        } else {
            ApplicationStatus::Rejected
        };

        if !already_accepted {
            if let Some(reason) = reject_reason {
                tracing::debug!(reject_reason = %reason, "Application reject reason (repo does not store yet)");
            }

            self.application_repo
                .update_status_by_application_id(app_id, status, operator_id)
                .await?;

            if approved {
                self.member_repo
                    .add_member(
                        application.group_id,
                        application.applicant_id,
                        GroupRole::Member,
                        None,
                    )
                    .await?;
            }
        }

        let group = self
            .group_repo
            .find_by_id(application.group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let signed_avatar = self
            .media_service
            .sign_url_if_path(group.avatar_url.as_deref())
            .await;
        let members_count = self.member_repo.count_members(application.group_id).await? as i32;

        let response_status = if approved {
            ResponseStatus::Accepted
        } else {
            ResponseStatus::Rejected
        };
        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            application_id: app_id,
            group_id: application.group_id,
            group_name: group.name.clone(),
            group_avatar_url: signed_avatar,
            members_count,
            applicant_id: application.applicant_id,
            status: response_status,
        });

        let applicant = self.build_user_info(application.applicant_id).await?;
        Ok(GroupApplicationItem {
            application_id: application.application_id.to_string(),
            applicant,
            message: application.message,
            status,
            created_at: application.created_at.to_rfc3339(),
            handled_at: Some(Utc::now().to_rfc3339()),
            handled_by: Some(self.build_user_info(operator_id).await?),
        })
    }

    pub async fn approve_application(
        &self,
        user_id: i64,
        req: ApproveGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id = user_id,
            application_id = %req.application_id,
            "group.application approve started"
        );
        let result = self
            .process_application(user_id, &req.application_id, true, None)
            .await?;
        tracing::info!(
            operator_id = user_id,
            application_id = %req.application_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.application approve completed"
        );
        Ok(result)
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id = user_id,
            application_id = %req.application_id,
            has_reason = req.reason.is_some(),
            "group.application reject started"
        );
        let result = self
            .process_application(user_id, &req.application_id, false, req.reason.as_deref())
            .await?;
        tracing::info!(
            operator_id = user_id,
            application_id = %req.application_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.application reject completed"
        );
        Ok(result)
    }

    pub async fn cancel_application(
        &self,
        user_id: i64,
        req: CancelApplicationRequest,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        let app_id = Validator::parse_uuid(&req.application_id, "申请ID")?;
        tracing::info!(
            user_id,
            application_id = %app_id,
            "group.application cancel started"
        );
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;
        if application.applicant_id != user_id {
            tracing::warn!(
                user_id,
                application_id = %app_id,
                applicant_id = application.applicant_id,
                "group.application cancel rejected: applicant mismatch"
            );
            return Err(AppError::Forbidden("权限不足".to_string()));
        }
        match application.status {
            ApplicationStatus::Revoked => {
                tracing::info!(
                    user_id,
                    application_id = %app_id,
                    elapsed_ms = started_at.elapsed().as_millis(),
                    "group.application cancel completed (noop_already_revoked)"
                );
                return Ok(());
            }
            ApplicationStatus::Pending => {
                self.application_repo
                    .update_status_by_application_id(app_id, ApplicationStatus::Revoked, user_id)
                    .await?;
            }
            ApplicationStatus::Accepted | ApplicationStatus::Rejected => {
                tracing::warn!(
                    user_id,
                    application_id = %app_id,
                    application_status = ?application.status,
                    "group.application cancel rejected: application already handled"
                );
                return Err(AppError::BusinessRuleViolation(
                    "申请已处理，无法取消".to_string(),
                ));
            }
        }
        tracing::info!(
            user_id,
            application_id = %app_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.application cancel completed"
        );
        Ok(())
    }

    pub async fn list_sent_invites(
        &self,
        user_id: i64,
        req: GroupSentInvitesRequest,
    ) -> Result<PaginatedData<GroupSentInviteItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.read")
            .await?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
            ApplicationStatusFilter::Revoked => Some(ApplicationStatus::Revoked),
        };

        let (rows, next_cursor) = self
            .invite_repo
            .list_by_group(
                group_id,
                status,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;
        let invitee_ids: Vec<i64> = rows.iter().map(|r| r.invitee_id).collect();
        let invitees = self.user_repo.find_profiles(&invitee_ids).await?;
        let invitee_map: HashMap<i64, _> = invitees.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let invitee = invitee_map.get(&row.invitee_id);
            let avatar_url = self
                .media_service
                .sign_url_if_path(invitee.and_then(|u| u.avatar_url.as_deref()))
                .await;
            let nickname = invitee
                .and_then(|u| u.nickname.clone())
                .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string());
            items.push(GroupSentInviteItem {
                invite_id: row.invite_id.to_string(),
                invitee: UserInfo::new(row.invitee_id, nickname, avatar_url),
                message: row.message.clone(),
                status: row.status,
                created_at: row.created_at.to_rfc3339(),
                expires_at: row.expires_at.map(|t| t.to_rfc3339()),
                responded_at: row.responded_at.map(|t| t.to_rfc3339()),
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_invites(
        &self,
        user_id: i64,
        req: MyInvitesRequest,
    ) -> Result<PaginatedData<GroupInviteItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
            ApplicationStatusFilter::Revoked => Some(ApplicationStatus::Revoked),
        };

        let (rows, next_cursor) = self
            .invite_repo
            .list_by_user(
                user_id,
                status,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;
        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let inviter_ids: Vec<i64> = rows.iter().map(|r| r.inviter_id).collect();
        let (groups, users, member_counts) = try_join!(
            self.group_repo.list_by_ids(&group_ids),
            self.user_repo.find_profiles(&inviter_ids),
            self.member_repo.batch_count_members(&group_ids)
        )?;
        let group_map: HashMap<i64, _> = groups.into_iter().map(|g| (g.group_id, g)).collect();
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let group = group_map.get(&r.group_id);
            let inviter = user_map.get(&r.inviter_id);
            let group_avatar_url = self
                .media_service
                .sign_url_if_path(group.and_then(|g| g.avatar_url.as_deref()))
                .await;
            let inviter_avatar_url = self
                .media_service
                .sign_url_if_path(inviter.and_then(|u| u.avatar_url.as_deref()))
                .await;
            items.push(GroupInviteItem {
                invite_id: r.invite_id.to_string(),
                group: GroupInfo {
                    group_id: r.group_id.to_string(),
                    name: group
                        .map(|g| g.name.clone())
                        .unwrap_or_else(|| "群组".to_string()),
                    avatar_url: group_avatar_url,
                    members_count: member_counts.get(&r.group_id).copied().unwrap_or(0) as i32,
                },
                inviter: UserInfo::new(
                    r.inviter_id,
                    inviter
                        .and_then(|u| u.nickname.clone())
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    inviter_avatar_url,
                ),
                message: r.message,
                status: r.status,
                created_at: r.created_at.to_rfc3339(),
                expires_at: r.expires_at.map(|t| t.to_rfc3339()),
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn accept_invite(&self, user_id: i64, req: AcceptInviteRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let invite_id = Validator::parse_uuid(&req.invite_id, "邀请ID")?;
        tracing::info!(
            user_id,
            invite_id = %invite_id,
            "group.invite accept started"
        );
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        if invite.status == ApplicationStatus::Accepted {
            if self.member_repo.is_member(invite.group_id, user_id).await? {
                tracing::info!(
                    user_id,
                    invite_id = %invite_id,
                    group_id = invite.group_id,
                    elapsed_ms = started_at.elapsed().as_millis(),
                    "group.invite accept completed (noop_already_accepted)"
                );
                return Ok(());
            }

            // 兼容中间态：状态已 accepted 但成员记录缺失时补齐
            self.member_repo
                .add_member(
                    invite.group_id,
                    user_id,
                    GroupRole::Member,
                    Some(invite.inviter_id),
                )
                .await?;
            tracing::warn!(
                user_id,
                invite_id = %invite_id,
                group_id = invite.group_id,
                "group.invite accept repaired missing member for accepted invite"
            );
            tracing::info!(
                user_id,
                invite_id = %invite_id,
                group_id = invite.group_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "group.invite accept completed (repaired_missing_member)"
            );
            return Ok(());
        }

        if invite.status != ApplicationStatus::Pending {
            tracing::warn!(
                user_id,
                invite_id = %invite_id,
                group_id = invite.group_id,
                invite_status = ?invite.status,
                "group.invite accept rejected: invite already handled"
            );
            return Err(AppError::BusinessRuleViolation(
                "邀请已处理，无法接受".to_string(),
            ));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, ApplicationStatus::Accepted)
            .await?;
        self.member_repo
            .add_member(
                invite.group_id,
                user_id,
                GroupRole::Member,
                Some(invite.inviter_id),
            )
            .await?;

        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_id,
            inviter_id: invite.inviter_id,
            invitee_id: invite.invitee_id,
            status: ResponseStatus::Accepted,
        });
        tracing::info!(
            user_id,
            invite_id = %invite_id,
            group_id = invite.group_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.invite accept completed"
        );

        Ok(())
    }

    pub async fn reject_invite(&self, user_id: i64, req: RejectInviteRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let invite_id = Validator::parse_uuid(&req.invite_id, "邀请ID")?;
        tracing::info!(
            user_id,
            invite_id = %invite_id,
            "group.invite reject started"
        );
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;
        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        if invite.status == ApplicationStatus::Rejected {
            tracing::info!(
                user_id,
                invite_id = %invite_id,
                group_id = invite.group_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "group.invite reject completed (noop_already_rejected)"
            );
            return Ok(());
        }

        if invite.status != ApplicationStatus::Pending {
            tracing::warn!(
                user_id,
                invite_id = %invite_id,
                group_id = invite.group_id,
                invite_status = ?invite.status,
                "group.invite reject rejected: invite already handled"
            );
            return Err(AppError::BusinessRuleViolation(
                "邀请已处理，无法拒绝".to_string(),
            ));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, ApplicationStatus::Rejected)
            .await?;
        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_id,
            inviter_id: invite.inviter_id,
            invitee_id: invite.invitee_id,
            status: ResponseStatus::Rejected,
        });
        tracing::info!(
            user_id,
            invite_id = %invite_id,
            group_id = invite.group_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.invite reject completed"
        );
        Ok(())
    }

    pub async fn revoke_sent_invite(
        &self,
        user_id: i64,
        req: RevokeSentInviteRequest,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        if let Some(reason) = req.reason.as_deref() {
            Validator::validate_length(reason, "reason", 0, 500)?;
        }
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            user_id,
            group_id,
            has_reason = req.reason.is_some(),
            "group.invite revoke started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;
        let invite_id = Validator::parse_uuid(&req.invite_id, "邀请ID")?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;
        if invite.group_id != group_id {
            tracing::warn!(
                user_id,
                group_id,
                invite_id = %invite_id,
                invite_group_id = invite.group_id,
                "group.invite revoke rejected: invite not in group"
            );
            return Err(AppError::Forbidden("权限不足".to_string()));
        }
        if invite.status != ApplicationStatus::Pending {
            if invite.status == ApplicationStatus::Revoked {
                tracing::info!(
                    user_id,
                    group_id,
                    invite_id = %invite_id,
                    elapsed_ms = started_at.elapsed().as_millis(),
                    "group.invite revoke completed (noop_already_revoked)"
                );
                return Ok(());
            }
            tracing::warn!(
                user_id,
                group_id,
                invite_id = %invite_id,
                invite_status = ?invite.status,
                "group.invite revoke rejected: invite is not pending"
            );
            return Err(AppError::BusinessRuleViolation(
                "仅可撤销等待处理的邀请".to_string(),
            ));
        }

        if !self.invite_repo.revoke_pending_invite(invite_id).await? {
            tracing::warn!(
                user_id,
                group_id,
                invite_id = %invite_id,
                "group.invite revoke rejected: invite status changed"
            );
            return Err(AppError::BusinessRuleViolation(
                "邀请状态已变更，无法撤销".to_string(),
            ));
        }

        if let Some(reason) = &req.reason {
            tracing::debug!(reason = %reason, "Sent invite revoked");
        }
        tracing::info!(
            user_id,
            group_id,
            invite_id = %invite_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.invite revoke completed"
        );
        Ok(())
    }
}
