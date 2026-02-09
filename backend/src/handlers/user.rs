use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            BlockRequest, DeleteAccountRequest, FollowRequest, GeoPoint,
            GetRelationshipRequest, MyBlockedRequest,
            MyFollowersRequest, MyFollowingRequest, MyFriendsRequest, MyProfileRequest,
            MySettingsRequest, SearchUsersByKeywordRequest,
            SearchUsersByLocationRequest, SearchUsersByTagRequest, SearchUsersRequest,
            SetRemarkRequest, UnblockRequest, UnfollowRequest, UpdateLocationRequest,
            UpdateProfileRequest, UpdateSettingsRequest, UpgradeAccountRequest,
            UserFollowersRequest, UserFollowingRequest, UserFriendsRequest, UserGroupsRequest,
            UserLikedMomentsRequest, UserMomentsRequest, UserOpenTheDoorRequest,
        },
        response::{
            AdminUserDetail, ApiResponse, AuthData, BlockedUserSearchItem,
            FollowerItem, FollowingItem, FriendItem, GroupSearchItem, MyProfile,
            PaginatedData, UserMomentListItem, UserRelation, UserSearchItem,
            UserSettingsData,
        },
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

// ==================== Token 可选 ====================

/// 统一搜索用户（支持关键字/标签/位置组合）
pub async fn search(
    State(state): State<AppState>,
    Json(req): Json<SearchUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserSearchItem>>>> {
    let reference_point = req.search.reference_point.unwrap_or(GeoPoint {
        latitude: 0.0,
        longitude: 0.0,
    });

    // 根据参数分发到不同的搜索逻辑
    if let Some(radius) = req.radius {
        // 位置搜索
        let location_req = SearchUsersByLocationRequest {
            kw: req.keyword,
            reference_point,
            radius,
            limit: req.search.limit,
            cursor: req.search.cursor,
        };
        let response = state.user_service.search_by_location(&location_req).await?;
        Ok(Json(ApiResponse::success(response)))
    } else if let Some(tag) = req.tag {
        // 标签搜索
        let tag_req = SearchUsersByTagRequest {
            tag,
            reference_point,
            limit: req.search.limit,
            cursor: req.search.cursor,
        };
        let response = state.user_service.search_by_tag(&tag_req).await?;
        Ok(Json(ApiResponse::success(response)))
    } else if let Some(keyword) = req.keyword {
        // 关键字搜索
        let kw_req = SearchUsersByKeywordRequest {
            kw: keyword,
            reference_point,
            limit: req.search.limit,
            cursor: req.search.cursor,
        };
        let response = state.user_service.search_by_keyword(&kw_req).await?;
        Ok(Json(ApiResponse::success(response)))
    } else {
        // 至少需要一个搜索条件
        Err(crate::error::AppError::BadRequest(
            "至少需要提供 keyword、tag、radius 中的一个搜索条件".to_string(),
        ))
    }
}

/// 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态）
pub async fn open_the_door(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<UserOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<UserSearchItem>>> {
    let response = state.user_service.open_the_door(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Token 必须 ====================

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
        && req.groups_visible.is_none()
        && req.moments_visible.is_none()
        && req.liked_moments_visible.is_none()
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
    user: AuthenticatedUser,
    Json(_req): Json<MyProfileRequest>,
) -> Result<Json<ApiResponse<MyProfile>>> {
    let data = state.user_service.get_me(user.user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 更新当前用户资料
pub async fn update_my_profile(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateProfileRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if is_update_profile_empty(&req) {
        return Ok(Json(ApiResponse::success(())));
    }

    state
        .user_service
        .update_profile(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(())))
}

/// 获取当前用户设置
pub async fn get_my_settings(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(_req): Json<MySettingsRequest>,
) -> Result<Json<ApiResponse<UserSettingsData>>> {
    let data = state.user_service.get_settings(user.user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 更新当前用户设置
pub async fn update_my_settings(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateSettingsRequest>,
) -> Result<Json<ApiResponse<UserSettingsData>>> {
    if !is_update_settings_empty(&req) {
        state
            .user_service
            .update_settings(user.user_id, &req)
            .await?;
    }
    let data = state.user_service.get_settings(user.user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 获取指定用户好友列表（公开 API）
pub async fn get_friends(
    State(state): State<AppState>,
    Json(req): Json<UserFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendItem>>>> {
    let response = state.user_service.get_friends_list(req.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户关注列表（公开 API）
pub async fn get_following(
    State(state): State<AppState>,
    Json(req): Json<UserFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowingItem>>>> {
    let response = state.user_service.get_following_list(req.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户粉丝列表（公开 API）
pub async fn get_followers(
    State(state): State<AppState>,
    Json(req): Json<UserFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowerItem>>>> {
    let response = state.user_service.get_followers_list(req.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户加入的公开群组（公开 API）
pub async fn get_groups(
    State(state): State<AppState>,
    Json(req): Json<UserGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupSearchItem>>>> {
    let response = state.user_service.get_user_groups_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户点赞的瞬间（公开 API）
pub async fn get_liked_moments(
    State(state): State<AppState>,
    Json(req): Json<UserLikedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserMomentListItem>>>> {
    let response = state.user_service.get_user_liked_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户生活瞬间历史（公开 API）
pub async fn get_moments(
    State(state): State<AppState>,
    Json(req): Json<UserMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserMomentListItem>>>> {
    let response = state.moment_service.get_user_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn update_location(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .user_service
        .update_location(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn upgrade_account(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthData>>> {
    let auth_data = state
        .user_service
        .upgrade_account(user.user_id, &req, &state.config.jwt_secret)
        .await?;
    Ok(Json(ApiResponse::success(auth_data)))
}

pub async fn delete_account(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DeleteAccountRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .user_service
        .delete_account(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 关注系统（互相关注 = 好友）====================

pub async fn follow(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FollowRequest>,
) -> Result<Json<ApiResponse<UserRelation>>> {
    state.user_service.follow_user(user.user_id, &req).await?;
    let rel_req = GetRelationshipRequest {
        user_id: req.user_id,
    };
    let data = state
        .user_service
        .get_relationship(user.user_id, &rel_req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn unfollow(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnfollowRequest>,
) -> Result<Json<ApiResponse<UserRelation>>> {
    state.user_service.unfollow_user(user.user_id, &req).await?;
    let rel_req = GetRelationshipRequest {
        user_id: req.user_id,
    };
    let data = state
        .user_service
        .get_relationship(user.user_id, &rel_req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn get_my_friends(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendItem>>>> {
    let service_req = UserFriendsRequest {
        user_id: user.user_id,
        cursor: req.cursor,
        limit: req.limit,
    };
    let response = state
        .user_service
        .get_friends_list(user.user_id, &service_req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_my_following(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowingItem>>>> {
    let service_req = UserFollowingRequest {
        user_id: user.user_id,
        cursor: req.cursor,
        limit: req.limit,
    };
    let response = state
        .user_service
        .get_following_list(user.user_id, &service_req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_my_followers(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowerItem>>>> {
    let service_req = UserFollowersRequest {
        user_id: user.user_id,
        cursor: req.cursor,
        limit: req.limit,
    };
    let response = state
        .user_service
        .get_followers_list(user.user_id, &service_req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== 拉黑管理 ====================

pub async fn block(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<BlockRequest>,
) -> Result<Json<ApiResponse<UserRelation>>> {
    state.user_service.block_user(user.user_id, &req).await?;
    let rel_req = GetRelationshipRequest {
        user_id: req.user_id,
    };
    let data = state
        .user_service
        .get_relationship(user.user_id, &rel_req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn unblock(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnblockRequest>,
) -> Result<Json<ApiResponse<UserRelation>>> {
    state.user_service.unblock_user(user.user_id, &req).await?;
    let rel_req = GetRelationshipRequest {
        user_id: req.user_id,
    };
    let data = state
        .user_service
        .get_relationship(user.user_id, &rel_req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn get_my_blocked(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyBlockedRequest>,
) -> Result<Json<ApiResponse<PaginatedData<BlockedUserSearchItem>>>> {
    let response = state
        .user_service
        .get_blocked_users_list(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== 好友备注 ====================

pub async fn set_remark(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<SetRemarkRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .user_service
        .set_friend_remark(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 管理员用户管理 (Admin) ====================

pub async fn admin_search(
    State(state): State<AppState>,
    Json(req): Json<crate::dto::request::AdminUserSearchRequest>,
) -> Result<Json<ApiResponse<PaginatedData<crate::dto::response::AdminUserSearchItem>>>> {
    let response = state.user_service.admin_search_users(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn admin_ban(
    State(state): State<AppState>,
    admin: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminBanUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if req.user_id == admin.user_id {
        return Err(crate::error::AppError::CannotOperateOnSelf);
    }
    let operator_is_super = state
        .admin_rbac_service
        .is_super_admin(admin.user_id)
        .await?;
    if state.admin_rbac_service.is_admin(req.user_id).await? && !operator_is_super {
        return Err(crate::error::AppError::CannotOperateAdmin(
            "无法操作管理员账户".to_string(),
        ));
    }
    state.user_service.admin_ban_user(&req).await?;
    let details = serde_json::json!({
        "reason": req.reason,
        "duration_days": req.duration_days,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "user.ban".to_string(),
            operator_id: admin.user_id,
            target_type: "user".to_string(),
            target_id: req.user_id.to_string(),
            target_name: None,
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn admin_detail(
    State(state): State<AppState>,
    _admin: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminUserDetailRequest>,
) -> Result<Json<ApiResponse<AdminUserDetail>>> {
    let mut detail = state.user_service.admin_user_detail(&req).await?;
    detail.ban_history = state
        .admin_audit_service
        .get_user_ban_history(req.user_id)
        .await?;
    Ok(Json(ApiResponse::success(detail)))
}

pub async fn admin_unban(
    State(state): State<AppState>,
    admin: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminUnbanUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.admin_unban_user(&req).await?;
    let details = serde_json::json!({
        "reason": req.reason,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "user.unban".to_string(),
            operator_id: admin.user_id,
            target_type: "user".to_string(),
            target_id: req.user_id.to_string(),
            target_name: None,
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
