use crate::{handlers::message_common, state::AppState};
/// 消息通用API路由
///
/// 根据 API 规范 3.10 节，提供跨私聊和群聊的通用消息功能
///
/// ⚠️ 架构决策：
/// - ❌ 消息搜索由前端本地实现（已移除 /search 路由）
/// - ❌ 未读汇总由前端本地统计（已移除 /get_unread_summary 路由）
/// - ✅ 批量标记已读由后端实现
///
/// 现有路由：
/// - POST /api/v1/protected/messages/mark_all_as_read - 标记所有消息已读
/// - POST /api/v1/protected/messages/mark_as_read_batch - 批量标记消息已读
use axum::{routing::post, Router};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/mark_all_as_read", post(message_common::mark_all_as_read))
        .route(
            "/mark_as_read_batch",
            post(message_common::mark_as_read_batch),
        )
}
