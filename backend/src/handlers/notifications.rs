//! 通知中心 HTTP Handler

use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{MarkNotificationsReadRequest, NotificationListRequest},
        response::{ApiResponse, NotificationItem, PaginatedData, UnreadCountResult},
    },
    error::{AppError, Result},
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

/// 通知列表
pub async fn list(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<NotificationListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<NotificationItem>>>> {
    let data = state.notification_service.list(user_id, req).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 未读数量
pub async fn unread_count(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(_req): Json<serde_json::Value>,
) -> Result<Json<ApiResponse<UnreadCountResult>>> {
    let data = state.notification_service.unread_count(user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 标记已读
pub async fn mark_as_read(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<MarkNotificationsReadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if req.notification_ids.is_empty() {
        return Err(AppError::BadRequest("notification_ids 不能为空".to_string()));
    }
    state
        .notification_service
        .mark_as_read(user_id, &req.notification_ids)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 标记全部已读
pub async fn mark_all_read(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(_req): Json<serde_json::Value>,
) -> Result<Json<ApiResponse<()>>> {
    state.notification_service.mark_all_read(user_id).await?;
    Ok(Json(ApiResponse::success_empty()))
}
