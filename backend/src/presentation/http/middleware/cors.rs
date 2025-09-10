/// CORS中间件
/// 
/// 处理跨域请求的中间件
use axum::http::{HeaderValue, Method};
use tower_http::cors::{CorsLayer, Any};

/// 创建CORS中间件层
pub fn create_cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(Any)
        .allow_methods([
            Method::GET,
            Method::POST,
            Method::PUT,
            Method::DELETE,
            Method::OPTIONS,
        ])
        .allow_headers(Any)
        .allow_credentials(true)
}

/// 开发环境CORS配置
pub fn development_cors() -> CorsLayer {
    CorsLayer::new()
        .allow_origin("http://localhost:3000".parse::<HeaderValue>().unwrap())
        .allow_origin("http://127.0.0.1:3000".parse::<HeaderValue>().unwrap())
        .allow_methods([
            Method::GET,
            Method::POST,
            Method::PUT,
            Method::DELETE,
            Method::OPTIONS,
        ])
        .allow_headers(Any)
        .allow_credentials(true)
}

/// 生产环境CORS配置
pub fn production_cors(allowed_origins: Vec<&str>) -> CorsLayer {
    let mut cors = CorsLayer::new()
        .allow_methods([
            Method::GET,
            Method::POST,
            Method::PUT,
            Method::DELETE,
            Method::OPTIONS,
        ])
        .allow_headers(Any)
        .allow_credentials(true);

    for origin in allowed_origins {
        if let Ok(header_value) = origin.parse::<HeaderValue>() {
            cors = cors.allow_origin(header_value);
        }
    }

    cors
}
