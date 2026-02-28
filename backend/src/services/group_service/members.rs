use chrono::{Duration, Utc};

use super::GroupService;
use crate::{
    dto::{
        request::{
            BanMemberRequest, BannedMembersRequest, GroupMembersRequest, InviteMemberRequest,
            LeaveGroupRequest, MuteMemberRequest, RemoveMemberRequest, SearchGroupMembersRequest,
            SetMemberRoleRequest, UnbanMemberRequest, UnmuteMemberRequest,
        },
        response::{BannedMemberItem, GroupMemberItem, InviteMemberData, PaginatedData, UserInfo},
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    types::enums::{GroupRole, MemberLeaveReason, SettableGroupRole},
    utils::validation::Validator,
};

impl GroupService {
    pub async fn search_members(
        &self,
        viewer_id: Option<i64>,
        req: SearchGroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        Validator::validate_keyword(&req.keyword, "keyword")?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as usize;

        let viewer_role = if let Some(user_id) = viewer_id {
            self.member_repo.get_member_role(group_id, user_id).await?
        } else {
            None
        };
        let can_view_admin_fields =
            matches!(viewer_role, Some(GroupRole::Admin | GroupRole::Owner));
        if viewer_role.is_none() {
            let settings = self.settings_repo.find_by_group(group_id).await?;
            let is_searchable = settings.map(|s| s.is_searchable).unwrap_or(true);
            if !is_searchable {
                return Err(AppError::BusinessRuleViolation(
                    "当前群组不允许检索成员".to_string(),
                ));
            }
        }

        let (rows, next_cursor) = self
            .member_repo
            .search_members(
                group_id,
                &req.keyword,
                req.role,
                limit,
                pagination.and_then(|p| p.cursor.as_deref()),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for m in rows {
            items.push(
                self.map_member_detail(viewer_id, can_view_admin_fields, m)
                    .await?,
            );
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_members(
        &self,
        viewer_id: Option<i64>,
        req: GroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as usize;
        let can_view_admin_fields = if let Some(user_id) = viewer_id {
            matches!(
                self.member_repo.get_member_role(group_id, user_id).await?,
                Some(GroupRole::Admin | GroupRole::Owner)
            )
        } else {
            false
        };

        let (rows, next_cursor) = self
            .member_repo
            .list_members(
                group_id,
                req.role,
                limit,
                pagination.and_then(|p| p.cursor.as_deref()),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for m in rows {
            items.push(
                self.map_member_detail(viewer_id, can_view_admin_fields, m)
                    .await?,
            );
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn invite_member(
        &self,
        user_id: i64,
        req: InviteMemberRequest,
    ) -> Result<InviteMemberData> {
        let started_at = std::time::Instant::now();
        if let Some(message) = req.message.as_deref() {
            Validator::validate_invite_or_apply_message(message)?;
        }
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            has_message = req.message.is_some(),
            "group.member invite started"
        );
        let _ = self.require_member(group_id, user_id).await?;

        if self.member_repo.is_member(group_id, target_user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        if let Some(existing) = self
            .invite_repo
            .find_pending_by_group_and_invitee(group_id, target_user_id)
            .await?
        {
            tracing::info!(
                operator_id = user_id,
                target_user_id,
                group_id,
                invite_id = %existing.invite_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "group.member invite completed (reused_pending_invite)"
            );
            return Ok(InviteMemberData {
                invite_id: existing.invite_id.to_string(),
                expires_at: existing
                    .expires_at
                    .unwrap_or(existing.created_at)
                    .to_rfc3339(),
            });
        }

        let invite = self
            .invite_repo
            .create(group_id, user_id, target_user_id, req.message.clone())
            .await?;

        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        let signed_avatar = self
            .media_service
            .sign_url_if_path(group.avatar_url.as_deref())
            .await;
        let members_count = self.member_repo.count_members(group_id).await? as i32;

        self.event_bus.publish(AppEvent::GroupInviteSent {
            invite_id: invite.invite_id,
            group_id,
            group_name: group.name,
            group_avatar_url: signed_avatar,
            members_count,
            from_user_id: user_id,
            to_user_id: target_user_id,
            message: req.message,
        });
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            invite_id = %invite.invite_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member invite completed"
        );

        Ok(InviteMemberData {
            invite_id: invite.invite_id.to_string(),
            expires_at: invite.expires_at.unwrap_or(invite.created_at).to_rfc3339(),
        })
    }

    pub async fn set_role(&self, user_id: i64, req: SetMemberRoleRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            target_role = ?req.role,
            "group.member set_role started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        let target_role = match req.role {
            SettableGroupRole::Admin => GroupRole::Admin,
            SettableGroupRole::Member => GroupRole::Member,
        };

        let target_member = self
            .member_repo
            .find_member(group_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("目标用户不是群成员".to_string()))?;
        let old_role = target_member.role;
        if old_role == target_role {
            tracing::info!(
                operator_id = user_id,
                target_user_id,
                group_id,
                role = ?target_role,
                elapsed_ms = started_at.elapsed().as_millis(),
                "group.member set_role completed (noop)"
            );
            return Ok(());
        }

        self.member_repo
            .update_role(group_id, target_user_id, target_role)
            .await?;
        self.member_history_repo
            .record_role_changed(group_id, target_user_id, user_id, old_role, target_role)
            .await
            .ok();

        self.event_bus.publish(AppEvent::GroupMemberRoleChanged {
            group_id,
            user_id: target_user_id,
            old_role,
            new_role: target_role,
        });
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            old_role = ?old_role,
            new_role = ?target_role,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member set_role completed"
        );

        Ok(())
    }

    pub async fn mute_member(&self, user_id: i64, req: MuteMemberRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        Validator::validate_mute_duration_minutes(req.duration_minutes)?;
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            duration_minutes = req.duration_minutes,
            "group.member mute started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        let muted_until = Utc::now() + Duration::minutes(req.duration_minutes as i64);
        self.member_repo
            .set_mute(group_id, target_user_id, true, Some(muted_until))
            .await?;
        let _ = self
            .member_history_repo
            .record_muted(group_id, target_user_id, user_id, muted_until)
            .await;

        let user_info = self.build_user_info(target_user_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberMuted {
            group_id,
            user_id: target_user_id,
            user_nickname: Some(user_info.nickname.clone()),
            user_avatar_url: user_info.avatar_url.clone(),
            muted_by: user_id,
            muted_until: muted_until.to_rfc3339(),
            duration_minutes: req.duration_minutes,
        });
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            muted_until = %muted_until.to_rfc3339(),
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member mute completed"
        );
        Ok(())
    }

    pub async fn unmute_member(&self, user_id: i64, req: UnmuteMemberRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            "group.member unmute started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        self.member_repo
            .set_mute(group_id, target_user_id, false, None)
            .await?;
        let _ = self
            .member_history_repo
            .record_unmuted(group_id, target_user_id, user_id)
            .await;

        let user_info = self.build_user_info(target_user_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberUnmuted {
            group_id,
            user_id: target_user_id,
            user_nickname: Some(user_info.nickname.clone()),
            user_avatar_url: user_info.avatar_url.clone(),
            unmuted_by: user_id,
        });
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member unmute completed"
        );
        Ok(())
    }

    pub async fn ban_member(
        &self,
        user_id: i64,
        req: BanMemberRequest,
    ) -> Result<BannedMemberItem> {
        let started_at = std::time::Instant::now();
        if let Some(reason) = req.reason.as_deref() {
            Validator::validate_length(reason, "reason", 0, 500)?;
        }
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            has_reason = req.reason.is_some(),
            "group.member ban started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        self.bans_repo
            .ban(group_id, target_user_id, user_id, req.reason.clone())
            .await?;
        let was_member = self.member_repo.is_member(group_id, target_user_id).await?;
        if was_member {
            self.member_repo
                .remove_member(group_id, target_user_id)
                .await?;
            let member_count = self.member_repo.count_members(group_id).await?;
            self.event_bus.publish(AppEvent::GroupMemberRemoved {
                group_id,
                user_id: target_user_id,
                removed_by: user_id,
                member_count,
                reason: MemberLeaveReason::Kick,
            });
        }
        let _ = self
            .member_history_repo
            .record_banned(group_id, target_user_id, user_id, req.reason.clone())
            .await;

        let user_info = self.build_user_info(target_user_id).await?;
        let banned_by_info = self.build_user_info(user_id).await.ok();
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            was_member,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member ban completed"
        );
        Ok(BannedMemberItem {
            user: user_info,
            reason: req.reason,
            banned_at: Utc::now().to_rfc3339(),
            banned_by: banned_by_info,
        })
    }

    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            "group.member unban started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        self.bans_repo
            .unban(group_id, target_user_id, user_id)
            .await?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member unban completed"
        );
        Ok(())
    }

    pub async fn get_banned_list(
        &self,
        user_id: i64,
        req: BannedMembersRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.read")
            .await?;
        let is_admin = true;

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let (rows, next_cursor) = self
            .bans_repo
            .list_by_group(group_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;
        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(r.avatar_url.as_deref())
                .await;
            let banned_by = if is_admin {
                self.build_user_info(r.banned_by).await.ok()
            } else {
                None
            };
            items.push(BannedMemberItem {
                user: UserInfo::new(
                    r.user_id,
                    r.nickname
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                ),
                reason: r.reason,
                banned_at: r.banned_at.to_rfc3339(),
                banned_by,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            "group.member remove started"
        );
        self.require_group_admin_or_system_permission(group_id, user_id, "admin.groups.write")
            .await?;

        self.member_repo
            .remove_member(group_id, target_user_id)
            .await?;
        let _ = self
            .member_history_repo
            .record_removed(group_id, target_user_id, user_id, None)
            .await;

        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id: target_user_id,
            removed_by: user_id,
            member_count,
            reason: MemberLeaveReason::Kick,
        });
        tracing::info!(
            operator_id = user_id,
            target_user_id,
            group_id,
            member_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member remove completed"
        );
        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let group_id = Validator::parse_group_id(&req.group_id)?;
        tracing::info!(user_id, group_id, "group.member leave started");
        let role = self.require_member(group_id, user_id).await?;
        if role == GroupRole::Owner {
            tracing::warn!(
                user_id,
                group_id,
                "group.member leave rejected: owner cannot leave"
            );
            return Err(AppError::BusinessRuleViolation("群主不能退出".to_string()));
        }

        self.member_repo.remove_member(group_id, user_id).await?;
        let _ = self
            .member_history_repo
            .record_leave(group_id, user_id)
            .await;
        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id,
            removed_by: user_id,
            member_count,
            reason: MemberLeaveReason::Quit,
        });
        tracing::info!(
            user_id,
            group_id,
            member_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "group.member leave completed"
        );
        Ok(())
    }
}
