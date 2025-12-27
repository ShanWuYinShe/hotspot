// src/routes/v1/protected/private_messages.rs
// 私聊消息路由

use crate::{handlers::private_messages, state::AppState};
use axum::{routing::post, Router};

/// Protected 私聊消息路由
///
/// 这些接口需要认证，必须提供有效的 access_token
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/send", post(private_messages::send)) // 发送私聊消息
        .route("/history", post(private_messages::history)) // 获取私聊消息历史
        .route("/update", post(private_messages::update)) // 编辑私聊消息
        .route("/recall", post(private_messages::recall)) // 撤回私聊消息
        .route("/delete", post(private_messages::delete)) // 删除私聊消息
        .route("/mark_as_read", post(private_messages::mark_as_read)) // 标记私聊消息已读
        .route("/add_reaction", post(private_messages::add_reaction)) // 添加消息反应
        .route("/remove_reaction", post(private_messages::remove_reaction)) // 移除消息反应
        .route("/reactions", post(private_messages::reactions)) // 获取消息反应列表
}
