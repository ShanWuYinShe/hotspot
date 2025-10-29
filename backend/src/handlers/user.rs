use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            AddFriendRequest, AcceptFriendRequest, RejectFriendRequest, RemoveFriendRequest,
            FollowRequest, UnfollowRequest, BlockRequest, UnblockRequest, ListFriendsRequest,
            ListFollowingRequest, ListFollowersRequest, ListBlockedRequest, SetFriendRemarkRequest,
            SearchUserByNameRequest, SearchUserByIdRequest, SearchUserByLocationRequest,
            UpdateLocationRequest, UpdateProfileRequest,
        },
        response::{ApiResponse, PaginatedResponse, UserProfileResponse, NearbyUserResponse, UserResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::user_service::UserService,
    state::AppState,
};

/// POST /api/v1/public/user/search_by_name
pub async fn search_by_name(
    State(state): State<AppState>,
    Json(req): Json<SearchUserByNameRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<UserResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.search_by_name(&req.name, req.limit, req.cursor).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/user/search_by_id
pub async fn search_by_id(
    State(state): State<AppState>,
    request: Request,
    Json(req): Json<SearchUserByIdRequest>,
) -> Result<Json<ApiResponse<UserProfileResponse>>> {
    let current_user_id = extract_user_id(&request).ok();
    
    let service = UserService::new(&state.db, &state.redis);
    let response = service.search_by_id(&req, current_user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/user/search_by_location
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchUserByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<NearbyUserResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.search_by_location(&req).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/me
pub async fn me(
    State(state): State<AppState>,
    req: Request,
) -> Result<Json<ApiResponse<UserProfileResponse>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    let search_req = SearchUserByIdRequest {
        id: user_id.to_string(),
    };
    let response = service.search_by_id(&search_req, Some(user_id)).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/update_location
pub async fn update_location(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.update_location(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "位置更新成功"
    }))))
}

/// POST /api/v1/protected/user/update_profile
pub async fn update_profile(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.update_profile(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "资料更新成功"
    }))))
}

/// POST /api/v1/protected/user/add_friend
pub async fn add_friend(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<AddFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    let (relation_id, status) = service.add_friend(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "好友申请已发送",
        "id": relation_id,
        "status": status
    }))))
}

/// POST /api/v1/protected/user/accept_friend
pub async fn accept_friend(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<AcceptFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let relationship_id = req_body.id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的关系ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.accept_friend(user_id, relationship_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已接受好友申请"
    }))))
}

/// POST /api/v1/protected/user/reject_friend
pub async fn reject_friend(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<RejectFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let relationship_id = req_body.id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的关系ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.reject_friend(user_id, relationship_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已拒绝好友申请"
    }))))
}

/// POST /api/v1/protected/user/remove_friend
pub async fn remove_friend(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<RemoveFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let friend_id = req_body.user_id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的用户ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.remove_friend(user_id, friend_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已删除好友"
    }))))
}

/// POST /api/v1/protected/user/follow
pub async fn follow(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<FollowRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let to_user_id = req_body.user_id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的用户ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    let relation_id = service.follow(user_id, to_user_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "关注成功",
        "id": relation_id
    }))))
}

/// POST /api/v1/protected/user/unfollow
pub async fn unfollow(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UnfollowRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let to_user_id = req_body.user_id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的用户ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.unfollow(user_id, to_user_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消关注"
    }))))
}

/// POST /api/v1/protected/user/block
pub async fn block(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<BlockRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    let relation_id = service.block(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已拉黑该用户",
        "id": relation_id
    }))))
}

/// POST /api/v1/protected/user/unblock
pub async fn unblock(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UnblockRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let to_user_id = req_body.user_id.parse::<i64>()
        .map_err(|_| crate::error::AppError::BadRequest("无效的用户ID".to_string()))?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.unblock(user_id, to_user_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已解除拉黑"
    }))))
}

/// POST /api/v1/protected/user/set_friend_remark
pub async fn set_friend_remark(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<SetFriendRemarkRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = UserService::new(&state.db, &state.redis);
    service.set_friend_remark(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "备注设置成功"
    }))))
}

// TODO: 实现列表查询端点
// - list_friends
// - list_following
// - list_followers
// - list_blocked
// - upgrade_account
// - settings
// - update_settings
// - delete_account
