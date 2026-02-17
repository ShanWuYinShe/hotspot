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
    types::enums::{GroupRole, MemberLeaveReason, SetableGroupRole},
    utils::validation::Validator,
};

impl GroupService {
    pub async fn search_members(
        &self,
        user_id: i64,
        req: SearchGroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        Validator::validate_not_empty(&req.keyword, "keyword")?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_member(group_id, user_id).await?;
        let limit = Validator::validate_limit(req.limit, 50)? as usize;

        let (rows, next_cursor) = self
            .member_repo
            .search_members(
                group_id,
                &req.keyword,
                req.role,
                limit,
                req.cursor.as_deref(),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for m in rows {
            items.push(self.map_member_detail(m).await?);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_members(
        &self,
        req: GroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let limit =
            Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as usize;

        if let Some(ref role) = req.role {
            tracing::debug!(role = ?role, "List members role filter (repo does not support yet)");
        }

        let (rows, next_cursor) = self
            .member_repo
            .list_members(group_id, limit, req.cursor.as_deref())
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for m in rows {
            items.push(self.map_member_detail(m).await?);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn invite_member(
        &self,
        user_id: i64,
        req: InviteMemberRequest,
    ) -> Result<InviteMemberData> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_member(group_id, user_id).await?;

        if self.member_repo.is_member(group_id, target_user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        if let Some(existing) = self
            .invite_repo
            .find_pending_by_group_and_invitee(group_id, target_user_id)
            .await?
        {
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

        Ok(InviteMemberData {
            invite_id: invite.invite_id.to_string(),
            expires_at: invite.expires_at.unwrap_or(invite.created_at).to_rfc3339(),
        })
    }

    pub async fn set_role(
        &self,
        user_id: i64,
        req: SetMemberRoleRequest,
    ) -> Result<GroupMemberItem> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_owner(group_id, user_id).await?;

        let target_role = match req.role {
            SetableGroupRole::Admin => GroupRole::Admin,
            SetableGroupRole::Member => GroupRole::Member,
        };

        self.member_repo
            .update_role(group_id, target_user_id, target_role)
            .await?;
        self.member_history_repo
            .record_role_changed(
                group_id,
                target_user_id,
                user_id,
                GroupRole::Member,
                target_role,
            )
            .await
            .ok();

        self.event_bus.publish(AppEvent::GroupMemberRoleChanged {
            group_id,
            user_id: target_user_id,
            old_role: GroupRole::Member,
            new_role: target_role,
        });

        let user_info = self.build_user_info(target_user_id).await?;
        Ok(GroupMemberItem {
            user: user_info,
            role: target_role,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: None,
            presence_status: None,
            last_online: None,
        })
    }

    pub async fn mute_member(
        &self,
        user_id: i64,
        req: MuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

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
        Ok(GroupMemberItem {
            user: user_info,
            role: GroupRole::Member,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: Some(muted_until.to_rfc3339()),
            presence_status: None,
            last_online: None,
        })
    }

    pub async fn unmute_member(
        &self,
        user_id: i64,
        req: UnmuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

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
        Ok(GroupMemberItem {
            user: user_info,
            role: GroupRole::Member,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: None,
            presence_status: None,
            last_online: None,
        })
    }

    pub async fn ban_member(
        &self,
        user_id: i64,
        req: BanMemberRequest,
    ) -> Result<BannedMemberItem> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

        self.bans_repo
            .ban(group_id, target_user_id, user_id, req.reason.clone())
            .await?;
        let _ = self
            .member_history_repo
            .record_banned(group_id, target_user_id, user_id, req.reason.clone())
            .await;

        let user_info = self.build_user_info(target_user_id).await?;
        let banned_by_info = self.build_user_info(user_id).await.ok();
        Ok(BannedMemberItem {
            user: user_info,
            reason: req.reason,
            banned_at: Utc::now().to_rfc3339(),
            banned_by: banned_by_info,
        })
    }

    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

        self.bans_repo
            .unban(group_id, target_user_id, user_id)
            .await?;
        Ok(())
    }

    pub async fn get_banned_list(
        &self,
        user_id: i64,
        req: BannedMembersRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_member(group_id, user_id).await?;

        // spec: banned_by 仅对群管理员/群主可见 (@group-admin-only)
        let is_admin = self.require_admin_or_owner(group_id, user_id).await.is_ok();

        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i32;
        let (rows, next_cursor) = self
            .bans_repo
            .list_by_group(group_id, limit, req.cursor)
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
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

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
        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let role = self.require_member(group_id, user_id).await?;
        if role == GroupRole::Owner {
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
        Ok(())
    }
}
