//! 系统公告路由模块
//!
//! 统一的系统公告 API 路由
//! 所有公告接口都是 Token 可选的（公开查询）

use axum::{routing::post, Router};

use crate::{handlers::announcements, state::AppState};

/// Token 可选的公告路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/", post(announcements::public_list))
}
