//! 信息流路由模块
//!
//! 统一的信息流 API 路由
//! 信息流接口默认 Token 必须（个性化推荐依赖身份）
//! 搜索建议允许 Token 可选（登录态返回更丰富信息）

use axum::{routing::post, Router};

use crate::{handlers::feed, state::AppState};

/// Token 可选的信息流路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/suggest", post(feed::suggest))
}

/// Token 必须的信息流路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/moments", post(feed::get_moments))
        .route("/groups", post(feed::get_groups))
        .route("/users", post(feed::get_users))
        .route("/trending", post(feed::get_trending))
        .route("/feedback", post(feed::feedback))
}
