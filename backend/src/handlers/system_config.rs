use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminGetGlobalSettingsRequest, FetchGlobalSettingsRequest, UpdateContactRequest,
            UpdateDefaultsRequest, UpdateFeaturesRequest, UpdateLimitsRequest,
            UpdateUploadConfigRequest,
        },
        response::{AdminGlobalSettingsData, ApiResponse, FetchSettingsData, GlobalSettings},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

// ==================== 客户端拉取配置 ====================

pub async fn fetch_global_settings(
    State(state): State<AppState>,
    Json(req): Json<FetchGlobalSettingsRequest>,
) -> Result<Json<ApiResponse<FetchSettingsData>>> {
    let data = state
        .system_config_service
        .fetch_global_settings(req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

// ==================== 管理员配置管理 (Admin) ====================

pub async fn admin_get_global_settings(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminGetGlobalSettingsRequest>,
) -> Result<Json<ApiResponse<AdminGlobalSettingsData>>> {
    let data = state
        .system_config_service
        .get_admin_global_settings()
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn admin_update_upload_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateUploadConfigRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_upload_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}

pub async fn admin_update_defaults_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateDefaultsRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_defaults_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}

pub async fn admin_update_features_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateFeaturesRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_features_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}

pub async fn admin_update_contact_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateContactRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_contact_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}

pub async fn admin_update_limits_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateLimitsRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_limits_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}
