use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{GetFeedRequest, SubmitFeedbackRequest},
        response::{ApiResponse, FeedItemResponse, PaginatedResponse},
    },
    error::AppError,
    services::feed_service::FeedService,
    state::AppState,
};

pub struct FeedHandler;

impl FeedHandler {
    /// 获取个性化信息流
    pub async fn get_feed(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetFeedRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<FeedItemResponse>>>, AppError> {
        let service = FeedService::new(state.db.clone(), state.redis.clone());
        let result = service.get_feed(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 提交反馈
    pub async fn submit_feedback(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<SubmitFeedbackRequest>,
    ) -> Result<Json<ApiResponse<()>>, AppError> {
        let service = FeedService::new(state.db.clone(), state.redis.clone());
        service.submit_feedback(user_id, req).await?;
        Ok(Json(ApiResponse::success("反馈提交成功")))
    }
}

