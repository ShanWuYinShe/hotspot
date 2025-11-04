use axum::{extract::State, Extension, Json};

use crate::{
    middleware::auth::OptionalUserId,
    dto::{
        request::{
            AddFriendRequest, AcceptFriendRequest, RejectFriendRequest, RemoveFriendRequest,
            FollowRequest, UnfollowRequest, BlockRequest, UnblockRequest, ListFriendsRequest,
            ListFollowingRequest, ListFollowersRequest, ListBlockedRequest, SetFriendRemarkRequest,
            SearchUserByNameRequest, SearchUserByIdRequest, SearchUserByLocationRequest,
            UpdateLocationRequest, UpdateProfileRequest, UpgradeAccountRequest,
            GetSettingsRequest, UpdateSettingsRequest, DeleteAccountRequest,
        },
        response::{
            ApiResponse, PaginatedResponse, UserProfileResponse, NearbyUserResponse, UserResponse,
            AuthResponse, GetSettingsResponse, UpdateSettingsResponse, DeleteAccountResponse,
            FriendResponse, BlockedUserResponse, GetMeResponse,
        },
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
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<SearchUserByIdRequest>,
) -> Result<Json<ApiResponse<UserProfileResponse>>> {
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
    Extension(user_id): Extension<i64>,
) -> Result<Json<ApiResponse<GetMeResponse>>> {
    
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_me(user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/update_location
pub async fn update_location(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<AddFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<AcceptFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<RejectFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<RemoveFriendRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<FollowRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UnfollowRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<BlockRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UnblockRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
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
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<SetFriendRemarkRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = UserService::new(&state.db, &state.redis);
    service.set_friend_remark(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "备注设置成功"
    }))))
}

/// POST /api/v1/protected/user/list_friends
pub async fn list_friends(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<FriendResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_friends(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/list_following
pub async fn list_following(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<UserResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_following(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/list_followers
pub async fn list_followers(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<UserResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_followers(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/list_blocked
pub async fn list_blocked(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListBlockedRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<BlockedUserResponse>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_blocked(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/upgrade_account
pub async fn upgrade_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.upgrade_account(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/settings
pub async fn settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req_body): Json<GetSettingsRequest>,
) -> Result<Json<ApiResponse<GetSettingsResponse>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_settings(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/update_settings
pub async fn update_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateSettingsRequest>,
) -> Result<Json<ApiResponse<UpdateSettingsResponse>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.update_settings(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/delete_account
pub async fn delete_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<DeleteAccountRequest>,
) -> Result<Json<ApiResponse<DeleteAccountResponse>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.delete_account(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}
