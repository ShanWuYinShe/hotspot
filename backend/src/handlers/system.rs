use axum::{extract::State, Json};
use std::time::Duration;

use crate::dto::{
    request::SystemHealthRequest,
    response::{ApiResponse, HealthCheckData},
};
use crate::error::Result;
use crate::state::AppState;
use crate::types::enums::SystemHealthStatus;

pub async fn health(
    State(state): State<AppState>,
    Json(_req): Json<SystemHealthRequest>,
) -> Result<Json<ApiResponse<HealthCheckData>>> {
    // 数据库健康检查（基础设施检查，不涉及业务逻辑）
    let db_ok = match tokio::time::timeout(
        Duration::from_secs(2),
        sqlx::query!(r#"SELECT 1 as "check!""#).fetch_one(&state.db),
    )
    .await
    {
        Ok(Ok(_)) => true,
        _ => false,
    };

    let redis_ok = {
        let mut conn = state.redis.clone();
        match tokio::time::timeout(
            Duration::from_secs(2),
            redis::cmd("PING").query_async::<String>(&mut conn),
        )
        .await
        {
            Ok(Ok(_)) => true,
            _ => false,
        }
    };

    let status = if db_ok && redis_ok {
        SystemHealthStatus::Healthy
    } else if db_ok || redis_ok {
        SystemHealthStatus::Degraded
    } else {
        SystemHealthStatus::Unhealthy
    };

    Ok(Json(ApiResponse::success(HealthCheckData {
        status,
        version: env!("CARGO_PKG_VERSION").to_string(),
        timestamp: chrono::Utc::now().to_rfc3339(),
    })))
}
