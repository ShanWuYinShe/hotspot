/// 监控指标控制器
/// 
/// 提供 Prometheus 格式的监控指标数据

use axum::{
    response::{IntoResponse, Response},
    http::{StatusCode, header},
};
use crate::utils::metrics::get_app_metrics;

/// 获取 Prometheus 格式的监控指标
pub async fn get_metrics() -> impl IntoResponse {
    let metrics = get_app_metrics();
    let metrics_data = metrics.export_metrics();
    
    Response::builder()
        .status(StatusCode::OK)
        .header(header::CONTENT_TYPE, "text/plain; charset=utf-8")
        .body(metrics_data)
        .unwrap()
}

/// 获取应用健康状态
pub async fn get_health() -> impl IntoResponse {
    use serde_json::json;
    
    let metrics = get_app_metrics();
    
    // 简单的健康检查逻辑
    let health_status = json!({
        "status": "healthy",
        "timestamp": chrono::Utc::now().to_rfc3339(),
        "version": env!("CARGO_PKG_VERSION"),
        "uptime_seconds": metrics.uptime.get()
    });
    
    Response::builder()
        .status(StatusCode::OK)
        .header(header::CONTENT_TYPE, "application/json")
        .body(health_status.to_string())
        .unwrap()
}