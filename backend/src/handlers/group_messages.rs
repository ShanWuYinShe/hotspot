// src/handlers/group_messages.rs
// 群聊消息Handler

use crate::{
    dto::{
        request::{
            AddGroupMessageReactionRequest, DeleteGroupMessageRequest,
            GetGroupMessageReactionsRequest, GetGroupMessagesRequest, GetMessageReadStatusRequest,
            MarkGroupMessagesReadRequest, RecallGroupMessageRequest, SendGroupMessageRequest,
            UpdateGroupMessageRequest,
        },
        response::{
            ApiResponse, GroupMessageItem, MessageReactionItem, MessageReadItem, PaginatedData,
            SendGroupMessageData,
        },
    },
    error::Result,
    services::group_message_service::GroupMessageService,
    state::AppState,
};
use axum::{extract::State, Extension, Json};

/// POST /api/v1/protected/group_messages/send - 发送群聊消息
pub async fn send(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SendGroupMessageRequest>,
) -> Result<Json<ApiResponse<SendGroupMessageData>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.send(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_messages/get_history - 获取群聊消息历史
pub async fn get_history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupMessagesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<GroupMessageItem>>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_history(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_messages/update - 编辑群聊消息
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_messages/recall - 撤回群聊消息
pub async fn recall(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RecallGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    service.recall(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_messages/delete - 删除群聊消息
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteGroupMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_messages/mark_as_read - 标记群聊消息已读
pub async fn mark_as_read(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkGroupMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    service.mark_as_read(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_messages/get_read_status - 获取消息已读状态
pub async fn get_read_status(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetMessageReadStatusRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReadItem>>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_read_status(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/group_messages/add_reaction - 添加消息反应
pub async fn add_reaction(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AddGroupMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    service.add_reaction(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/group_messages/get_reactions - 获取消息反应列表
pub async fn get_reactions(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetGroupMessageReactionsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReactionItem>>>> {
    let service = GroupMessageService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_reactions(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}
