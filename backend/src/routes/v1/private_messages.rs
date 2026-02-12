//! 私聊消息路由模块
//!
//! 统一的私聊消息 API 路由
//! 所有私聊消息接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::private_messages, state::AppState};

/// Token 必须的私聊消息路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        .route("/send", post(private_messages::send))
        .route("/get_history", post(private_messages::history))
        .route("/update", post(private_messages::update))
        .route("/recall", post(private_messages::recall));

    // /me 子路由
    let me_routes = Router::new().route("/mark_as_read", post(private_messages::mark_as_read));

    main_routes.nest("/me", me_routes)
}
