//! 系统路由模块
//!
//! 统一的系统 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（健康检查）
//! - token_required_routes: Token 必须的路由（预留）

use axum::{routing::post, Router};

use crate::{handlers::system, state::AppState};

/// Token 可选的系统路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 系统健康检查
        .route("/health", post(system::health))
}

/// Token 必须的系统路由
/// 额度敏感的系统能力需要登录后调用，避免匿名刷第三方额度
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
}
