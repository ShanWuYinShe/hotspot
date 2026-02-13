//! 通知中心路由模块

use axum::{routing::post, Router};

use crate::{handlers::notifications, state::AppState};

/// Token 必须的通知路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/list", post(notifications::list))
        .route("/unread_count", post(notifications::unread_count))
        .route("/mark_as_read", post(notifications::mark_as_read))
        .route("/mark_all_read", post(notifications::mark_all_read))
}
