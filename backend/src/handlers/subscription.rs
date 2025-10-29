use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{GetSubscriptionsRequest, UpdateSubscriptionRequest, UpdateSubscriptionsBatchRequest},
        response::{ApiResponse, SubscriptionResponse},
    },
    error::AppError,
    services::subscription_service::SubscriptionService,
    state::AppState,
};

pub struct SubscriptionHandler;

impl SubscriptionHandler {
    /// 获取订阅配置
    pub async fn get_subscriptions(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetSubscriptionsRequest>,
    ) -> Result<Json<ApiResponse<Vec<SubscriptionResponse>>>, AppError> {
        let service = SubscriptionService::new(state.db.clone());
        let result = service.get_subscriptions(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 更新订阅配置
    pub async fn update_subscription(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<UpdateSubscriptionRequest>,
    ) -> Result<Json<ApiResponse<SubscriptionResponse>>, AppError> {
        let service = SubscriptionService::new(state.db.clone());
        let result = service.update_subscription(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 批量更新订阅配置
    pub async fn update_subscriptions_batch(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<UpdateSubscriptionsBatchRequest>,
    ) -> Result<Json<ApiResponse<Vec<SubscriptionResponse>>>, AppError> {
        let service = SubscriptionService::new(state.db.clone());
        let result = service.update_subscriptions_batch(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }
}

