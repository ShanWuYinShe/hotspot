//! 监控指标控制器

use axum::{Json, extract::State};
use std::sync::Arc;

use crate::middleware::auth::AppState;

/// 指标处理程序
/// 返回服务的监控指标
pub async fn metrics_handler(
    State(_state): State<Arc<AppState>>,
    Json(_request): Json<serde_json::Value>, // 添加空的请求体参数
) -> Json<serde_json::Value> {
    // 返回基本的指标信息
    Json(serde_json::json!({
        "status": "ok",
        "timestamp": chrono::Utc::now().to_rfc3339(),
        "version": env!("CARGO_PKG_VERSION")
    }))
}
