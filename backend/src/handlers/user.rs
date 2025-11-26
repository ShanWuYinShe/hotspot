use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            DiscoverUsersRequest, GetMyProfileRequest, OpenTheDoorRequest, GetStatsRequest,
            UpdateLocationRequest, UpdateProfileRequest, UpgradeAccountRequest,
            GetSettingsRequest, UpdateSettingsRequest, DeleteAccountRequest,
            SendFriendRequestRequest, AcceptFriendRequestRequest, RejectFriendRequestRequest,
            RemoveFriendRequest, FollowUserRequest, UnfollowUserRequest, BlockUserRequest,
            ListFriendsRequest, ListFollowingRequest, ListFollowersRequest, GetBlockedUsersRequest,
            GetFriendRequestsRequest, CancelFriendRequestRequest, SetFriendRemarkRequest,
            UnblockUserRequest,
        },
        response::{
            ApiResponse, PaginatedData, DiscoveredUserItem, MyProfileData,
            UserDetailData, UserStatsData, UserSettingsData, SendFriendRequestData,
            FriendListItem, UserInfo, FriendRequestItem, AuthData,
        },
    },
    error::Result,
    services::user_service::UserService,
    state::AppState,
};

/// POST /api/v1/public/user/discover
pub async fn discover(
    State(state): State<AppState>,
    Json(req): Json<DiscoverUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<DiscoveredUserItem>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.discover(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/me
pub async fn me(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetMyProfileRequest>,
) -> Result<Json<ApiResponse<MyProfileData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_me(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<OpenTheDoorRequest>,
) -> Result<Json<ApiResponse<UserDetailData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.open_the_door(&req, Some(user_id)).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/get_stats
pub async fn get_stats(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetStatsRequest>,
) -> Result<Json<ApiResponse<UserStatsData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_stats(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/update_location
pub async fn update_location(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.update_location(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/update_profile
pub async fn update_profile(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.update_profile(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/upgrade_account
pub async fn upgrade_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let auth_data = service.upgrade_account(user_id, &req, &state.config.jwt_secret).await?;
    Ok(Json(ApiResponse::success(auth_data)))
}

/// POST /api/v1/protected/user/get_settings
pub async fn get_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetSettingsRequest>,
) -> Result<Json<ApiResponse<UserSettingsData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_settings(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/update_settings
pub async fn update_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.update_settings(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/delete_account
pub async fn delete_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteAccountRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.delete_account(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/send_friend_request
pub async fn send_friend_request(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SendFriendRequestRequest>,
) -> Result<Json<ApiResponse<SendFriendRequestData>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.send_friend_request(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/accept_friend_request
pub async fn accept_friend_request(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AcceptFriendRequestRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.accept_friend_request(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/reject_friend_request
pub async fn reject_friend_request(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RejectFriendRequestRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.reject_friend_request(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/remove_friend
pub async fn remove_friend(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemoveFriendRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.remove_friend(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/follow
pub async fn follow(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FollowUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.follow(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/unfollow
pub async fn unfollow(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnfollowUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.unfollow(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/block_user
pub async fn block_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<BlockUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.block_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/get_friends
pub async fn get_friends(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendListItem>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_friends(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/get_following
pub async fn get_following(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_following(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/get_followers
pub async fn get_followers(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.list_followers(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/get_blocked_users
pub async fn get_blocked_users(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetBlockedUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_blocked_users(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/get_friend_requests
pub async fn get_friend_requests(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFriendRequestsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendRequestItem>>>> {
    let service = UserService::new(&state.db, &state.redis);
    let response = service.get_friend_requests(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/user/cancel_friend_request
pub async fn cancel_friend_request(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CancelFriendRequestRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.cancel_friend_request(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/set_friend_remark
pub async fn set_friend_remark(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SetFriendRemarkRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.set_friend_remark(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/user/unblock_user
pub async fn unblock_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnblockUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis);
    service.unblock_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
