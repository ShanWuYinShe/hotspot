// src/handlers/private_messages.rs
// 私聊消息Handler

use crate::{
    dto::{
        request::{
            DeletePrivateMessageRequest, GetPrivateMessagesRequest, MarkPrivateMessagesReadRequest,
            RecallPrivateMessageRequest, SendPrivateMessageRequest, UpdatePrivateMessageRequest,
            AddPrivateMessageReactionRequest, RemovePrivateMessageReactionRequest, GetPrivateMessageReactionsListRequest,
        },
        response::{ApiResponse, SendPrivateMessageData, PaginatedData, PrivateMessageItem, MessageReactionItem},
    },
    error::Result,
    services::private_message_service::PrivateMessageService,
    state::AppState,
};
use axum::{
    extract::{Extension, State},
    Json,
};


pub async fn send(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<SendPrivateMessageRequest>,
) -> Result<Json<ApiResponse<SendPrivateMessageData>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    let response = service.send(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_history(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetPrivateMessagesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateMessageItem>>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    let response = service.get_history(&req, user_id).await?;

    Ok(Json(ApiResponse::success(response)))
}


pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdatePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.update(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn recall(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RecallPrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.recall(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeletePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.delete(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn mark_as_read(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkPrivateMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.mark_as_read(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn add_reaction(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AddPrivateMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.add_reaction(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn remove_reaction(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemovePrivateMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    service.remove_reaction(&req, user_id).await?;

    Ok(Json(ApiResponse::success_empty()))
}


pub async fn get_reactions_list(
    State(state): State<AppState>,
    Extension(_user_id): Extension<i64>,
    Json(req): Json<GetPrivateMessageReactionsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReactionItem>>>> {
    let service = PrivateMessageService::new(state.db.clone(), state.event_bus.clone());
    let response = service.get_reactions_list(&req).await?;

    Ok(Json(ApiResponse::success(response)))
}
