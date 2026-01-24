use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            FeedFeedbackRequest, FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest,
            FeedUsersRequest,
        },
        response::{
            ApiResponse, FeedGroupItem, FeedMomentItem, FeedTrendingData, FeedUserItem,
            PaginatedData,
        },
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

/// 获取瞬间信息流
/// POST /api/v1/feed/moments
pub async fn get_moments(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedMomentItem>>>> {
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
) -> Result<Json<ApiResponse<PaginatedData<FeedGroupItem>>>> {
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
) -> Result<Json<ApiResponse<PaginatedData<FeedUserItem>>>> {
    let result = state.feed_service.get_users_feed(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取热门内容
/// POST /api/v1/feed/trending
pub async fn get_trending(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedTrendingRequest>,
) -> Result<Json<ApiResponse<FeedTrendingData>>> {
    let result = state.feed_service.get_trending(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 提交反馈
/// POST /api/v1/feed/feedback
pub async fn feedback(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FeedFeedbackRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .feed_service
        .submit_feedback(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
