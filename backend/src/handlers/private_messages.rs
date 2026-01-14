// src/handlers/private_messages.rs
// 私聊消息Handler

use crate::{
    dto::{
        request::{
            AddPrivateMessageReactionRequest, DeletePrivateMessageRequest,
            MarkPrivateMessagesReadRequest, PrivateMessageReactionsRequest,
            PrivateMessagesHistoryRequest, RecallPrivateMessageRequest,
            RemovePrivateMessageReactionRequest, SendPrivateMessageRequest,
            UpdatePrivateMessageRequest,
        },
        response::{
            ApiResponse, MessageReactionItem, PaginatedData, PrivateMessageItem,
            SendPrivateMessageData,
        },
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};
use axum::{extract::State, Json};

pub async fn send(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<SendPrivateMessageRequest>,
) -> Result<Json<ApiResponse<SendPrivateMessageData>>> {
    let response = state
        .private_message_service
        .send(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}

pub async fn history(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<PrivateMessagesHistoryRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateMessageItem>>>> {
    let response = state
        .private_message_service
        .get_history(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}

pub async fn update(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdatePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .update(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn recall(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RecallPrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .recall(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn delete(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DeletePrivateMessageRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .delete(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn mark_as_read(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MarkPrivateMessagesReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .mark_as_read(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn add_reaction(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AddPrivateMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .add_reaction(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn remove_reaction(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<RemovePrivateMessageReactionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_message_service
        .remove_reaction(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn reactions(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<PrivateMessageReactionsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MessageReactionItem>>>> {
    let response = state
        .private_message_service
        .get_reactions_list(&req, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success(response)))
}
