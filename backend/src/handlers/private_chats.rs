use axum::{extract::State, Extension, Json};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreatePrivateChatRequest, DeletePrivateChatRequest, MyPrivateChatsRequest,
            PrivateChatOpenTheDoorRequest, UpdatePrivateChatSettingsRequest,
        },
        response::{
            ApiResponse, CreatePrivateChatData, PaginatedData, PaginationMeta,
            PrivateChatDetailData, PrivateChatListItem,
        },
    },
    error::{AppError, Result},
    state::AppState,
    utils::validation::Validator,
};

/// 3.5.1 创建私聊
pub async fn create(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<CreatePrivateChatRequest>,
) -> Result<Json<ApiResponse<CreatePrivateChatData>>> {
    let target_user_id = req.target_user_id;

    let chat = state
        .private_chat_service
        .create_or_get(current_user_id, target_user_id, req.initial_message)
        .await?;

    Ok(Json(ApiResponse::success(CreatePrivateChatData {
        id: chat.private_chat_id.to_string(),
    })))
}

/// 3.5.2 获取私聊列表
pub async fn me(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<MyPrivateChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateChatListItem>>>> {
    let limit = Validator::validate_limit(req.limit, 100)? as i64;
    let include_archived = req.include_archived.unwrap_or(false);
    let (items, has_more, next_cursor) = state
        .private_chat_service
        .get_list_with_details(current_user_id, limit, req.cursor, include_archived)
        .await?;

    Ok(Json(ApiResponse::success(PaginatedData {
        items,
        pagination: PaginationMeta {
            has_more,
            next_cursor,
        },
    })))
}

/// 获取私聊详情 - POST /api/v1/protected/private_chats/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<PrivateChatOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<PrivateChatDetailData>>> {
    let private_chat_id =
        Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

    let detail = state
        .private_chat_service
        .get_details(private_chat_id, current_user_id)
        .await?;

    Ok(Json(ApiResponse::success(detail)))
}

/// 3.5.4 更新私聊设置
pub async fn update_settings(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<UpdatePrivateChatSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let private_chat_id =
        Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

    state
        .private_chat_service
        .update_settings(
            private_chat_id,
            current_user_id,
            req.is_muted,
            req.is_pinned,
            req.is_archived,
        )
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 3.5.5 删除私聊
pub async fn delete(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<DeletePrivateChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let private_chat_id =
        Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

    state
        .private_chat_service
        .delete(private_chat_id, current_user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}
