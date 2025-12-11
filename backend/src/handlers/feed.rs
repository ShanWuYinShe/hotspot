use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{GetFeedMomentsRequest, GetFeedGroupsRequest, GetFeedUsersRequest, GetFeedTrendingRequest, FeedFeedbackRequest},
        response::{ApiResponse, PaginatedData, FeedMomentItem, FeedGroupItem, FeedUserItem, FeedTrendingData},
    },
    error::Result,
    services::feed_service::FeedService,
    state::AppState,
};

/// 获取瞬间信息流
/// POST /api/v1/protected/feed/moments
pub async fn get_moments(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFeedMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedMomentItem>>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    let result = service.get_moments_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取群组推荐
/// POST /api/v1/protected/feed/groups
pub async fn get_groups(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFeedGroupsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedGroupItem>>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    let result = service.get_groups_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取用户推荐
/// POST /api/v1/protected/feed/users
pub async fn get_users(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFeedUsersRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedUserItem>>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    let result = service.get_users_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 获取热门内容
/// POST /api/v1/protected/feed/trending
pub async fn get_trending(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFeedTrendingRequest>,
) -> Result<Json<ApiResponse<FeedTrendingData>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    let result = service.get_trending(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 提交反馈
/// POST /api/v1/protected/feed/feedback
pub async fn feedback(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedFeedbackRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    service.submit_feedback(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
