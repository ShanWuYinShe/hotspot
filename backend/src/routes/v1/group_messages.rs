//! 群聊消息路由模块
//!
//! 统一的群聊消息 API 路由
//! 所有群聊消息接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::group_messages, state::AppState};

/// Token 必须的群聊消息路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/send", post(group_messages::send))
        .route("/history", post(group_messages::history))
        .route("/update", post(group_messages::update))
        .route("/recall", post(group_messages::recall))
        .route("/delete", post(group_messages::delete))
        .route("/mark_as_read", post(group_messages::mark_as_read))
        .route("/read_status", post(group_messages::read_status))
        .route("/add_reaction", post(group_messages::add_reaction))
        .route("/remove_reaction", post(group_messages::remove_reaction))
        .route("/reactions", post(group_messages::reactions))
}
