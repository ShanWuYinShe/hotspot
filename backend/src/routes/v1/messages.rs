//! 消息通用路由模块
//!
//! 提供跨私聊和群聊的通用消息功能
//! 所有消息通用接口都是 Token 必须的
//!
//! 架构决策：
//! - 消息搜索由客户端本地实现
//! - 未读汇总由客户端本地统计
//! - 标记全部已读由后端实现

use axum::{routing::post, Router};

use crate::{handlers::message_common, state::AppState};

/// Token 必须的消息通用路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new().route("/mark_all_as_read", post(message_common::mark_all_as_read))
}
