use axum::Json;

use crate::{
    dto::request::SystemHealthRequest,
    dto::response::{ApiResponse, HealthCheckData},
    error::Result,
};

pub async fn health(
    Json(_req): Json<SystemHealthRequest>,
) -> Result<Json<ApiResponse<HealthCheckData>>> {
    Ok(Json(ApiResponse::success(HealthCheckData {
        status: "healthy".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
        timestamp: chrono::Utc::now().to_rfc3339(),
    })))
}
