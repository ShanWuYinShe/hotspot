use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
            GetDeviceTokensRequest,
        },
        response::{
            ApiResponse, PaginatedData, DeviceTokenItem,
        },
    },
    error::AppError,
    services::inbox_service::InboxService,
    state::AppState,
};

/// 注册设备推送Token
pub async fn register_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RegisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    inbox_service.register_device_token(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 注销设备推送Token
pub async fn unregister_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnregisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    inbox_service.unregister_device_token(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 获取设备推送Token列表
pub async fn get_device_tokens(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetDeviceTokensRequest>,
) -> Result<Json<ApiResponse<PaginatedData<DeviceTokenItem>>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    let result = inbox_service.get_device_tokens(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}
