use axum::{body::Body, extract::Request, http::header, middleware::Next, response::Response};

const LOG_PREVIEW_BYTES: usize = 8 * 1024; // 8KB 预览，避免日志过长

/// 记录 JSON 响应体，方便排查 App 端无法直接看到的响应内容。
///
/// - 仅在 Content-Type 含 `json` 时记录，避免影响文件/流式响应
/// - 记录前会完整缓冲响应体，再重新构造 Response 返回给客户端
/// - 日志中只输出前 8KB，超过部分标记 `truncated=true`
pub async fn response_logger(req: Request, next: Next) -> Response {
    let method = req.method().clone();
    let uri = req.uri().clone();

    let response = next.run(req).await;

    if !should_log_body(&response) {
        return response;
    }

    let (parts, body) = response.into_parts();
    let bytes = axum::body::to_bytes(body, usize::MAX)
        .await
        .unwrap_or_default();

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
        status = parts.status.as_u16(),
        truncated = truncated,
        response_preview = %preview,
        "response body"
    );

    Response::from_parts(parts, Body::from(bytes))
}

fn should_log_body(response: &Response) -> bool {
    if matches!(response.status().as_u16(), 204 | 304) {
        return false;
    }

    response
        .headers()
        .get(header::CONTENT_TYPE)
        .and_then(|value| value.to_str().ok())
        .map(|value| value.to_ascii_lowercase().contains("json"))
        .unwrap_or(false)
}
