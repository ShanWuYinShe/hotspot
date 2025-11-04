use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            SendMessageRequest, EditMessageRequest, RecallMessageRequest, DeleteMessageRequest,
            ListMessagesRequest, MarkReadRequest, ReactToMessageRequest, ListReactionsRequest,
            MarkAllReadRequest, GetUnreadSummaryRequest, GetUnreadCountRequest,
            MarkAsReadRequest, AddReactionRequest, RemoveReactionRequest,
        },
        response::{ApiResponse, PaginatedResponse, MessageResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::chat_messages_service::ChatMessagesService,
    state::AppState,
};

/// POST /protected/chat_messages/send
pub async fn send(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<SendMessageRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let message_id = service.send_message(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "消息发送成功",
        "id": message_id
    }))))
}

/// POST /api/v1/protected/message/edit
pub async fn edit(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<EditMessageRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.edit_message(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "消息编辑成功"
    }))))
}

/// POST /api/v1/protected/message/recall
pub async fn recall(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<RecallMessageRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.recall_message(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "消息撤回成功"
    }))))
}

/// POST /api/v1/protected/message/delete
pub async fn delete(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<DeleteMessageRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_message(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "消息删除成功"
    }))))
}

/// POST /api/v1/protected/message/list
pub async fn list(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<ListMessagesRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MessageResponse>>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.list_messages(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/message/mark_read
pub async fn mark_read(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<MarkAsReadRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.mark_as_read(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已标记为已读"
    }))))
}

/// POST /api/v1/protected/message/add_reaction
pub async fn add_reaction(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<AddReactionRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.add_reaction(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "反应添加成功"
    }))))
}

/// POST /api/v1/protected/message/remove_reaction
pub async fn remove_reaction(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<RemoveReactionRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.remove_reaction(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "反应已移除"
    }))))
}

/// POST /api/v1/protected/chat_messages/unread_summary
pub async fn unread_summary(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<GetUnreadSummaryRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let include_details = req_body.include_details.unwrap_or(false);
    let summary = service.get_unread_summary(user_id, include_details).await?;
    
    Ok(Json(ApiResponse::success(summary)))
}

/// POST /api/v1/protected/chat_messages/mark_all_read
pub async fn mark_all_read(
    State(state): State<AppState>,
    req: Request,
    Json(_req_body): Json<MarkAllReadRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.mark_all_read(user_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "所有消息已标记为已读"
    }))))
}

/// POST /api/v1/protected/chat_messages/unread_count
pub async fn unread_count(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<GetUnreadCountRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let room_id = uuid::Uuid::parse_str(&req_body.target_id)
        .map_err(|_| crate::error::AppError::BadRequest("无效的会话ID".to_string()))?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let unread_count = service.get_unread_count(user_id, room_id).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "target_id": req_body.target_id,
        "target_type": req_body.target_type,
        "unread_count": unread_count
    }))))
}

/// POST /api/v1/protected/chat_messages/reactions
pub async fn list_reactions(
    State(state): State<AppState>,
    req: Request,
    Json(_req_body): Json<ListReactionsRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let _user_id = extract_user_id(&req)?;
    
    // TODO: 实现reactions列表查询
    // 当前返回空列表作为占位
    Ok(Json(ApiResponse::success(serde_json::json!({
        "items": [],
        "has_more": false,
        "next_cursor": null
    }))))
}
