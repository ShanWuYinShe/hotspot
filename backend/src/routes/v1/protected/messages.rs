use crate::{handlers::message_common, state::AppState};
use axum::{routing::post, Router};

/// 消息通用API路由
///
/// 提供跨私聊和群聊的通用消息功能
///
/// ⚠️ 架构决策：
/// - ❌ 消息搜索由客户端本地实现
/// - ❌ 未读汇总由客户端本地统计
/// - ✅ 标记全部已读由后端实现

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/mark_all_as_read", post(message_common::mark_all_as_read))
}
