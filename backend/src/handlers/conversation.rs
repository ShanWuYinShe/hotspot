use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            ListConversationsRequest, PinConversationRequest, UnpinConversationRequest,
            MuteConversationRequest, UnmuteConversationRequest, DeleteConversationRequest,
        },
        response::{ApiResponse, PaginatedResponse, ConversationResponse},
    },
    error::{AppError, Result},
    middleware::auth::extract_user_id,
    repositories::conversation_repo::ConversationRepository,
    state::AppState,
};

/// POST /api/v1/protected/conversation/list
pub async fn list(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<ListConversationsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<ConversationResponse>>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    
    let limit = req_body.limit.unwrap_or(20).min(100);
    let offset = req_body.cursor.unwrap_or(0);
    
    let conversations = conversation_repo.list_by_user(user_id, limit + 1, offset).await?;
    
    let has_more = conversations.len() > limit as usize;
    let items: Vec<ConversationResponse> = conversations
        .into_iter()
        .take(limit as usize)
        .map(|c| ConversationResponse {
            target_id: c.target_id,
            target_type: c.target_type,
            last_message_id: c.last_message_id.map(|id| id.to_string()),
            last_message_at: c.last_message_at.map(|dt| dt.and_utc().timestamp()),
            unread_count: c.unread_count,
            is_pinned: c.is_pinned,
            is_muted: c.is_muted,
        })
        .collect();
    
    let next_cursor = if has_more {
        Some(offset + limit)
    } else {
        None
    };
    
    Ok(Json(ApiResponse::success(PaginatedResponse {
        items,
        has_more,
        next_cursor,
    })))
}

/// POST /api/v1/protected/conversation/pin
pub async fn pin(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<PinConversationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    conversation_repo
        .pin_conversation(user_id, &req_body.target_id, &req_body.target_type)
        .await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "会话已置顶"
    }))))
}

/// POST /api/v1/protected/conversation/unpin
pub async fn unpin(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UnpinConversationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    conversation_repo
        .unpin_conversation(user_id, &req_body.target_id, &req_body.target_type)
        .await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消置顶"
    }))))
}

/// POST /api/v1/protected/conversation/mute
pub async fn mute(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<MuteConversationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    conversation_repo
        .mute_conversation(user_id, &req_body.target_id, &req_body.target_type)
        .await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "会话已静音"
    }))))
}

/// POST /api/v1/protected/conversation/unmute
pub async fn unmute(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UnmuteConversationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    conversation_repo
        .unmute_conversation(user_id, &req_body.target_id, &req_body.target_type)
        .await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消静音"
    }))))
}

/// POST /api/v1/protected/conversation/delete
pub async fn delete(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<DeleteConversationRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let conversation_repo = ConversationRepository::new(&state.db);
    conversation_repo
        .delete_conversation(user_id, &req_body.target_id, &req_body.target_type)
        .await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "会话已删除"
    }))))
}
