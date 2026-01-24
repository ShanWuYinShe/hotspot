//! 通知订阅路由模块
//!
//! 统一的通知订阅 API 路由
//! 所有订阅接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::subscriptions, state::AppState};

/// Token 必须的订阅路由
pub fn token_required_routes() -> Router<AppState> {
    // /me 子路由 - 当前用户订阅配置
    let me_routes = Router::new()
        .route("/", post(subscriptions::get_me))
        .route("/update", post(subscriptions::update_me));

    Router::new().nest("/me", me_routes)
}
