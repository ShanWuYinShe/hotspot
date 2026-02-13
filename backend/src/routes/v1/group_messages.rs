//! 群聊消息路由模块
//!
//! 统一的群聊消息 API 路由
//! 所有群聊消息接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::group_messages, state::AppState};

/// Token 必须的群聊消息路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        .route("/send", post(group_messages::send))
        .route("/get_history", post(group_messages::history))
        .route("/update", post(group_messages::update))
        .route("/recall", post(group_messages::recall));

    // /me 子路由
    let me_routes = Router::new()
        .route("/mark_as_read", post(group_messages::mark_as_read))
        .route("/mark_all_read", post(group_messages::mark_all_read));

    main_routes.nest("/me", me_routes)
}
