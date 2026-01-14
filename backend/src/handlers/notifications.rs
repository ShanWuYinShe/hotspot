use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            GetNotificationsRequest, GetUnreadNotificationCountRequest,
            MarkNotificationsReadRequest,
        },
        response::{ApiResponse, PaginatedData, UnreadNotificationCountData},
    },
    error::AppError,
    middleware::auth::AuthenticatedUser, // Fixed Import
    state::AppState,
};

/// 获取通知列表
pub async fn list_notifications(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<GetNotificationsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<crate::dto::response::NotificationItem>>>, AppError> {
    // Use service from state
    let data = state
        .notification_service
        .list_notifications(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 获取未读计数
pub async fn get_unread_count(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    _req: Json<GetUnreadNotificationCountRequest>,
) -> Result<Json<ApiResponse<UnreadNotificationCountData>>, AppError> {
    // Use service from state
    let data = state
        .notification_service
        .get_unread_count(user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 标记已读
pub async fn mark_read(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MarkNotificationsReadRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    // Use service from state
    state
        .notification_service
        .mark_read(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(())))
}
