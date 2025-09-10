/// 监控指标路由
/// 
/// 提供 Prometheus 监控指标和健康检查端点

use axum::{
    routing::post,
    Router,
};
use crate::presentation::http::controllers::metrics_controller::{get_metrics, get_health};

/// 创建监控路由
/// 
/// 路径：
/// - POST /metrics - Prometheus 格式的监控指标
/// - POST /health - 应用健康状态
pub fn create_metrics_routes() -> Router {
    Router::new()
        .route("/metrics", post(get_metrics))
        .route("/health", post(get_health))
}
