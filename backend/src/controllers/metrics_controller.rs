//! 监控指标控制器

use axum::{extract::State, response::IntoResponse};
use std::sync::Arc;

use crate::middleware::auth::AppState;
// use crate::utils::metrics::AppMetrics;

/// 指标处理程序
/// 返回服务的监控指标
pub async fn metrics_handler(
    State(state): State<Arc<AppState>>,
) -> impl IntoResponse {
    // 使用全局单例 metrics 导出 Prometheus 文本
    let body = state.metrics.export_metrics();
    (
        [(axum::http::header::CONTENT_TYPE, "text/plain; version=0.0.4")],
        body,
    )
}
