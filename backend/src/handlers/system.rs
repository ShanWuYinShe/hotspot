use axum::{extract::State, Json};
use std::time::Duration;

use crate::dto::{
    request::{SystemGetMetaRequest, SystemHealthRequest},
    response::{ApiResponse, ApiVersionMeta, ApiVersionStatus, HealthCheckData},
};
use crate::error::Result;
use crate::state::AppState;
use crate::types::enums::SystemHealthStatus;

/// 获取系统元信息（API 版本状态）
pub async fn get_meta(
    Json(_req): Json<SystemGetMetaRequest>,
) -> Result<Json<ApiResponse<ApiVersionMeta>>> {
    Ok(Json(ApiResponse::success(ApiVersionMeta {
        version: "v1".to_string(),
        status: ApiVersionStatus::Active,
        deprecated_at: None,
        sunset_at: None,
        message: None,
        recommended_version: None,
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
        timestamp: chrono::Utc::now().to_rfc3339(),
    })))
}
