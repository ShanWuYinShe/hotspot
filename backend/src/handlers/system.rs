use axum::{extract::State, Json};

use crate::{
    dto::response::ApiResponse,
    error::Result,
    state::AppState,
};

/// POST /api/v1/public/system/health
pub async fn health() -> Result<Json<ApiResponse<serde_json::Value>>> {
    Ok(Json(ApiResponse::success(serde_json::json!({
        "status": "healthy",
        "timestamp": chrono::Utc::now().timestamp(),
    }))))
}

/// POST /api/v1/public/system/config
pub async fn config(State(state): State<AppState>) -> Result<Json<ApiResponse<serde_json::Value>>> {
    // 返回公开的配置信息（不包含敏感信息）
    Ok(Json(ApiResponse::success(serde_json::json!({
        "version": env!("CARGO_PKG_VERSION"),
        "server_time": chrono::Utc::now().timestamp(),
        "features": {
            "websocket": true,
            "file_upload": true,
            "geolocation": true,
        },
        "limits": {
            "max_file_size": 10 * 1024 * 1024, // 10MB
            "max_message_length": 5000,
            "max_group_members": 500,
        }
    }))))
}

/// POST /api/v1/protected/system/stats
pub async fn stats(State(state): State<AppState>) -> Result<Json<ApiResponse<serde_json::Value>>> {
    // 获取系统统计信息
    let online_users = state.ws_manager.online_user_count().await;
    let subscription_stats = state.ws_subscription.get_stats().await;

    Ok(Json(ApiResponse::success(serde_json::json!({
        "online_users": online_users,
        "subscriptions": {
            "total_users": subscription_stats.total_users,
            "total_types": subscription_stats.total_subscription_types,
            "total_subscriptions": subscription_stats.total_subscriptions,
        },
        "timestamp": chrono::Utc::now().timestamp(),
    }))))
}

