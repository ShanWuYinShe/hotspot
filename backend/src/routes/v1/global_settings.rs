//! 全局配置路由模块
//!
//! 统一的全局配置 API 路由
//! 所有配置接口都是 Token 可选的（公开查询）

use axum::{routing::post, Router};

use crate::{handlers::system_config, state::AppState};

/// Token 可选的全局配置路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/fetch", post(system_config::fetch_global_settings))
}
