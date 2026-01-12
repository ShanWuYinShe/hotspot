use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            FeedFeedbackRequest, FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest,
            FeedUsersRequest,
        },
        response::{
            ApiResponse, FeedGroupItem, FeedTrendingData, FeedUserItem,
            MomentItem, PaginatedData,
        },
    },
    error::Result,
    state::AppState,
};

/// 获取瞬间信息流
/// POST /api/v1/protected/feed/moments
pub async fn get_moments(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let result = state.feed_service.get_moments_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取群组推荐
/// POST /api/v1/protected/feed/groups
pub async fn get_groups(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedGroupItem>>>> {
    let result = state.feed_service.get_groups_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取用户推荐
/// POST /api/v1/protected/feed/users
pub async fn get_users(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedUserItem>>>> {
    let result = state.feed_service.get_users_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取热门内容
/// POST /api/v1/protected/feed/trending
pub async fn get_trending(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedTrendingRequest>,
) -> Result<Json<ApiResponse<FeedTrendingData>>> {
    let result = state.feed_service.get_trending(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 提交反馈
/// POST /api/v1/protected/feed/feedback
pub async fn feedback(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedFeedbackRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.feed_service.submit_feedback(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
