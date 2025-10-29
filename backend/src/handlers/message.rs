use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            SendMessageRequest, EditMessageRequest, RecallMessageRequest, DeleteMessageRequest,
            ListMessagesRequest, MarkAsReadRequest, AddReactionRequest, RemoveReactionRequest,
        },
        response::{ApiResponse, PaginatedResponse, MessageResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::message_service::MessageService,
    state::AppState,
};

/// POST /api/v1/protected/message/send
pub async fn send(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<SendMessageRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
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
    
    let service = MessageService::new(&state.db, &state.redis);
    service.remove_reaction(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "反应已移除"
    }))))
}

/// POST /api/v1/protected/message/unread_summary
pub async fn unread_summary(
    State(state): State<AppState>,
    req: Request,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = MessageService::new(&state.db, &state.redis);
    let summary = service.get_unread_summary(user_id).await?;
    
    Ok(Json(ApiResponse::success(summary)))
}
