use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            MyDeviceTokensRequest, RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
        },
        response::{ApiResponse, DeviceTokenItem, PaginatedData},
    },
    error::Result,
    state::AppState,
};

/// 注册设备推送Token
pub async fn register_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RegisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .device_tokens_service
        .register_device_token(user_id, req)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 注销设备推送Token
pub async fn unregister_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnregisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .device_tokens_service
        .unregister_device_token(user_id, req)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 获取设备推送Token列表
pub async fn me(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyDeviceTokensRequest>,
) -> Result<Json<ApiResponse<PaginatedData<DeviceTokenItem>>>> {
    let result = state
        .device_tokens_service
        .get_device_tokens(user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}
