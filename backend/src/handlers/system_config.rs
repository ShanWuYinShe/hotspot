use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            ConfigChangesRequest, ConfigDetailRequest, ConfigHistoryRequest, ConfigListRequest,
            CreateConfigRequest, DeleteConfigRequest, UpdateConfigRequest,
        },
        response::{ApiResponse, ConfigHistoryItem, ConfigItem, CreateConfigData, PaginatedData},
    },
    error::Result,
    state::AppState,
};

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateConfigRequest>,
) -> Result<Json<ApiResponse<CreateConfigData>>> {
    let config = state
        .system_config_service
        .create_config(req, user_id)
        .await?;

    Ok(Json(ApiResponse::success(config)))
}

pub async fn get_config(
    State(state): State<AppState>,
    Extension(_user_id): Extension<i64>,
    Json(req): Json<ConfigDetailRequest>,
) -> Result<Json<ApiResponse<ConfigItem>>> {
    let config = state
        .system_config_service
        .get_single_config(&req.key)
        .await?;

    Ok(Json(ApiResponse::success(config)))
}

pub async fn get_list(
    State(state): State<AppState>,
    Extension(_user_id): Extension<i64>,
    Json(req): Json<ConfigListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigItem>>>> {
    let configs = state.system_config_service.get_list(req).await?;

    Ok(Json(ApiResponse::success(configs)))
}

pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateConfigRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // Note: req contains key and value. state.system_config_service.update_config takes key as separate arg.
    let key = req.key.clone();
    state
        .system_config_service
        .update_config(&key, req, user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteConfigRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .system_config_service
        .delete_config(&req.key, user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn get_config_history(
    State(state): State<AppState>,
    Extension(_user_id): Extension<i64>,
    Json(req): Json<ConfigHistoryRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigHistoryItem>>>> {
    // Note: req contains key. state.system_config_service.get_config_history takes key separate.
    let key = req.key.clone();
    let logs = state
        .system_config_service
        .get_config_history(&key, req)
        .await?;

    Ok(Json(ApiResponse::success(logs)))
}

pub async fn get_changes(
    State(state): State<AppState>,
    Extension(_user_id): Extension<i64>,
    Json(req): Json<ConfigChangesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigHistoryItem>>>> {
    let logs = state.system_config_service.get_changes(req).await?;

    Ok(Json(ApiResponse::success(logs)))
}
