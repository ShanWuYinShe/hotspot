// src/routes/v1/protected/private_messages.rs
// 私聊消息路由

use crate::{handlers::private_messages, state::AppState};
use axum::{routing::post, Router};

/// Protected 私聊消息路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 第 3.8 节定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/send", post(private_messages::send)) // 发送私聊消息
        .route("/get_history", post(private_messages::get_history)) // 获取私聊消息历史
        .route("/update", post(private_messages::update)) // 编辑私聊消息
        .route("/recall", post(private_messages::recall)) // 撤回私聊消息
        .route("/delete", post(private_messages::delete)) // 删除私聊消息
        .route("/mark_as_read", post(private_messages::mark_as_read)) // 标记私聊消息已读
}


