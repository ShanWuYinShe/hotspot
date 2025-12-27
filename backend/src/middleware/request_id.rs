use axum::{
    body::{to_bytes, Body},
    http::{header::CONTENT_TYPE, HeaderValue},
    middleware::Next,
    response::Response,
};
use serde_json::Value;
use tracing::warn;
use uuid::Uuid;

/// request_id 中间件
///
/// - 优先回显客户端传入的 `X-Request-Id`（大小写不敏感）
/// - 若客户端未提供，则服务端生成 UUID
/// - 始终写入响应头 `x-request-id`
/// - 对 `application/json` 响应体自动注入 `request_id` 字段（避免 handler 漏写）
pub async fn request_id_middleware(req: axum::http::Request<Body>, next: Next) -> Response {
    let request_id =
        extract_request_id(req.headers()).unwrap_or_else(|| Uuid::new_v4().to_string());

    // 仅负责“注入/回显 request_id”，不创建额外 span（避免日志出现多层 spans）
    let mut req = req;
    if let Ok(v) = HeaderValue::from_str(&request_id) {
        req.headers_mut().insert("x-request-id", v);
    }

    let resp = next.run(req).await;
    inject_request_id(resp, &request_id).await
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

async fn inject_request_id(resp: Response, request_id: &str) -> Response {
    let mut resp = resp;

    // 1) header 必返
    if let Ok(v) = HeaderValue::from_str(request_id) {
        resp.headers_mut().insert("x-request-id", v);
    }

    // 2) 仅对 JSON 响应注入 body.request_id
    let content_type = resp
        .headers()
        .get(CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");
    if !content_type.starts_with("application/json") {
        return resp;
    }

    let (parts, body) = resp.into_parts();
    let bytes = match to_bytes(body, usize::MAX).await {
        Ok(b) => b,
        Err(e) => {
            // body 读取失败时无法再恢复原 body，只能返回空 body，但必须保留 request_id header
            warn!(error = %e, request_id = %request_id, "failed to read response body for request_id injection");
            let mut resp = Response::from_parts(parts, Body::from(Vec::<u8>::new()));
            if let Ok(v) = HeaderValue::from_str(request_id) {
                resp.headers_mut().insert("x-request-id", v);
            }
            return resp;
        }
    };

    let mut value: Value = match serde_json::from_slice(&bytes) {
        Ok(v) => v,
        Err(_) => {
            // JSON parse 失败则原样返回，但仍保留 header
            let mut resp = Response::from_parts(parts, Body::from(bytes));
            if let Ok(v) = HeaderValue::from_str(request_id) {
                resp.headers_mut().insert("x-request-id", v);
            }
            return resp;
        }
    };

    if let Some(obj) = value.as_object_mut() {
        obj.insert(
            "request_id".to_string(),
            Value::String(request_id.to_string()),
        );
    }

    let new_bytes = serde_json::to_vec(&value).unwrap_or_else(|_| bytes.to_vec());
    let mut resp = Response::from_parts(parts, Body::from(new_bytes));
    if let Ok(v) = HeaderValue::from_str(request_id) {
        resp.headers_mut().insert("x-request-id", v);
    }
    resp
}
