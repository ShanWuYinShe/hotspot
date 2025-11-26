use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateConfigRequest, UpdateConfigRequest, GetConfigRequest, GetConfigListRequest,
            DeleteConfigRequest, GetConfigHistoryRequest, ListAllConfigChangesRequest,
        },
        response::{
            ApiResponse, ConfigItem, CreateConfigData, ConfigHistoryItem, PaginatedData,
        },
    },
    error::{AppError, Result},
    services::system_config_service::SystemConfigService,
    state::AppState,
};

/// 检查管理员权限
async fn check_admin(state: &AppState, user_id: i64) -> Result<()> {
    let is_admin = sqlx::query_scalar::<_, bool>(
        r#"
        SELECT is_admin
        FROM users
        WHERE user_id = $1
        "#,
    )
    .bind(user_id)
    .fetch_optional(&state.db)
    .await?
    .unwrap_or(false);

    if !is_admin {
        return Err(AppError::Forbidden("需要管理员权限".to_string()));
    }

    Ok(())
}

/// POST /api/v1/admin/config/create
pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateConfigRequest>,
) -> Result<Json<ApiResponse<CreateConfigData>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let config = service.create_config(req, user_id).await?;

    Ok(Json(ApiResponse::success(config)))
}

/// POST /api/v1/admin/config/get_config
pub async fn get_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetConfigRequest>,
) -> Result<Json<ApiResponse<ConfigItem>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let config = service.get_single_config(&req.key).await?;

    Ok(Json(ApiResponse::success(config)))
}

/// POST /api/v1/admin/config/get_list
pub async fn get_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetConfigListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigItem>>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let configs = service.get_list(req).await?;

    Ok(Json(ApiResponse::success(configs)))
}

/// POST /api/v1/admin/config/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateConfigRequest>,
) -> Result<Json<ApiResponse<()>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    // Note: req contains key and value. service.update_config takes key as separate arg.
    let key = req.key.clone();
    service.update_config(&key, req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/admin/config/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteConfigRequest>,
) -> Result<Json<ApiResponse<()>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    service.delete_config(&req.key, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/admin/config/get_config_history
pub async fn get_config_history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetConfigHistoryRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigHistoryItem>>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    // Note: req contains key. service.get_config_history takes key separate.
    let key = req.key.clone();
    let logs = service.get_config_history(&key, req).await?;

    Ok(Json(ApiResponse::success(logs)))
}

/// POST /api/v1/admin/config/list_all_changes
pub async fn list_all_changes(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListAllConfigChangesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ConfigHistoryItem>>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let logs = service.list_all_changes(req).await?;

    Ok(Json(ApiResponse::success(logs)))
}
