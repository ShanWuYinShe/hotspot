use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            StartChatRequest, ListChatsRequest, GetChatSettingsRequest,
            UpdateChatSettingsRequest, DeleteChatRequest, UpgradeToGroupRequest,
        },
        response::{ApiResponse, PaginatedResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::chat_messages_service::ChatMessagesService,
    state::AppState,
};

/// POST /api/v1/protected/chat/start - 发起私聊
pub async fn start(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<StartChatRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let chat_id = service.start_chat(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "私聊创建成功",
        "chat_id": chat_id
    }))))
}

/// POST /api/v1/protected/chat/list - 获取私聊列表
pub async fn list(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<ListChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<serde_json::Value>>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.list_chats(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/chat/settings - 获取私聊设置
pub async fn settings(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<GetChatSettingsRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let settings = service.get_chat_settings(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(settings)))
}

/// POST /api/v1/protected/chat/update_settings - 更新私聊设置
pub async fn update_settings(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UpdateChatSettingsRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_chat_settings(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "私聊设置更新成功"
    }))))
}

/// POST /api/v1/protected/chat/delete - 删除私聊
pub async fn delete(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<DeleteChatRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_chat(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "私聊删除成功"
    }))))
}

/// POST /api/v1/protected/chat/upgrade_to_group - 升级为群组
pub async fn upgrade_to_group(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UpgradeToGroupRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = ChatMessagesService::new(&state.db, &state.redis, state.event_bus.clone());
    let group = service.upgrade_to_group(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "升级为群组成功",
        "group": group
    }))))
}





