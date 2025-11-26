// src/handlers/private_messages.rs
// 私聊消息Handler

use crate::{
    dto::{
        request::{
            DeletePrivateMessageRequest, GetPrivateMessagesRequest, MarkPrivateMessagesReadRequest,
            RecallPrivateMessageRequest, SendPrivateMessageRequest, UpdatePrivateMessageRequest,
        },
        response::{ApiResponse, SendPrivateMessageData, PaginatedData, PrivateMessageItem},
    },
    error::Result,
    services::private_message_service::PrivateMessageService,
    state::AppState,
};
use axum::{
    extract::{Extension, State},
    Json,
};

/// POST /api/v1/protected/private_messages/send
pub async fn send(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SendPrivateMessageRequest>,
) -> Result<Json<ApiResponse<SendPrivateMessageData>>> {
    let service = PrivateMessageService::new(state.db.clone());
    let response = service.send(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/private_messages/get_history
pub async fn get_history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetPrivateMessagesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateMessageItem>>>> {
    let service = PrivateMessageService::new(state.db.clone());
    let response = service.get_history(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/private_messages/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdatePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone());
    service.update(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/private_messages/recall
pub async fn recall(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RecallPrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone());
    service.recall(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/private_messages/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeletePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone());
    service.delete(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/private_messages/mark_as_read
pub async fn mark_as_read(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkPrivateMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone());
    service.mark_as_read(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}
