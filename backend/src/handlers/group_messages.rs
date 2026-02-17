//! 群聊消息 Handler

use crate::{
    dto::{
        request::{
            GroupMessagesHistoryRequest, MarkGroupMessagesReadRequest, RecallGroupMessageRequest,
            SendGroupMessageRequest, UpdateGroupMessageRequest,
        },
        response::{ApiResponse, GroupMessageItem, PaginatedData, SendGroupMessageData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};
use axum::{extract::State, Json};

/// 发送群聊消息
pub async fn send(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<SendGroupMessageRequest>,
) -> Result<Json<ApiResponse<SendGroupMessageData>>> {
    let response = state.group_message_service.send(&req, user.user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// 获取群聊消息历史
pub async fn history(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<GroupMessagesHistoryRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMessageItem>>>> {
    let response = state
        .group_message_service
        .get_history(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}

/// 编辑群聊消息
pub async fn update(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .update(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 撤回群聊消息
pub async fn recall(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RecallGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .recall(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 标记群聊消息已读
pub async fn mark_as_read(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MarkGroupMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .mark_as_read(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 标记所有群聊消息已读
pub async fn mark_all_read(
    State(state): State<AppState>,
    user: AuthenticatedUser,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .mark_all_read(user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}
