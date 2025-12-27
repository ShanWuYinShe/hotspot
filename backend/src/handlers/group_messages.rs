// src/handlers/group_messages.rs
// 群聊消息Handler

use crate::{
    dto::{
        request::{
            AddGroupMessageReactionRequest, DeleteGroupMessageRequest,
            GroupMessageReactionsRequest, GroupMessagesHistoryRequest,
            MarkGroupMessagesReadRequest, MessageReadStatusRequest, RecallGroupMessageRequest,
            RemoveGroupMessageReactionRequest, SendGroupMessageRequest, UpdateGroupMessageRequest,
        },
        response::{
            ApiResponse, GroupMessageItem, MessageReactionItem, MessageReadItem, PaginatedData,
            SendGroupMessageData,
        },
    },
    error::Result,
    state::AppState,
};
use axum::{extract::State, Extension, Json};

/// 发送群聊消息
pub async fn send(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SendGroupMessageRequest>,
) -> Result<Json<ApiResponse<SendGroupMessageData>>> {
    let response = state.group_message_service.send(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// 获取群聊消息历史
pub async fn history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GroupMessagesHistoryRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMessageItem>>>> {
    let response = state
        .group_message_service
        .get_history(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}

/// 编辑群聊消息
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_message_service.update(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 撤回群聊消息
pub async fn recall(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RecallGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_message_service.recall(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 删除群聊消息
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.group_message_service.delete(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 标记群聊消息已读
pub async fn mark_as_read(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkGroupMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .mark_as_read(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 获取消息已读状态
pub async fn read_status(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MessageReadStatusRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReadItem>>>> {
    let response = state
        .group_message_service
        .get_read_status(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}

/// 添加消息反应
pub async fn add_reaction(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AddGroupMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .add_reaction(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 移除消息反应
pub async fn remove_reaction(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemoveGroupMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .group_message_service
        .remove_reaction(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 获取消息反应列表
pub async fn reactions(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GroupMessageReactionsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReactionItem>>>> {
    let response = state
        .group_message_service
        .get_reactions_list(&req, user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}
