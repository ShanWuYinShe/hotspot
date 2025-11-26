use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            ListInboxItemsRequest, GetUnreadInboxSummaryRequest, MarkInboxItemsReadRequest,
            DeleteInboxItemsRequest, RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
            RemoveInboxBatchRequest,
        },
        response::{
            ApiResponse, PaginatedData, InboxItemData, UnreadInboxSummaryData,
            MarkInboxReadData, BatchResultData,
        },
    },
    error::AppError,
    services::inbox_service::InboxService,
    state::AppState,
};

/// POST /api/v1/protected/inbox/get_list
pub async fn get_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListInboxItemsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<InboxItemData>>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    let result = inbox_service.get_list(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// POST /api/v1/protected/inbox/get_unread_count
pub async fn get_unread_count(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(_req): Json<GetUnreadInboxSummaryRequest>,
) -> Result<Json<ApiResponse<UnreadInboxSummaryData>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    let result = inbox_service.get_unread_count(user_id).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// POST /api/v1/protected/inbox/mark_all_as_read
pub async fn mark_all_as_read(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MarkInboxItemsReadRequest>,
) -> Result<Json<ApiResponse<MarkInboxReadData>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    let result = inbox_service.mark_all_as_read(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// POST /api/v1/protected/inbox/remove
pub async fn remove(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteInboxItemsRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    inbox_service.remove(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/inbox/remove_batch
pub async fn remove_batch(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RemoveInboxBatchRequest>,
) -> Result<Json<ApiResponse<BatchResultData>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    let result = inbox_service.remove_batch(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

/// POST /api/v1/protected/inbox/register_device_token
pub async fn register_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<RegisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    inbox_service.register_device_token(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/inbox/unregister_device_token
pub async fn unregister_device_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnregisterDeviceTokenRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    let inbox_service = InboxService::new(state.db.clone());
    inbox_service.unregister_device_token(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
