//! 推送设备管理路由模块
//!
//! 统一的设备令牌 API 路由
//! 所有设备令牌接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::device_tokens, state::AppState};

/// Token 必须的设备令牌路由
pub fn token_required_routes() -> Router<AppState> {
    let me_routes = Router::new()
        .route("/register", post(device_tokens::register_device_token))
        .route("/unregister", post(device_tokens::unregister_device_token))
        .route("/", post(device_tokens::me));

    Router::new().nest("/me", me_routes)
}
