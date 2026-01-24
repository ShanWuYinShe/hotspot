//! 信息流路由模块
//!
//! 统一的信息流 API 路由
//! 所有信息流接口都是 Token 必须的（个性化推荐依赖身份）

use axum::{routing::post, Router};

use crate::{handlers::feed, state::AppState};

/// Token 必须的信息流路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/moments", post(feed::get_moments))
        .route("/groups", post(feed::get_groups))
        .route("/users", post(feed::get_users))
        .route("/trending", post(feed::get_trending))
        .route("/feedback", post(feed::feedback))
}
