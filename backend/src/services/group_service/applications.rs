use std::collections::HashMap;

use tokio::try_join;

use super::GroupService;
use crate::{
    dto::{
        request::{
            AcceptInviteRequest, ApproveGroupApplicationRequest, CancelApplicationRequest,
            GroupApplicationsRequest, JoinGroupRequest, MyApplicationsRequest, MyInvitesRequest,
            RejectGroupApplicationRequest, RejectInviteRequest,
        },
        response::{
            GroupApplicationItem, GroupInfo, GroupInviteItem, JoinGroupApplication,
            JoinGroupResult, JoinGroupStatus, MyApplicationItem, PaginatedData, UserInfo,
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
        let group_id = Validator::parse_group_id(&req.group_id)?;
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
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
        };
        let (rows, next_cursor) = self
            .application_repo
            .list_by_group(group_id, status, limit, req.cursor)
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
    ) -> Result<PaginatedData<MyApplicationItem>> {
        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
        };
        let (rows, next_cursor) = self
            .application_repo
            .list_by_applicant(user_id, status, limit, req.cursor)
            .await?;

        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let handler_ids: Vec<i64> = rows.iter().filter_map(|r| r.handled_by).collect();
        let (groups, handlers, member_counts) = try_join!(
            self.group_repo.list_by_ids(&group_ids),
            self.user_repo.find_profiles(&handler_ids),
            self.member_repo.batch_count_members(&group_ids)
        )?;
        let group_map: HashMap<i64, _> = groups.into_iter().map(|g| (g.group_id, g)).collect();
        let handler_map: HashMap<i64, _> = handlers.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let group = group_map.get(&r.group_id);
            let members_count = member_counts.get(&r.group_id).copied().unwrap_or(0) as i32;
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
            let avatar_url = self
                .media_service
                .sign_url_if_path(group.and_then(|g| g.avatar_url.as_deref()))
                .await;
            items.push(MyApplicationItem {
                application_id: r.application_id.to_string(),
                group: GroupInfo {
                    group_id: r.group_id.to_string(),
                    name: group
                        .map(|g| g.name.clone())
                        .unwrap_or_else(|| "群组".to_string()),
                    avatar_url,
                    members_count,
                },
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

        let app_id = Validator::parse_uuid(application_id, "申请ID")?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;

        let _ = self
            .require_admin_or_owner(application.group_id, operator_id)
            .await?;

        if let Some(reason) = reject_reason {
            tracing::debug!(reject_reason = %reason, "Application reject reason (repo does not store yet)");
        }

        let status = if approved {
            ApplicationStatus::Accepted
        } else {
            ApplicationStatus::Rejected
        };
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
        self.process_application(user_id, &req.application_id, true, None)
            .await
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        self.process_application(user_id, &req.application_id, false, req.reason.as_deref())
            .await
    }

    pub async fn cancel_application(
        &self,
        user_id: i64,
        req: CancelApplicationRequest,
    ) -> Result<()> {
        let app_id = Validator::parse_uuid(&req.application_id, "申请ID")?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;
        if application.applicant_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }
        self.application_repo
            .delete_by_application_id(app_id)
            .await?;
        Ok(())
    }

    pub async fn get_invites(
        &self,
        user_id: i64,
        req: MyInvitesRequest,
    ) -> Result<PaginatedData<GroupInviteItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i32;
        let status = req.status.map(|s| match s {
            ApplicationStatusFilter::Pending => ApplicationStatus::Pending,
            ApplicationStatusFilter::Accepted => ApplicationStatus::Accepted,
            ApplicationStatusFilter::Rejected => ApplicationStatus::Rejected,
            ApplicationStatusFilter::All => ApplicationStatus::Pending,
        });

        let (rows, next_cursor) = self
            .invite_repo
            .list_by_user(user_id, status, limit, req.cursor)
            .await?;
        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let inviter_ids: Vec<i64> = rows.iter().map(|r| r.inviter_id).collect();
        let (groups, users) = try_join!(
            self.group_repo.list_by_ids(&group_ids),
            self.user_repo.find_profiles(&inviter_ids)
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
                    members_count: group.map(|g| g.max_members).unwrap_or(0),
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
        let invite_id = Validator::parse_uuid(&req.invite_id, "邀请ID")?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
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

        Ok(())
    }

    pub async fn reject_invite(&self, user_id: i64, req: RejectInviteRequest) -> Result<()> {
        let invite_id = Validator::parse_uuid(&req.invite_id, "邀请ID")?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;
        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
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
        Ok(())
    }
}
