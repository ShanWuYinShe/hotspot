use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            BlockRequest, DeleteAccountRequest, FollowRequest, ListFollowersRequest,
            ListFollowingRequest, ListFriendsRequest, MyBlockedRequest, MyProfileRequest,
            MySettingsRequest, SearchUsersRequest, SetRemarkRequest, UnblockRequest,
            UnfollowRequest, UpdateLocationRequest, UpdateProfileRequest, UpdateSettingsRequest,
            UpgradeAccountRequest, UserGroupsRequest, UserLikedMomentsRequest, UserMomentsRequest,
            UserOpenTheDoorRequest,
        },
        response::{
            AdminUserDetail, ApiResponse, BlockedUserSearchItem, FollowerItem, FollowingItem,
            FriendItem, GroupItem, MomentItem, MyProfile, PaginatedData, UserItem,
            UserSettingsData,
        },
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    services::admin_audit_service::AuditLogInput,
    state::AppState,
    utils::validation::Validator,
};

// ==================== Token 可选 ====================

/// Resolve target user_id: None or empty -> require auth and use current user; Some(s) -> parse to i64
fn resolve_target_user_id(
    req_user_id: &Option<String>,
    auth_user_id: Option<i64>,
) -> crate::error::Result<i64> {
    match req_user_id {
        None => auth_user_id.ok_or_else(|| {
            crate::error::ApiError::Unauthorized("需要登录以查询自己的列表".to_string())
        }),
        Some(s) if s.trim().is_empty() => auth_user_id.ok_or_else(|| {
            crate::error::ApiError::Unauthorized("需要登录以查询自己的列表".to_string())
        }),
        Some(s) => Validator::parse_user_id(s.trim()),
    }
}

/// 统一搜索用户（支持关键字/标签/位置组合）
pub async fn search(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserItem>>>> {
    let started_at = std::time::Instant::now();
    let sort_by = req.sort_by.as_deref().unwrap_or("relevance");
    tracing::debug!(sort_by = %sort_by, "Search sort parameter");
    tracing::info!(
        viewer_id = user.user_id,
        has_keyword = req.keyword.is_some(),
        has_tag = req.tag.is_some(),
        radius = req.radius.unwrap_or(5000.0),
        has_cursor = req
            .pagination
            .as_ref()
            .and_then(|p| p.cursor.as_ref())
            .is_some(),
        limit = req.pagination.as_ref().and_then(|p| p.limit).unwrap_or(20),
        "users.search request received"
    );

    let response = state.user_service.search(user.user_id, &req).await?;
    tracing::info!(
        viewer_id = user.user_id,
        items = response.items.len(),
        has_next_cursor = response.next_cursor.is_some(),
        elapsed_ms = started_at.elapsed().as_millis(),
        "users.search request completed"
    );
    Ok(Json(ApiResponse::success(response)))
}

/// 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态）
pub async fn open_the_door(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<UserOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<UserItem>>> {
    let response = state.user_service.open_the_door(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Token 必须 ====================

fn is_update_profile_empty(req: &UpdateProfileRequest) -> bool {
    req.nickname.is_none()
        && req.avatar_file_id.is_none()
        && req.bio.is_none()
        && req.tags.is_none()
        && req.gender.is_none()
}

fn is_update_settings_empty(req: &UpdateSettingsRequest) -> bool {
    let s = &req.settings;
    s.privacy_level.is_none()
        && s.location_visible.is_none()
        && s.presence_visible.is_none()
        && s.social_lists_visible.is_none()
        && s.groups_visible.is_none()
        && s.liked_moments_visible.is_none()
        && s.stats_visible.is_none()
        && s.allow_follow.is_none()
        && s.allow_private_chat.is_none()
        && s.searchable.is_none()
        && s.phone_searchable.is_none()
        && s.nickname_searchable.is_none()
        && s.location_searchable.is_none()
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
) -> Result<Json<ApiResponse<()>>> {
    if !is_update_settings_empty(&req) {
        state
            .user_service
            .update_settings(user.user_id, &req)
            .await?;
    }
    Ok(Json(ApiResponse::success_empty()))
}

/// 获取好友列表（user_id 可选，不传时=当前用户，需登录）
pub async fn get_friends(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<ListFriendsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FriendItem>>>> {
    let target_user_id = resolve_target_user_id(&req.user_id, user.user_id)?;
    let response = state
        .user_service
        .get_friends_list(target_user_id, user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取关注列表（user_id 可选，不传时=当前用户，需登录）
pub async fn get_following(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<ListFollowingRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowingItem>>>> {
    let target_user_id = resolve_target_user_id(&req.user_id, user.user_id)?;
    let response = state
        .user_service
        .get_following_list(target_user_id, user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取粉丝列表（user_id 可选，不传时=当前用户，需登录）
pub async fn get_followers(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<ListFollowersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FollowerItem>>>> {
    let target_user_id = resolve_target_user_id(&req.user_id, user.user_id)?;
    let response = state
        .user_service
        .get_followers_list(target_user_id, user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户加入的公开群组（受 groups_visible 隐私设置控制）
pub async fn get_groups(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<UserGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let response = state
        .user_service
        .get_user_groups_list(&req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户点赞的瞬间（受 liked_moments_visible 隐私设置控制）
pub async fn get_liked_moments(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<UserLikedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .user_service
        .get_user_liked_moments_list(&req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户生活瞬间历史（公开 API）
pub async fn get_moments(
    State(state): State<AppState>,
    Json(req): Json<UserMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
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
) -> Result<Json<ApiResponse<()>>> {
    state
        .user_service
        .upgrade_account(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
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
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.follow_user(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unfollow(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnfollowRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.unfollow_user(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 拉黑管理 ====================

pub async fn block(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<BlockRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.block_user(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unblock(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnblockRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.user_service.unblock_user(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
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
    let target_user_id = Validator::parse_user_id(&req.user_id)?;
    let is_target_admin = state.admin_rbac_service.is_admin(target_user_id).await?;
    let is_operator_super = state
        .admin_rbac_service
        .is_super_admin(admin.user_id)
        .await?;

    // 业务规则校验 + 封禁操作（在 service 内部完成）
    state
        .user_service
        .admin_ban_user(
            admin.user_id,
            target_user_id,
            is_target_admin,
            is_operator_super,
            req.reason.clone(),
            req.duration_days,
        )
        .await?;

    // 审计日志（跨服务协调，保留在 handler）
    let details = serde_json::json!({
        "reason": req.reason,
        "duration_days": req.duration_days,
    });
    state
        .admin_audit_service
        .record(AuditLogInput::user_action(
            admin.user_id,
            "user.ban",
            target_user_id,
            Some(details),
        ))
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn admin_detail(
    State(state): State<AppState>,
    _admin: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminUserDetailRequest>,
) -> Result<Json<ApiResponse<AdminUserDetail>>> {
    let target_user_id = Validator::parse_user_id(&req.user_id)?;
    let mut detail = state.user_service.admin_user_detail(target_user_id).await?;
    detail.ban_history = state
        .admin_audit_service
        .get_user_ban_history(target_user_id)
        .await?;
    Ok(Json(ApiResponse::success(detail)))
}

pub async fn admin_unban(
    State(state): State<AppState>,
    admin: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminUnbanUserRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let target_user_id = Validator::parse_user_id(&req.user_id)?;
    state.user_service.admin_unban_user(target_user_id).await?;
    let details = serde_json::json!({
        "reason": req.reason,
    });
    state
        .admin_audit_service
        .record(AuditLogInput::user_action(
            admin.user_id,
            "user.unban",
            target_user_id,
            Some(details),
        ))
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
