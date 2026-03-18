//! 地图路由模块
//!
//! 统一的地图 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（地图渲染）
//! - token_required_routes: Token 必须的路由（无）

use axum::{routing::post, Router};

use crate::handlers::map as map_handlers;
use crate::state::AppState;

/// Token 可选的地图路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/canvas", post(map_handlers::canvas))
}
