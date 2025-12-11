use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            SearchUsersByKeywordRequest, SearchUsersByTagRequest, SearchUsersByLocationRequest,
            GetStatusRequest, UserOpenTheDoorRequest, GetMyProfileRequest, GetRelationshipRequest,
            GetUserStatsRequest, GetMyStatsRequest,
            GetUserFriendsListRequest, GetUserFollowingListRequest, GetUserFollowersListRequest,
            GetUserGroupsListRequest, GetUserLikedMomentsListRequest,
            UpdateLocationRequest, UpdateProfileRequest, UpgradeAccountRequest,
            GetSettingsRequest, UpdateSettingsRequest, DeleteAccountRequest,
            FollowUserRequest, UnfollowUserRequest, BlockUserRequest, UnblockUserRequest,
            GetFriendsListRequest, GetFollowingListRequest, GetFollowersListRequest, GetBlockedUsersListRequest,
            SetFriendRemarkRequest,
        },
        response::{
            ApiResponse, PaginatedData, UserSearchItem, MyProfileData, UserInfo,
            UserStatusData, UserRelationship, UserStatsData, UserSettingsData,
            FriendListItem, FollowingItem, FollowerItem, BlockedUserItem, AuthData,
            GroupInfo, MomentSearchItem,
        },
    },
    error::Result,
    services::user_service::UserService,
    state::AppState,
};


// ==================== Public APIs ====================

/// 按关键字搜索用户（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_keyword(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索用户（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_tag(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索用户（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_location(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取用户资料（通过 ID 获取指定用户的资料）
pub async fn open_the_door(
    State(state): State<AppState>,
    Json(req): Json<UserOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<UserSearchItem>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.open_the_door(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取用户状态（受隐私设置控制）
pub async fn get_status(
    State(state): State<AppState>,
    Json(req): Json<GetStatusRequest>,
) -> Result<Json<ApiResponse<UserStatusData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_status(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== Protected APIs ====================

pub async fn me(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetMyProfileRequest>,
) -> Result<Json<ApiResponse<MyProfileData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_me(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_relationship(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetRelationshipRequest>,
) -> Result<Json<ApiResponse<UserRelationship>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_relationship(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


/// 获取指定用户统计数据（公开 API）
pub async fn get_user_stats(
    State(state): State<AppState>,
    Json(req): Json<GetUserStatsRequest>,
) -> Result<Json<ApiResponse<UserStatsData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_stats(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户统计数据（/me/ API）
pub async fn get_my_stats(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetMyStatsRequest>,
) -> Result<Json<ApiResponse<UserStatsData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_my_stats(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户好友列表（公开 API）
pub async fn get_user_friends_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserFriendsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_friends_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户关注列表（公开 API）
pub async fn get_user_following_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserFollowingListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_following_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户粉丝列表（公开 API）
pub async fn get_user_followers_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserFollowersListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_followers_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户加入的公开群组（公开 API）
pub async fn get_user_groups_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserGroupsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupInfo>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_groups_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户点赞的瞬间（公开 API）
pub async fn get_user_liked_moments_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserLikedMomentsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_liked_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn update_location(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_location(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn update_profile(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_profile(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn upgrade_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let auth_data = service.upgrade_account(user_id, &req, &state.config.jwt_secret).await?;
    Ok(Json(ApiResponse::success(auth_data)))
}


pub async fn get_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetSettingsRequest>,
) -> Result<Json<ApiResponse<UserSettingsData>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_settings(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn update_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_settings(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn delete_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteAccountRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_account(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


// ==================== 关注系统（互相关注 = 好友）====================

pub async fn follow_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FollowUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.follow_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unfollow_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnfollowUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unfollow_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn get_friends_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFriendsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendListItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_friends_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_following_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFollowingListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowingItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_following_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_followers_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFollowersListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowerItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_followers_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== 拉黑管理 ====================

pub async fn block_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<BlockUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.block_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unblock_user(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnblockUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unblock_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn get_blocked_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetBlockedUsersListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<BlockedUserItem>>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_blocked_users_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== 好友备注 ====================

pub async fn set_friend_remark(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SetFriendRemarkRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = UserService::new(&state.db, &state.redis, state.event_bus.clone());
    service.set_friend_remark(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
