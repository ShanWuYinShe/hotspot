use axum::{body::Body, extract::Request, http::header, middleware::Next, response::Response};

const LOG_PREVIEW_BYTES: usize = 8 * 1024; // 与响应日志策略保持一致

/// 记录 JSON 请求体，便于排查客户端无法直接观察的请求内容。
/// - 仅在 Content-Type 含 `json` 时记录，避免在文件/表单上传等场景产生大量日志
/// - 会完整读取请求体并重新构造 Request 交给下游，确保业务逻辑不受影响
pub async fn request_logger(req: Request, next: Next) -> Response {
    if !should_log_body(&req) {
        return next.run(req).await;
    }

    let method = req.method().clone();
    let uri = req.uri().clone();

    let (parts, body) = req.into_parts();
    let bytes = match axum::body::to_bytes(body, usize::MAX).await {
        Ok(bytes) => bytes,
        Err(error) => {
            tracing::warn!(
                method = %method,
                uri = %uri,
                error = %error,
                "failed to read request body for logging"
            );
            return next.run(Request::from_parts(parts, Body::empty())).await;
        }
    };

    let truncated = bytes.len() > LOG_PREVIEW_BYTES;
    let preview_bytes = if truncated {
        &bytes[..LOG_PREVIEW_BYTES]
    } else {
        bytes.as_ref()
    };
    let preview = String::from_utf8_lossy(preview_bytes);

    tracing::info!(
        method = %method,
        uri = %uri,
        request_size_bytes = bytes.len(),
        truncated = truncated,
        request_body_preview = %preview,
        "request body"
    );

    let req = Request::from_parts(parts, Body::from(bytes));
    next.run(req).await
}

fn should_log_body(req: &Request) -> bool {
    matches!(req.method().as_str(), "POST" | "PUT" | "PATCH" | "DELETE")
        && req
            .headers()
            .get(header::CONTENT_TYPE)
            .and_then(|value| value.to_str().ok())
            .map(|value| value.to_ascii_lowercase().contains("json"))
            .unwrap_or(false)
}
