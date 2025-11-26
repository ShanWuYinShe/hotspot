/// 消息通用API处理器
///
/// 提供跨私聊和群聊的通用消息功能
///
/// ⚠️ 架构决策：
/// - ❌ 消息搜索由前端本地实现（基于客户端缓存的消息数据）
/// - ❌ 未读汇总由前端本地统计（基于聊天列表的未读计数）
/// - ✅ 批量标记已读由后端实现（跨聊天批量操作）
use axum::{Extension, Json};

use crate::{
    dto::{
        request::{MarkAllMessagesReadRequest, MarkAsReadBatchRequest},
        response::{ApiResponse, BatchResultData},
    },
    error::AppError,
    services::message_common_service::MessageCommonService,
    state::AppState,
};

/// 标记所有消息已读
#[tracing::instrument(name = "mark_all_as_read", skip(state))]
pub async fn mark_all_as_read(
    axum::extract::State(state): axum::extract::State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<MarkAllMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let service = MessageCommonService::new(state.db.clone());
    service.mark_all_as_read(user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 批量标记消息已读
#[tracing::instrument(name = "mark_as_read_batch", skip(state))]
pub async fn mark_as_read_batch(
    axum::extract::State(state): axum::extract::State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkAsReadBatchRequest>,
) -> Result<Json<ApiResponse<BatchResultData>>, AppError> {
    let service = MessageCommonService::new(state.db.clone());
    let result = service.mark_as_read_batch(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}
