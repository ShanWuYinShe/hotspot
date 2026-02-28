use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AcceptInviteRequest, AdminDismissGroupRequest, AdminGroupDetailRequest,
            AdminGroupSearchRequest, ApproveGroupApplicationRequest, BanMemberRequest,
            BannedMembersRequest, CancelApplicationRequest, CreateGroupAnnouncementRequest,
            CreateGroupChatRequest, DeleteGroupAnnouncementRequest, GroupAnnouncementsRequest,
            GroupApplicationsRequest, GroupMembersRequest, GroupOpenTheDoorRequest,
            GroupSentInvitesRequest, InviteMemberRequest, JoinGroupRequest, LeaveGroupRequest,
            MuteMemberRequest, MyApplicationsRequest, MyGroupChatsRequest, MyInvitesRequest,
            RejectGroupApplicationRequest, RejectInviteRequest, RemoveMemberRequest,
            RevokeSentInviteRequest, SearchGroupMembersRequest, SearchGroupsRequest,
            SetMemberRoleRequest, TransferOwnershipRequest, UnbanMemberRequest,
            UnmuteMemberRequest, UpdateGroupAnnouncementRequest, UpdateGroupChatRequest,
            UpdateGroupMySettingsRequest,
        },
        response::{
            AdminGroupItem, ApiResponse, BannedMemberItem, GroupAnnouncementItem,
            GroupApplicationItem, GroupInviteItem, GroupItem, GroupMemberItem, GroupSentInviteItem,
            InviteMemberData, JoinGroupResult, PaginatedData,
        },
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

// ==================== Token 可选 ====================

/// 统一搜索群组（支持关键字/标签/位置组合）
pub async fn search(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let started_at = std::time::Instant::now();
    let sort_by = req.sort_by.as_deref().unwrap_or("relevance");
    tracing::debug!(sort_by = %sort_by, "Search sort parameter");
    tracing::info!(
        has_keyword = req.keyword.is_some(),
        has_tag = req.tag.is_some(),
        radius = req.radius.unwrap_or(5000.0),
        has_cursor = req
            .pagination
            .as_ref()
            .and_then(|p| p.cursor.as_ref())
            .is_some(),
        limit = req.pagination.as_ref().and_then(|p| p.limit).unwrap_or(20),
        "groups.search request received"
    );

    let response = state.group_service.search(&req).await?;
    tracing::info!(
        items = response.items.len(),
        has_next_cursor = response.next_cursor.is_some(),
        elapsed_ms = started_at.elapsed().as_millis(),
        "groups.search request completed"
    );
    Ok(Json(ApiResponse::success(response)))
}

/// 搜索群组成员
pub async fn search_members(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchGroupMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMemberItem>>>> {
    let response = state
        .group_service
        .search_members(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 我的入群申请列表
pub async fn my_applications(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyApplicationsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupApplicationItem>>>> {
    let response = state
        .group_service
        .get_my_applications(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取群组详情（通过 ID 获取指定群组的详情）
///
/// 支持可选认证：
/// - 未登录：返回基础群组信息
/// - 已登录：额外返回 my_state（Zero-Lookup）
pub async fn open_the_door(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<GroupOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<GroupItem>>> {
    tracing::info!(
        "open_the_door: group_id={}, user_id={:?}",
        req.group_id,
        user.user_id
    );

    let response = state
        .group_service
        .open_the_door(&req, user.user_id)
        .await?;

    tracing::info!(
        "open_the_door response: has_my_state={}",
        response.my_state.is_some()
    );

    Ok(Json(ApiResponse::success(response)))
}

/// 获取成员列表
pub async fn members(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<GroupMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMemberItem>>>> {
    let response = state.group_service.get_members(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Token 必须 ====================

pub async fn create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateGroupChatRequest>,
) -> Result<Json<ApiResponse<GroupItem>>> {
    let created = state.group_service.create_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(created)))
}

pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyGroupChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let response = state.group_service.get_joined(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn update(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.update_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn delete(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<crate::dto::request::DeleteGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.delete_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn transfer_ownership(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .transfer_ownership(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 成员操作 ====================

pub async fn invite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<InviteMemberRequest>,
) -> Result<Json<ApiResponse<InviteMemberData>>> {
    let response = state.group_service.invite_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn join(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<JoinGroupResult>>> {
    let result = state.group_service.join_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn set_role(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<SetMemberRoleRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.set_role(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn mute(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.mute_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unmute(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnmuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.unmute_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn ban(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<BanMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.ban_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unban(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnbanMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.unban_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn banned(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<BannedMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<BannedMemberItem>>>> {
    let result = state
        .group_service
        .get_banned_list(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn remove_member(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RemoveMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.remove_member(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn leave_group(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.leave_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 申请管理 ====================

pub async fn applications(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<GroupApplicationsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupApplicationItem>>>> {
    let response = state
        .group_service
        .get_applications(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn approve_application(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<ApproveGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .approve_application(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn reject_application(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RejectGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .reject_application(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn cancel_application(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CancelApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .cancel_application(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 邀请管理 ====================

pub async fn list_sent_invites(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<GroupSentInvitesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupSentInviteItem>>>> {
    let response = state
        .group_service
        .list_sent_invites(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn revoke_sent_invite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RevokeSentInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .revoke_sent_invite(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn my_invites(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyInvitesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupInviteItem>>>> {
    let response = state.group_service.get_invites(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn accept_invite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AcceptInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.accept_invite(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn reject_invite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RejectInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.reject_invite(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 管理员群组管理 (Admin) ====================

pub async fn admin_search(
    State(state): State<AppState>,
    Json(req): Json<AdminGroupSearchRequest>,
) -> Result<Json<ApiResponse<PaginatedData<AdminGroupItem>>>> {
    let response = state.group_service.admin_search_groups(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn admin_detail(
    State(state): State<AppState>,
    _admin: AuthenticatedUser,
    Json(req): Json<AdminGroupDetailRequest>,
) -> Result<Json<ApiResponse<AdminGroupItem>>> {
    let detail = state.group_service.admin_group_detail(&req).await?;
    Ok(Json(ApiResponse::success(detail)))
}

pub async fn admin_dismiss(
    State(state): State<AppState>,
    admin: AuthenticatedUser,
    Json(req): Json<AdminDismissGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .admin_dismiss_group(admin.user_id, &req)
        .await?;
    let details = serde_json::json!({
        "reason": req.reason,
    });
    state
        .admin_audit_service
        .record(AuditLogInput::group_action(
            admin.user_id,
            "group.dismiss",
            &req.group_id,
            Some(details),
        ))
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 群公告 APIs ====================

/// 发布群公告
pub async fn create_announcement(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateGroupAnnouncementRequest>,
) -> Result<Json<ApiResponse<GroupAnnouncementItem>>> {
    let response = state
        .group_service
        .create_announcement(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取群公告列表
pub async fn list_announcements(
    State(state): State<AppState>,
    Json(req): Json<GroupAnnouncementsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupAnnouncementItem>>>> {
    let response = state.group_service.list_announcements(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 更新群公告
pub async fn update_announcement(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateGroupAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .update_announcement(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 删除群公告
pub async fn delete_announcement(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DeleteGroupAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .delete_announcement(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 更新我的群组偏好设置
pub async fn update_my_settings(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateGroupMySettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .update_my_settings(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
