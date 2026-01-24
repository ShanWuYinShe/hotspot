//! 通知路由模块
//!
//! 统一的通知 API 路由
//! 所有通知接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::notifications, state::AppState};

/// Token 必须的通知路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/", post(notifications::list_notifications))
        .route("/unread_count", post(notifications::get_unread_count))
        .route("/mark_read", post(notifications::mark_read))
}
