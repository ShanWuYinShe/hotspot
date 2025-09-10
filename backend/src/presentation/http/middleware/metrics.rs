/// 监控指标中间件
/// 
/// 记录HTTP请求的指标数据，用于Prometheus监控

use axum::{
    extract::MatchedPath,
    http::Request,
    middleware::Next,
    response::IntoResponse,
};
use std::time::Instant;
use crate::utils::metrics::get_app_metrics;

/// HTTP请求监控中间件
pub async fn metrics_middleware(
    req: Request<axum::body::Body>,
    next: Next,
) -> impl IntoResponse {
    let start = Instant::now();
    let method = req.method().to_string();
    
    // 获取路径模式（如 /api/v1/user/:id）
    let path = if let Some(matched_path) = req.extensions().get::<MatchedPath>() {
        matched_path.as_str().to_string()
    } else {
        req.uri().path().to_string()
    };
    
    // 执行请求
    let response = next.run(req).await;
    
    // 记录指标
    let duration = start.elapsed().as_secs_f64();
    let status = response.status().as_u16();
    
    let metrics = get_app_metrics();
    metrics.record_http_request(&method, &path, status, duration);
    
    response
}

/// WebSocket连接监控
pub fn record_ws_connection_opened() {
    let metrics = get_app_metrics();
    metrics.ws_connections.inc();
}

pub fn record_ws_connection_closed() {
    let metrics = get_app_metrics();
    metrics.ws_connections.dec();
}

pub fn record_ws_message_sent() {
    let metrics = get_app_metrics();
    metrics.record_ws_message_sent();
}

pub fn record_ws_message_received() {
    let metrics = get_app_metrics();
    metrics.record_ws_message_received();
}

pub fn record_ws_subscription(room_type: &str, success: bool) {
    let metrics = get_app_metrics();
    let result = if success { "success" } else { "error" };
    metrics.record_ws_subscription(room_type, result);
}
