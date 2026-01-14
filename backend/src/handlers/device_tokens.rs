use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            MyDeviceTokensRequest, RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
        },
        response::{ApiResponse, DeviceTokenItem, PaginatedData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

/// 注册设备推送Token
pub async fn register_device_token(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RegisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .device_tokens_service
        .register_device_token(user.user_id, req)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 注销设备推送Token
pub async fn unregister_device_token(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnregisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .device_tokens_service
        .unregister_device_token(user.user_id, req)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 获取设备推送Token列表
pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyDeviceTokensRequest>,
) -> Result<Json<ApiResponse<PaginatedData<DeviceTokenItem>>>> {
    let result = state
        .device_tokens_service
        .get_device_tokens(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}
