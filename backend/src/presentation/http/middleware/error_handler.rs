/// 统一错误处理中间件
/// 
/// 提供全局的错误捕获和响应格式化，确保所有API响应格式一致
use axum::{
    extract::Request,
    http::StatusCode,
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use tracing::{error, warn};
use std::panic;

/// 全局错误处理中间件
pub async fn error_handler_middleware(
    request: Request,
    next: Next,
) -> Response {
    // 设置panic hook来捕获panic
    let original_hook = panic::take_hook();
    panic::set_hook(Box::new(|panic_info| {
        error!("Panic occurred: {:?}", panic_info);
    }));

    let response = next.run(request).await;
    
    // 恢复原始panic hook
    panic::set_hook(original_hook);
    
    // 检查响应状态码，对非200状态码进行统一处理
    match response.status() {
        StatusCode::OK => response,
        status_code => {
            warn!("Non-OK status code returned: {}", status_code);
            
            // 将HTTP错误状态码转换为统一的JSON响应格式
            let (error_code, message) = map_http_status_to_error_code(status_code);
            
            let error_response = json!({
                "code": error_code,
                "msg": message,
                "data": null
            });
            
            // 根据hotspot.md规范，统一返回HTTP 200状态码
            (StatusCode::OK, Json(error_response)).into_response()
        }
    }
}

/// 将HTTP状态码映射到业务错误码
fn map_http_status_to_error_code(status: StatusCode) -> (u16, &'static str) {
    match status {
        StatusCode::BAD_REQUEST => (1000, "请求参数错误"),
        StatusCode::UNAUTHORIZED => (1002, "未授权访问"),
        StatusCode::FORBIDDEN => (1003, "权限不足"),
        StatusCode::NOT_FOUND => (1004, "资源不存在"),
        StatusCode::CONFLICT => (1005, "资源冲突"),
        StatusCode::UNPROCESSABLE_ENTITY => (1000, "数据验证失败"),
        StatusCode::TOO_MANY_REQUESTS => (1007, "请求过于频繁"),
        StatusCode::INTERNAL_SERVER_ERROR => (5000, "服务器内部错误"),
        StatusCode::BAD_GATEWAY => (5002, "外部服务错误"),
        StatusCode::SERVICE_UNAVAILABLE => (5001, "服务暂时不可用"),
        StatusCode::GATEWAY_TIMEOUT => (5002, "服务响应超时"),
        _ => (5000, "未知错误"),
    }
}

/// 请求超时处理中间件
pub async fn timeout_handler_middleware(
    request: Request,
    next: Next,
) -> Response {
    use tokio::time::{timeout, Duration};
    
    // 设置30秒超时
    let timeout_duration = Duration::from_secs(30);
    
    match timeout(timeout_duration, next.run(request)).await {
        Ok(response) => response,
        Err(_) => {
            error!("Request timeout after {:?}", timeout_duration);
            
            let error_response = json!({
                "code": 5003,
                "msg": "请求处理超时",
                "data": null
            });
            
            (StatusCode::OK, Json(error_response)).into_response()
        }
    }
}

/// 请求大小限制中间件
pub async fn request_size_limit_middleware(
    request: Request,
    next: Next,
) -> Response {
    // 检查Content-Length头
    if let Some(content_length) = request.headers().get("content-length") {
        if let Ok(length_str) = content_length.to_str() {
            if let Ok(length) = length_str.parse::<usize>() {
                // 限制请求体大小为10MB
                const MAX_REQUEST_SIZE: usize = 10 * 1024 * 1024;
                
                if length > MAX_REQUEST_SIZE {
                    warn!("Request size {} exceeds limit {}", length, MAX_REQUEST_SIZE);
                    
                    let error_response = json!({
                        "code": 1000,
                        "msg": "请求体过大",
                        "data": null
                    });
                    
                    return (StatusCode::OK, Json(error_response)).into_response();
                }
            }
        }
    }
    
    next.run(request).await
}

/// 安全头中间件
pub async fn security_headers_middleware(
    request: Request,
    next: Next,
) -> Response {
    let mut response = next.run(request).await;
    
    // 添加安全相关的响应头
    let headers = response.headers_mut();
    
    // 防止XSS攻击
    headers.insert("X-Content-Type-Options", "nosniff".parse().unwrap());
    headers.insert("X-Frame-Options", "DENY".parse().unwrap());
    headers.insert("X-XSS-Protection", "1; mode=block".parse().unwrap());
    
    // 内容安全策略
    headers.insert(
        "Content-Security-Policy",
        "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'".parse().unwrap()
    );
    
    // 严格传输安全（HTTPS环境）
    if cfg!(not(debug_assertions)) {
        headers.insert(
            "Strict-Transport-Security",
            "max-age=31536000; includeSubDomains".parse().unwrap()
        );
    }
    
    response
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_map_http_status_to_error_code() {
        assert_eq!(map_http_status_to_error_code(StatusCode::BAD_REQUEST), (1000, "请求参数错误"));
        assert_eq!(map_http_status_to_error_code(StatusCode::UNAUTHORIZED), (1002, "未授权访问"));
        assert_eq!(map_http_status_to_error_code(StatusCode::FORBIDDEN), (1003, "权限不足"));
        assert_eq!(map_http_status_to_error_code(StatusCode::NOT_FOUND), (1004, "资源不存在"));
        assert_eq!(map_http_status_to_error_code(StatusCode::INTERNAL_SERVER_ERROR), (5000, "服务器内部错误"));
    }
}
