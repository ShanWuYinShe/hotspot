use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::UpdateSubscriptionsRequest,
        response::{ApiResponse, SubscriptionsData},
    },
    error::Result,
    services::subscription_service::SubscriptionService,
    state::AppState,
};


pub async fn get_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
) -> Result<Json<ApiResponse<SubscriptionsData>>> {
    let service = SubscriptionService::new(state.db.clone());
    let result = service.get_config(user_id).await?;
    Ok(Json(ApiResponse::success(result)))
}


pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateSubscriptionsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = SubscriptionService::new(state.db.clone());
    service.update_config(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
