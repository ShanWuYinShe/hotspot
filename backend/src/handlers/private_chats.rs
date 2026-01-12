use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            DeletePrivateChatRequest, MyPrivateChatsRequest,
            PrivateChatOpenTheDoorRequest, UpdatePrivateChatSettingsRequest,
        },
        response::{
            ApiResponse, PaginatedData, PaginationMeta,
            PrivateChatDetailData, PrivateChatListItem,
        },
    },
    error::Result,
    state::AppState,
    utils::validation::Validator,
};

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
    let detail = state
        .private_chat_service
        .get_details(req.target_user_id, current_user_id)
        .await?;

    Ok(Json(ApiResponse::success(detail)))
}

/// 3.5.4 更新私聊设置
pub async fn update_settings(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<UpdatePrivateChatSettingsRequest>,
) -> Result<Json<ApiResponse<PrivateChatDetailData>>> {
    state
        .private_chat_service
        .update_settings(
            req.target_user_id,
            current_user_id,
            req.is_muted,
            req.is_pinned,
            req.is_archived,
        )
        .await?;

    let detail = state
        .private_chat_service
        .get_details(req.target_user_id, current_user_id)
        .await?;

    Ok(Json(ApiResponse::success(detail)))
}

/// 3.5.5 删除私聊
pub async fn delete(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<DeletePrivateChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .private_chat_service
        .delete(req.target_user_id, current_user_id)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}
