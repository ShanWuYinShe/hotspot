use axum::{extract::{Request, State}, Json};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateGroupRequest, UpdateGroupRequest, JoinGroupRequest, LeaveGroupRequest,
            InviteMemberRequest, RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupByNameRequest, SearchGroupByLocationRequest, ListGroupMembersRequest,
        },
        response::{ApiResponse, PaginatedResponse, GroupResponse, GroupMemberResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::group_service::GroupService,
    state::AppState,
};

/// POST /api/v1/protected/group/create
pub async fn create(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<CreateGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<UpdateGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let group_id = Uuid::parse_str(&req_body.group_id)
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
    req: Request,
    Json(req_body): Json<serde_json::Value>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
) -> Result<Json<ApiResponse<PaginatedResponse<GroupMemberResponse>>>> {
    let group_id = Uuid::parse_str(&req.group_id)
        .map_err(|_| crate::error::AppError::BadRequest("无效的群组ID".to_string()))?;
    
    let limit = req.limit.unwrap_or(20).min(100);
    let offset = req.cursor.unwrap_or(0);
    
    let service = GroupService::new(&state.db, &state.redis);
    let response = service.list_members(group_id, limit, offset).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group/remove_member
pub async fn remove_member(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<RemoveMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<MuteMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<UnmuteMemberRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<SetMemberRoleRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
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
    req: Request,
    Json(req_body): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = GroupService::new(&state.db, &state.redis);
    service.transfer_ownership(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "群主已转让"
    }))))
}

// TODO: 实现以下端点
// - invite_member (邀请成员)
// - ban_member (封禁成员)
// - list_my_groups (我的群组列表)
// - list_invites (我的邀请列表)
// - accept_invite (接受邀请)
// - reject_invite (拒绝邀请)
