use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateGroupChatRequest, UpdateGroupChatRequest, JoinGroupRequest, LeaveGroupRequest,
            InviteMemberRequest, RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, SetMemberRoleRequest, TransferOwnershipRequest,
            DiscoverGroupChatsRequest, OpenTheDoorRequest, GetMembersRequest,
            GetJoinedGroupChatsRequest, AcceptGroupInviteRequest, RejectGroupInviteRequest,
            GetGroupInvitesRequest, ApplyToGroupRequest, ApproveGroupApplicationRequest,
            RejectGroupApplicationRequest, GetGroupApplicationsRequest,
        },
        response::{
            ApiResponse, PaginatedData, DiscoveredGroupItem, JoinedGroupItem,
            GroupDetailData, CreateGroupChatData, GroupMemberItem, InviteMemberData,
            ApplyToGroupData, GroupInviteItem, GroupApplicationItem,
        },
    },
    error::Result,
    services::group_service::GroupService,
    state::AppState,
};

/// POST /api/v1/protected/group_chats/create
pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateGroupChatRequest>,
) -> Result<Json<ApiResponse<CreateGroupChatData>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.create_group(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/group_chats/discover
pub async fn discover(
    State(state): State<AppState>,
    Json(req): Json<DiscoverGroupChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<DiscoveredGroupItem>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.discover(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/get_joined
pub async fn get_joined(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetJoinedGroupChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<JoinedGroupItem>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.get_joined(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/group_chats/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<OpenTheDoorRequest>,
) -> Result<Json<ApiResponse<GroupDetailData>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.open_the_door(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.update_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<crate::dto::request::DeleteGroupChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.delete_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/public/group_chats/members/get_list
pub async fn get_members(
    State(state): State<AppState>,
    Json(req): Json<GetMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMemberItem>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.get_members(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/members/invite
pub async fn invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<InviteMemberRequest>,
) -> Result<Json<ApiResponse<InviteMemberData>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.invite_member(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/members/join
pub async fn join(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.join_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/apply
pub async fn apply(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ApplyToGroupRequest>,
) -> Result<Json<ApiResponse<ApplyToGroupData>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.apply_to_group(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/members/set_role
pub async fn set_role(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SetMemberRoleRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.set_role(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/mute
pub async fn mute(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.mute_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/unmute
pub async fn unmute(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnmuteMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.unmute_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/ban
pub async fn ban(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<BanMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.ban_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/remove
pub async fn remove_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemoveMemberRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.remove_member(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/members/leave
pub async fn leave(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.leave_group(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/applications/get_list
pub async fn get_applications(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupApplicationsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupApplicationItem>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.get_applications(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/applications/approve
pub async fn approve_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ApproveGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.approve_application(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/applications/reject
pub async fn reject_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RejectGroupApplicationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.reject_application(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/transfer_ownership
pub async fn transfer_ownership(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.transfer_ownership(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/invites/get_list
pub async fn get_invites(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupInvitesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupInviteItem>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.get_invites(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_chats/invites/accept
pub async fn accept_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AcceptGroupInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.accept_invite(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_chats/invites/reject
pub async fn reject_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RejectGroupInviteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.reject_invite(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
