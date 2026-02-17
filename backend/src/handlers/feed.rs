use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            DismissFeedItemRequest, FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest,
            FeedUsersRequest,
        },
        response::{ApiResponse, FeedTrendingData, GroupItem, MomentItem, PaginatedData, UserItem},
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    state::AppState,
};

/// 获取瞬间信息流
/// POST /api/v1/feed/moments
pub async fn get_moments(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let result = state
        .feed_service
        .get_moments_feed(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取群组推荐
/// POST /api/v1/feed/groups
pub async fn get_groups(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupItem>>>> {
    let result = state
        .feed_service
        .get_groups_feed(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取用户推荐
/// POST /api/v1/feed/users
pub async fn get_users(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserItem>>>> {
    let result = state.feed_service.get_users_feed(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取热门内容（Token 可选；登录态返回 my_state 等个性化字段）
/// POST /api/v1/feed/trending
pub async fn get_trending(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<FeedTrendingRequest>,
) -> Result<Json<ApiResponse<FeedTrendingData>>> {
    let result = state.feed_service.get_trending(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 标记 Feed 中的内容为“不感兴趣”
pub async fn dismiss_item(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DismissFeedItemRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.feed_service.dismiss_item(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
