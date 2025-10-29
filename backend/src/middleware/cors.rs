use tower_http::cors::{Any, CorsLayer};

/// 创建 CORS 中间件
pub fn create_cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any)
}

