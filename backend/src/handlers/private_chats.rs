use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            MyPrivateChatsRequest, PrivateChatOpenTheDoorRequest, UpdatePrivateChatSettingsRequest,
        },
        response::{ApiResponse, PaginatedData, PrivateChatListItem, PrivateChatOpenResult},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
    utils::validation::Validator,
};

/// 3.5.2 获取私聊列表
pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyPrivateChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateChatListItem>>>> {
    let pagination = req.pagination.as_ref();
    let limit = Validator::validate_limit(
        pagination.and_then(|p| p.limit),
        crate::constants::MAX_PAGE_LIMIT,
    )? as i64;
    let (items, _, next_cursor) = state
        .private_chat_service
        .get_list_with_details(
            user.user_id,
            limit,
            pagination.and_then(|p| p.cursor.clone()),
        )
        .await?;

    Ok(Json(ApiResponse::success(PaginatedData {
        items,
        next_cursor,
    })))
}

/// 获取私聊详情 - POST /api/v1/private_chats/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<PrivateChatOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<PrivateChatOpenResult>>> {
    let target_user_id = crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
    let detail = state
        .private_chat_service
        .get_details(target_user_id, user.user_id)
        .await?;

    Ok(Json(ApiResponse::success(detail)))
}

/// 3.5.4 更新私聊设置
pub async fn update_settings(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdatePrivateChatSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let target_user_id = crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
    state
        .private_chat_service
        .update_settings(target_user_id, user.user_id, req.is_muted, req.is_pinned)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}
