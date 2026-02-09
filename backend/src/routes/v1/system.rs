//! 系统路由模块
//!
//! 统一的系统 API 路由：
//! - token_optional_routes: Token 可选的路由（健康检查、系统信息、客户端配置、公告等）

use axum::{routing::post, Router};

use crate::{
    handlers::{announcements, system, system_config},
    state::AppState,
};

/// Token 可选的系统路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 系统健康检查
        .route("/health", post(system::health))
        // 系统元信息
        .route("/get_meta", post(system::get_meta))
        // 客户端配置
        .route(
            "/get_client_config",
            post(system_config::fetch_global_settings),
        )
        // 系统公告列表
        .route("/list_announcements", post(announcements::public_list))
}
