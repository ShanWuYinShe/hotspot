use axum::{extract::State, Extension, Json};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateGroupRequest, UpdateGroupRequest, JoinGroupRequest, LeaveGroupRequest,
            InviteMemberRequest, RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupByNameRequest, SearchGroupByLocationRequest, SearchGroupByIdRequest,
            ListGroupMembersRequest, UpdateGroupProfileRequest, UpdateGroupLocationRequest,
            UpdatePrivacyRequest, UpdateNotificationsRequest, InviteRequest,
            AcceptInviteRequest, RejectInviteRequest, ListInvitesRequest,
            ApplyToGroupRequest, ReviewApplicationRequest, ListGroupsRequest,
        },
        response::{
            ApiResponse, PaginatedResponse, GroupResponse, MemberResponse,
            GroupDetailResponse, UpdateGroupProfileResponse, UpdateGroupLocationResponse,
            UpdatePrivacyResponse, UpdateNotificationsResponse, InviteMembersResponse,
            AcceptInviteResponse, RejectInviteResponse, GroupInviteResponse,
            ApplyToGroupResponse, ReviewApplicationResponse, MyGroupResponse,
        },
    },
    error::Result,
    services::group_service::GroupService,
    state::AppState,
};

/// POST /api/v1/protected/group/create
pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<CreateGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    let group_id = service.create_group(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "群组创建成功",
        "id": group_id
    }))))
}

/// POST /api/v1/protected/group/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let group_id = Uuid::parse_str(&req_body.id)
        .map_err(|_| crate::error::AppError::BadRequest("无效的群组ID".to_string()))?;
    
    let service = GroupService::new(&state.db, &state.redis);
    service.update_group(user_id, group_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "群组信息更新成功"
    }))))
}

/// POST /api/v1/protected/group/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<serde_json::Value>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let group_id_str = req_body.get("group_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| crate::error::AppError::BadRequest("缺少group_id".to_string()))?;
    
    let group_id = Uuid::parse_str(group_id_str)
        .map_err(|_| crate::error::AppError::BadRequest("无效的群组ID".to_string()))?;
    
    let service = GroupService::new(&state.db, &state.redis);
    service.delete_group(user_id, group_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "群组已解散"
    }))))
}

/// POST /api/v1/public/group/search_by_name
pub async fn search_by_name(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupByNameRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<GroupResponse>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.search_by_name(&req).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/group/search_by_location
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<GroupResponse>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.search_by_location(&req).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/join
pub async fn join(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.join_group(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "成功加入群组"
    }))))
}

/// POST /api/v1/protected/group/leave
pub async fn leave(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.leave_group(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已退出群组"
    }))))
}

/// POST /api/v1/protected/group/members
pub async fn list_members(
    State(state): State<AppState>,
    Json(req): Json<ListGroupMembersRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MemberResponse>>>> {
    let group_id = Uuid::parse_str(&req.id)
        .map_err(|_| crate::error::AppError::BadRequest("无效的群组ID".to_string()))?;
    
    let limit = req.limit.unwrap_or(20).min(100) as i64;
    
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.list_members(group_id, req.cursor.clone(), limit).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/remove_member
pub async fn remove_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<RemoveMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.remove_member(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "成员已移除"
    }))))
}

/// POST /api/v1/protected/group/mute_member
pub async fn mute_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<MuteMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.mute_member(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "成员已禁言"
    }))))
}

/// POST /api/v1/protected/group/unmute_member
pub async fn unmute_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UnmuteMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.unmute_member(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已解除禁言"
    }))))
}

/// POST /api/v1/protected/group/set_member_role
pub async fn set_member_role(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<SetMemberRoleRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.set_member_role(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "成员角色已更新"
    }))))
}

/// POST /api/v1/protected/group/transfer_ownership
pub async fn transfer_ownership(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = GroupService::new(&state.db, &state.redis);
    service.transfer_ownership(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "群主已转让"
    }))))
}

/// POST /api/v1/public/group/search_by_id
pub async fn search_by_id(
    State(state): State<AppState>,
    Json(req): Json<SearchGroupByIdRequest>,
) -> Result<Json<ApiResponse<GroupDetailResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.search_by_id(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/update_profile
pub async fn update_profile(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateGroupProfileRequest>,
) -> Result<Json<ApiResponse<UpdateGroupProfileResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.update_profile(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/update_location
pub async fn update_location(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateGroupLocationRequest>,
) -> Result<Json<ApiResponse<UpdateGroupLocationResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.update_location(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/update_privacy
pub async fn update_privacy(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdatePrivacyRequest>,
) -> Result<Json<ApiResponse<UpdatePrivacyResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.update_privacy(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/update_notifications
pub async fn update_notifications(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateNotificationsRequest>,
) -> Result<Json<ApiResponse<UpdateNotificationsResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.update_notifications(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/ban_member
pub async fn ban_member(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<BanMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let service = GroupService::new(&state.db, &state.redis);
    service.ban_member(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "成员已封禁"
    }))))
}

/// POST /api/v1/protected/group/invite
pub async fn invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<InviteRequest>,
) -> Result<Json<ApiResponse<InviteMembersResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.invite_members(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/accept_invite
pub async fn accept_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<AcceptInviteRequest>,
) -> Result<Json<ApiResponse<AcceptInviteResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.accept_invite(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/reject_invite
pub async fn reject_invite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<RejectInviteRequest>,
) -> Result<Json<ApiResponse<RejectInviteResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.reject_invite(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/list_invites
pub async fn list_invites(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListInvitesRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<GroupInviteResponse>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.list_invites(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/apply
pub async fn apply(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ApplyToGroupRequest>,
) -> Result<Json<ApiResponse<ApplyToGroupResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.apply_to_group(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/review_application
pub async fn review_application(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ReviewApplicationRequest>,
) -> Result<Json<ApiResponse<ReviewApplicationResponse>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.review_application(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/list
pub async fn list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MyGroupResponse>>>> {
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.list_my_groups(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}
