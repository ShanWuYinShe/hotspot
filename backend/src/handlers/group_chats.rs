use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AcceptInviteRequest, AdminDismissGroupRequest, AdminGroupSearchRequest,
            ApplyToGroupRequest, ApproveGroupApplicationRequest, BanMemberRequest,
            BannedMembersRequest, CancelApplicationRequest, CreateGroupAnnouncementRequest,
            CreateGroupChatRequest, DeleteGroupAnnouncementRequest, GroupAnnouncementsRequest,
            GroupApplicationsRequest, GroupMembersRequest, GroupOpenTheDoorRequest,
            GroupSettingsRequest, InviteMemberRequest, JoinGroupRequest, LeaveGroupRequest,
            MuteMemberRequest, MyGroupChatsRequest, MyInvitesRequest,
            RejectGroupApplicationRequest, RejectInviteRequest, RemoveMemberRequest,
            SearchGroupsByKeywordRequest, SearchGroupsByLocationRequest, SearchGroupsByTagRequest,
            SetMemberRoleRequest, TransferOwnershipRequest, UnbanMemberRequest,
            UnmuteMemberRequest, UpdateGroupAnnouncementRequest, UpdateGroupChatRequest,
        },
        response::{
            AdminGroupItem, ApiResponse, ApplyToGroupData, BannedMemberItem, CreateGroupChatData,
            GroupAnnouncementItem, GroupApplicationItem, GroupDetailData, GroupInviteItem,
            GroupItem, GroupMemberItem, GroupSettingsData, InviteMemberData, JoinedGroupItem,
            PaginatedData,
        },
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

// ==================== Public APIs ====================

/// 按关键字搜索群组（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let response = state.group_service.search_by_keyword(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索群组（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let response = state.group_service.search_by_tag(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索群组（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let response = state.group_service.search_by_location(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取群组详情（通过 ID 获取指定群组的详情）
///
/// 支持可选认证：
/// - 未登录：返回基础群组信息
/// - 已登录：额外返回 is_joined, my_role 等字段（Zero-Lookup）
pub async fn open_the_door(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<GroupOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<GroupDetailData>>> {
    // 提取 Authorization header
    let auth_header = headers.get("Authorization");
    tracing::info!("Authorization header exists: {}", auth_header.is_some());

    // 尝试从 Authorization header 提取用户ID（可选认证）
    let user_id = auth_header
        .and_then(|h| {
            let header_str = h.to_str().ok()?;
            tracing::info!("Authorization header value: {}", header_str);
            Some(header_str)
        })
        .and_then(|h| {
            let token = h.strip_prefix("Bearer ")?;
            tracing::info!(
                "Extracted token (first 20 chars): {}...",
                &token[..token.len().min(20)]
            );
            Some(token)
        })
        .and_then(|token| {
            match crate::utils::jwt::JwtUtil::verify_token(token, &state.config.jwt_secret) {
                Ok(claims) => {
                    tracing::info!(
                        "JWT verified successfully: sub={}, sid={}",
                        claims.sub,
                        claims.sid
                    );
                    Some(claims)
                }
                Err(e) => {
                    tracing::error!("JWT verification failed: {:?}", e);
                    None
                }
            }
        })
        .and_then(|claims| match claims.sub.parse::<i64>() {
            Ok(uid) => {
                tracing::info!("Parsed user_id: {}", uid);
                Some(uid)
            }
            Err(e) => {
                tracing::error!("Failed to parse user_id from sub '{}': {:?}", claims.sub, e);
                None
            }
        });

    tracing::info!(
        "open_the_door: group_id={}, user_id={:?}, has_auth_header={}",
        req.group_chat_id,
        user_id,
        auth_header.is_some()
    );

    let response = state.group_service.open_the_door(&req, user_id).await?;

    tracing::info!(
        "open_the_door response: is_joined={:?}, my_role={:?}",
        response.is_joined,
        response.my_role
    );

    Ok(Json(ApiResponse::success(response)))
}

/// 获取群组设置
pub async fn settings(
    State(state): State<AppState>,
    Json(req): Json<GroupSettingsRequest>,
) -> Result<Json<ApiResponse<GroupSettingsData>>> {
    let response = state.group_service.get_settings(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取成员列表
pub async fn members(
    State(state): State<AppState>,
    Json(req): Json<GroupMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMemberItem>>>> {
    let response = state.group_service.get_members(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Protected APIs ====================

pub async fn create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateGroupChatRequest>,
) -> Result<Json<ApiResponse<CreateGroupChatData>>> {
    let response = state.group_service.create_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyGroupChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<JoinedGroupItem>>>> {
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
) -> Result<Json<ApiResponse<()>>> {
    state.group_service.join_group(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn apply(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<ApplyToGroupRequest>,
) -> Result<Json<ApiResponse<ApplyToGroupData>>> {
    let response = state
        .group_service
        .apply_to_group(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
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

pub async fn admin_dismiss(
    State(state): State<AppState>,
    admin: AuthenticatedUser,
    Json(req): Json<AdminDismissGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_service
        .admin_dismiss_group(admin.user_id, &req)
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
