use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{ListFeedRequest, FeedFeedbackRequest},
        response::{ApiResponse, PaginatedData, FeedItem},
    },
    error::Result,
    services::feed_service::FeedService,
    state::AppState,
};

/// 获取个性化信息流
pub async fn get_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListFeedRequest>,
) -> Result<Json<ApiResponse<PaginatedData<FeedItem>>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    let result = service.get_feed(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// 提交反馈
pub async fn feedback(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FeedFeedbackRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = FeedService::new(state.db.clone(), state.redis.clone());
    service.submit_feedback(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
