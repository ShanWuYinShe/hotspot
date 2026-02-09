use axum::{extract::State, Json};
use serde::Serialize;
use std::time::Duration;

use crate::dto::{
    request::{SystemGetMetaRequest, SystemHealthRequest},
    response::{ApiResponse, HealthCheckData},
};
use crate::error::Result;
use crate::state::AppState;
use crate::types::enums::SystemHealthStatus;

/// 系统元信息
#[derive(Debug, Serialize)]
pub struct SystemMetaData {
    pub api_version: String,
    pub server_time: String,
}

/// 获取系统元信息
pub async fn get_meta(
    Json(_req): Json<SystemGetMetaRequest>,
) -> Result<Json<ApiResponse<SystemMetaData>>> {
    Ok(Json(ApiResponse::success(SystemMetaData {
        api_version: "v1".to_string(),
        server_time: chrono::Utc::now().to_rfc3339(),
    })))
}

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
