use axum::{extract::State, Extension, Json};
use serde::{Deserialize, Serialize};

use crate::{
    dto::response::ApiResponse,
    error::{AppError, Result},
    middleware::auth::extract_user_id,
    models::{
        config_change_log::ConfigChangeLogResponse,
        system_config::{
            CreateSystemConfigRequest, SystemConfigResponse, UpdateSystemConfigRequest,
        },
    },
    services::system_config_service::SystemConfigService,
    state::AppState,
};

#[derive(Debug, Deserialize)]
pub struct GetConfigRequest {
    pub config_key: String,
}

#[derive(Debug, Serialize)]
pub struct GetConfigResponse {
    pub config: Option<SystemConfigResponse>,
}

#[derive(Debug, Deserialize)]
pub struct ListConfigsRequest {}

#[derive(Debug, Serialize)]
pub struct ListConfigsResponse {
    pub configs: Vec<SystemConfigResponse>,
}

#[derive(Debug, Deserialize)]
pub struct GetConfigHistoryRequest {
    pub config_key: String,
    pub limit: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct GetConfigHistoryResponse {
    pub logs: Vec<ConfigChangeLogResponse>,
}

#[derive(Debug, Deserialize)]
pub struct ListAllChangesRequest {
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct ListAllChangesResponse {
    pub logs: Vec<ConfigChangeLogResponse>,
}

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

/// 获取配置项
pub async fn get_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetConfigRequest>,
) -> Result<Json<ApiResponse<GetConfigResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let config = service.get_config(&req.config_key).await?;

    Ok(Json(ApiResponse::success(GetConfigResponse { config })))
}

/// 列出所有配置项
pub async fn list_configs(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<ListConfigsRequest>,
) -> Result<Json<ApiResponse<ListConfigsResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let configs = service.list_configs().await?;

    Ok(Json(ApiResponse::success(ListConfigsResponse { configs })))
}

/// 创建配置项
pub async fn create_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateSystemConfigRequest>,
) -> Result<Json<ApiResponse<SystemConfigResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    
    let ip_address = None;
    let user_agent = None;

    let config = service
        .create_config(req, user_id, ip_address, user_agent)
        .await?;

    Ok(Json(ApiResponse::success(config)))
}

/// 更新配置项
pub async fn update_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateConfigRequest>,
) -> Result<Json<ApiResponse<SystemConfigResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    
    let ip_address = None;
    let user_agent = None;

    let config = service
        .update_config(&req.config_key, req.update, user_id, ip_address, user_agent)
        .await?;

    Ok(Json(ApiResponse::success(config)))
}

/// 删除配置项
pub async fn delete_config(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteConfigRequest>,
) -> Result<Json<ApiResponse<DeleteConfigResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    
    let ip_address = None;
    let user_agent = None;

    service
        .delete_config(&req.config_key, user_id, ip_address, user_agent)
        .await?;

    Ok(Json(ApiResponse::success(DeleteConfigResponse {
        success: true,
        message: "配置项已删除".to_string(),
    })))
}

/// 获取配置变更历史
pub async fn get_config_history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetConfigHistoryRequest>,
) -> Result<Json<ApiResponse<GetConfigHistoryResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let limit = req.limit.unwrap_or(50);
    let logs = service.get_config_history(&req.config_key, limit).await?;

    Ok(Json(ApiResponse::success(GetConfigHistoryResponse { logs })))
}

/// 获取所有变更历史
pub async fn list_all_changes(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListAllChangesRequest>,
) -> Result<Json<ApiResponse<ListAllChangesResponse>>> {
    check_admin(&state, user_id).await?;
    
    let service = SystemConfigService::new(state.db.clone(), state.redis_client.clone());
    let limit = req.limit.unwrap_or(50);
    let offset = req.offset.unwrap_or(0);
    let logs = service.list_all_changes(limit, offset).await?;

    Ok(Json(ApiResponse::success(ListAllChangesResponse { logs })))
}

#[derive(Debug, Deserialize)]
pub struct UpdateConfigRequest {
    pub config_key: String,
    #[serde(flatten)]
    pub update: UpdateSystemConfigRequest,
}

#[derive(Debug, Deserialize)]
pub struct DeleteConfigRequest {
    pub config_key: String,
}

#[derive(Debug, Serialize)]
pub struct DeleteConfigResponse {
    pub success: bool,
    pub message: String,
}

