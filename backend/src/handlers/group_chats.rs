use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateGroupChatRequest, UpdateGroupChatRequest, JoinGroupRequest, LeaveGroupRequest,
            InviteMemberRequest, RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, UnbanMemberRequest, GetBannedMembersListRequest,
            SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupsByKeywordRequest, SearchGroupsByTagRequest, SearchGroupsByLocationRequest,
            GroupOpenTheDoorRequest, GetGroupSettingsRequest, GetMembersRequest,
            GetJoinedGroupChatsRequest, AcceptGroupInviteRequest, RejectGroupInviteRequest,
            GetGroupInvitesListRequest, ApplyToGroupRequest, ApproveGroupApplicationRequest,
            RejectGroupApplicationRequest, CancelGroupApplicationRequest, GetGroupApplicationsListRequest,
        },
        response::{
            ApiResponse, PaginatedData, GroupSearchItem, JoinedGroupItem,
            GroupSettingsData, CreateGroupChatData, GroupMemberItem, BannedMemberItem,
            InviteMemberData, ApplyToGroupData, GroupInviteItem, GroupApplicationItem,
        },
    },
    error::Result,
    services::group_service::GroupService,
    state::AppState,
};


// ==================== Public APIs ====================

/// 按关键字搜索群组（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupSearchItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_keyword(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索群组（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupSearchItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_tag(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索群组（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupSearchItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_location(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取群组详情（通过 ID 获取指定群组的详情）
pub async fn open_the_door(
    State(state): State<AppState>,
    Json(req): Json<GroupOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<GroupSearchItem>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.open_the_door(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取群组设置
pub async fn get_settings(
    State(state): State<AppState>,
    Json(req): Json<GetGroupSettingsRequest>,
) -> Result<Json<ApiResponse<GroupSettingsData>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_settings(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取成员列表
pub async fn get_members(
    State(state): State<AppState>,
    Json(req): Json<GetMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMemberItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_members(req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== Protected APIs ====================

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateGroupChatRequest>,
) -> Result<Json<ApiResponse<CreateGroupChatData>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.create_group(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_joined(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetJoinedGroupChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<JoinedGroupItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_joined(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<crate::dto::request::DeleteGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn transfer_ownership(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.transfer_ownership(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


// ==================== 成员操作 ====================

pub async fn invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<InviteMemberRequest>,
) -> Result<Json<ApiResponse<InviteMemberData>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.invite_member(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn join(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.join_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn apply(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ApplyToGroupRequest>,
) -> Result<Json<ApiResponse<ApplyToGroupData>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.apply_to_group(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn set_role(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SetMemberRoleRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.set_role(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn mute(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.mute_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unmute(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnmuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unmute_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn ban_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<BanMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.ban_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unban_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnbanMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unban_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn get_banned_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetBannedMembersListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<BannedMemberItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let result = service.get_banned_list(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}


pub async fn remove_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemoveMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.remove_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn leave_group(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.leave_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


// ==================== 申请管理 ====================

pub async fn get_applications(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupApplicationsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupApplicationItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_applications(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn approve_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ApproveGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.approve_application(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn reject_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RejectGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.reject_application(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn cancel_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CancelGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.cancel_application(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


// ==================== 邀请管理 ====================

pub async fn get_invites(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupInvitesListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupInviteItem>>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_invites(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn accept_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AcceptGroupInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.accept_invite(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn reject_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RejectGroupInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis, state.event_bus.clone());
    service.reject_invite(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
