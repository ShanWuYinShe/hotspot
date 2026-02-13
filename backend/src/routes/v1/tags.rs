//! 标签路由模块
//!
//! 统一的标签 API 路由
//! 所有标签接口都是 Token 可选的（公开查询）

use axum::Router;

use crate::state::AppState;

/// Token 可选的标签路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
}
