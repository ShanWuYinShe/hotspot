use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            BlockRequest, DeleteAccountRequest, FollowRequest, MyBlockedRequest,
            MyFollowersRequest, MyFollowingRequest, MyFriendsRequest, MyProfileRequest,
            MySettingsRequest, MyStatsRequest, SearchUsersByKeywordRequest,
            SearchUsersByLocationRequest, SearchUsersByTagRequest, SetRemarkRequest,
            UnblockRequest, UnfollowRequest, UpdateLocationRequest, UpdateProfileRequest,
            UpdateSettingsRequest, UpgradeAccountRequest, UserFollowersRequest,
            UserFollowingRequest, UserFriendsRequest, UserGroupsRequest, UserLikedMomentsRequest,
            UserMomentsRequest, UserOpenTheDoorRequest, UserRelationshipRequest, UserStatsRequest,
        },
        response::{
            ApiResponse, AuthData, BlockedUserItem, FollowerItem, FollowingItem, FriendListItem,
            GroupInfo, MomentSearchItem, MyProfileData, PaginatedData, UserInfo, UserRelationship,
            UserSearchItem, UserSettingsData, UserStatsData,
        },
    },
    error::Result,
    state::AppState,
};

// ==================== Public APIs ====================

/// 按关键字搜索用户（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let response = state.user_service.search_by_keyword(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索用户（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let response = state.user_service.search_by_tag(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索用户（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let response = state.user_service.search_by_location(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态）
pub async fn open_the_door(
    State(state): State<AppState>,
    Json(req): Json<UserOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<UserSearchItem>>> {
    let response = state.user_service.open_the_door(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Protected APIs ====================

fn is_update_profile_empty(req: &UpdateProfileRequest) -> bool {
    req.nickname.is_none()
        && req.avatar_file_id.is_none()
        && req.bio.is_none()
        && req.tags.is_none()
}

fn is_update_settings_empty(req: &UpdateSettingsRequest) -> bool {
    req.privacy_level.is_none()
        && req.location_visible.is_none()
        && req.online_visible.is_none()
        && req.last_online_visible.is_none()
        && req.friends_visible.is_none()
        && req.following_visible.is_none()
        && req.followers_visible.is_none()
        && req.moments_visible.is_none()
        && req.stats_visible.is_none()
        && req.allow_follow.is_none()
        && req.allow_private_chat.is_none()
        && req.searchable.is_none()
        && req.phone_searchable.is_none()
        && req.nickname_searchable.is_none()
        && req.location_searchable.is_none()
}

/// 获取当前用户资料
pub async fn get_my_profile(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<MyProfileRequest>,
) -> Result<Json<ApiResponse<MyProfileData>>> {
    let data = state.user_service.get_me(user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 更新当前用户资料
pub async fn update_my_profile(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if is_update_profile_empty(&req) {
        return Ok(Json(ApiResponse::success(())));
    }

    state.user_service.update_profile(user_id, &req).await?;
    Ok(Json(ApiResponse::success(())))
}

pub async fn get_relationship(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UserRelationshipRequest>,
) -> Result<Json<ApiResponse<UserRelationship>>> {
    let response = state.user_service.get_relationship(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户统计数据（公开 API）
pub async fn get_stats(
    State(state): State<AppState>,
    Json(req): Json<UserStatsRequest>,
) -> Result<Json<ApiResponse<UserStatsData>>> {
    let response = state.user_service.get_user_stats(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户统计数据（/me/ API）
pub async fn get_my_stats(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<MyStatsRequest>,
) -> Result<Json<ApiResponse<UserStatsData>>> {
    let response = state.user_service.get_my_stats(user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户设置
pub async fn get_my_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<MySettingsRequest>,
) -> Result<Json<ApiResponse<UserSettingsData>>> {
    let data = state.user_service.get_settings(user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 更新当前用户设置
pub async fn update_my_settings(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if is_update_settings_empty(&req) {
        return Ok(Json(ApiResponse::success(())));
    }

    state.user_service.update_settings(user_id, &req).await?;
    Ok(Json(ApiResponse::success(())))
}

/// 获取指定用户好友列表（公开 API）
pub async fn get_friends(
    State(state): State<AppState>,
    Json(req): Json<UserFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let response = state.user_service.get_user_friends_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户关注列表（公开 API）
pub async fn get_following(
    State(state): State<AppState>,
    Json(req): Json<UserFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let response = state.user_service.get_user_following_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户粉丝列表（公开 API）
pub async fn get_followers(
    State(state): State<AppState>,
    Json(req): Json<UserFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserInfo>>>> {
    let response = state.user_service.get_user_followers_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户加入的公开群组（公开 API）
pub async fn get_groups(
    State(state): State<AppState>,
    Json(req): Json<UserGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupInfo>>>> {
    let response = state.user_service.get_user_groups_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户点赞的瞬间（公开 API）
pub async fn get_liked_moments(
    State(state): State<AppState>,
    Json(req): Json<UserLikedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state.user_service.get_user_liked_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户生活瞬间历史（公开 API）
pub async fn get_moments(
    State(state): State<AppState>,
    Json(req): Json<UserMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state.moment_service.get_user_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn update_location(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.update_location(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn upgrade_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthData>>> {
    let auth_data = state
        .user_service
        .upgrade_account(user_id, &req, &state.config.jwt_secret)
        .await?;
    Ok(Json(ApiResponse::success(auth_data)))
}

pub async fn delete_account(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteAccountRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.delete_account(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 关注系统（互相关注 = 好友）====================

pub async fn follow(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FollowRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.follow_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unfollow(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnfollowRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.unfollow_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn get_my_friends(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendListItem>>>> {
    let response = state.user_service.get_friends_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_my_following(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowingItem>>>> {
    let response = state.user_service.get_following_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_my_followers(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowerItem>>>> {
    let response = state.user_service.get_followers_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== 拉黑管理 ====================

pub async fn block(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<BlockRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.block_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unblock(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnblockRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.unblock_user(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn get_my_blocked(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyBlockedRequest>,
) -> Result<Json<ApiResponse<PaginatedData<BlockedUserItem>>>> {
    let response = state
        .user_service
        .get_blocked_users_list(user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== 好友备注 ====================

pub async fn set_remark(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SetRemarkRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.set_friend_remark(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
