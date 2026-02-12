//! 信息流路由模块
//!
//! 统一的信息流 API 路由
//! 信息流接口默认 Token 必须（个性化推荐依赖身份）

use axum::{routing::post, Router};

use crate::{handlers::feed, state::AppState};

/// Token 可选的信息流路由（公开可访问）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/list_trending", post(feed::get_trending))
}

/// Token 必须的信息流路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        .route("/list_moments", post(feed::get_moments))
        .route("/list_groups", post(feed::get_groups))
        .route("/list_users", post(feed::get_users));

    main_routes
}
