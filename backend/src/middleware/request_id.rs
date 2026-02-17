use axum::{body::Body, http::HeaderValue, middleware::Next, response::Response};
use uuid::Uuid;

/// request_id 中间件
///
/// - 优先回显客户端传入的 `X-Request-Id`（大小写不敏感）
/// - 若客户端未提供，则服务端生成 UUID
/// - 始终写入响应头 `x-request-id`
/// - spec: request_id 仅通过 header 传递，不注入 body
pub async fn request_id_middleware(req: axum::http::Request<Body>, next: Next) -> Response {
    let request_id =
        extract_request_id(req.headers()).unwrap_or_else(|| Uuid::new_v4().to_string());

    let mut req = req;
    if let Ok(v) = HeaderValue::from_str(&request_id) {
        req.headers_mut().insert("x-request-id", v);
    }

    let mut resp = next.run(req).await;

    // spec: X-Request-Id 通过响应头传递
    if let Ok(v) = HeaderValue::from_str(&request_id) {
        resp.headers_mut().insert("x-request-id", v);
    }

    resp
}

fn extract_request_id(headers: &axum::http::HeaderMap) -> Option<String> {
    let raw = headers.get("x-request-id")?;
    let s = raw.to_str().ok()?.trim();
    if s.is_empty() {
        None
    } else {
        Some(s.to_string())
    }
}
