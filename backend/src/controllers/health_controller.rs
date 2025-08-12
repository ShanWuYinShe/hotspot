use std::sync::Arc;
use axum::{extract::State, Json};
use crate::middleware::auth::AppState;

// 健康检查响应
#[derive(serde::Serialize)]
pub struct HealthCheckResponse {
    status: String,
    version: String,
    database: String,
    redis: String,
    message_queue: String,
}

/// 健康检查处理程序
/// 返回服务的健康状态
#[utoipa::path(
    get,
    path = "/health",
    responses(
        (status = 200, description = "服务健康状态", body = HealthCheckResponse)
    )
)]
pub async fn health_check(
    State(state): State<Arc<AppState>>,
) -> Json<HealthCheckResponse> {
    // 检查数据库连接
    let db_status = match state.health_service.check_database().await {
        Ok(true) => "healthy",
        _ => "unhealthy",
    };
    
    // 检查Redis连接
    let redis_status = match state.health_service.check_redis().await {
        Ok(true) => "healthy",
        Ok(false) => "disabled",
        _ => "unhealthy",
    };
    
    // 检查消息队列
    #[cfg(feature = "jobs")]
    let mq_status = if state.job_service.is_some() {
        "healthy"
    } else {
        "disabled"
    };
    #[cfg(not(feature = "jobs"))]
    let mq_status = "disabled";
    
    Json(HealthCheckResponse {
        status: "ok".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
        database: db_status.to_string(),
        redis: redis_status.to_string(),
        message_queue: mq_status.to_string(),
    })
}
