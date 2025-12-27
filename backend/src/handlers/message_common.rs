/// 消息通用API处理器
///
/// 提供跨私聊和群聊的通用消息功能
///
/// ⚠️ 架构决策：
/// - ❌ 消息搜索由客户端本地实现（基于客户端缓存的消息数据）
/// - ❌ 未读汇总由客户端本地统计（基于聊天列表的未读计数）
/// - ✅ 标记全部已读由后端实现（mark_all_as_read）
use axum::{Extension, Json};

use crate::{
    dto::{request::MarkAllMessagesReadRequest, response::ApiResponse},
    error::AppError,
    state::AppState,
};

/// 标记所有消息已读
#[tracing::instrument(name = "mark_all_as_read", skip(state))]
pub async fn mark_all_as_read(
    axum::extract::State(state): axum::extract::State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<MarkAllMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state
        .message_common_service
        .mark_all_as_read(user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}
