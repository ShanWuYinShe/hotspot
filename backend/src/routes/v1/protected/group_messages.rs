// src/routes/v1/protected/group_messages.rs
// 群聊消息路由

use crate::{handlers::group_messages, state::AppState};
use axum::{routing::post, Router};

/// Protected 群聊消息路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 第 3.9 节定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/send", post(group_messages::send)) // ✅ 发送群聊消息
        .route("/get_history", post(group_messages::get_history)) // ✅ 获取群聊消息历史
        .route("/update", post(group_messages::update)) // ✅ 编辑群聊消息
        .route("/recall", post(group_messages::recall)) // ✅ 撤回群聊消息
        .route("/delete", post(group_messages::delete)) // ✅ 删除群聊消息
        .route("/mark_as_read", post(group_messages::mark_as_read)) // ✅ 标记群聊消息已读
        .route("/get_read_status", post(group_messages::get_read_status)) // ✅ 获取消息已读状态
        .route("/add_reaction", post(group_messages::add_reaction)) // ✅ 添加消息反应
        .route("/get_reactions", post(group_messages::get_reactions)) // ✅ 获取消息反应列表
}
