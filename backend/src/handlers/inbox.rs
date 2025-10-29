use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            DeleteInboxItemsRequest, GetInboxRequest, MarkInboxReadRequest,
            RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
        },
        response::{ApiResponse, InboxItemResponse, PaginatedResponse},
    },
    error::AppError,
    services::inbox_service::InboxService,
    state::AppState,
};

pub struct InboxHandler;

impl InboxHandler {
    /// 获取收件箱列表
    pub async fn get_inbox(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetInboxRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<InboxItemResponse>>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        let result = inbox_service.get_inbox(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 标记已读
    pub async fn mark_as_read(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<MarkInboxReadRequest>,
    ) -> Result<Json<ApiResponse<u64>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        let affected = inbox_service.mark_as_read(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(affected)))
    }

    /// 删除收件箱项
    pub async fn delete_items(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<DeleteInboxItemsRequest>,
    ) -> Result<Json<ApiResponse<u64>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        let affected = inbox_service.delete_items(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(affected)))
    }

    /// 获取未读数量
    pub async fn get_unread_count(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
    ) -> Result<Json<ApiResponse<i64>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        let count = inbox_service.get_unread_count(user_id).await?;
        Ok(Json(ApiResponse::success_data(count)))
    }

    /// 注册设备Token
    pub async fn register_device_token(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<RegisterDeviceTokenRequest>,
    ) -> Result<Json<ApiResponse<()>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        inbox_service.register_device_token(user_id, req).await?;
        Ok(Json(ApiResponse::success("设备Token注册成功")))
    }

    /// 注销设备Token
    pub async fn unregister_device_token(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<UnregisterDeviceTokenRequest>,
    ) -> Result<Json<ApiResponse<()>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        inbox_service
            .unregister_device_token(user_id, req)
            .await?;
        Ok(Json(ApiResponse::success("设备Token注销成功")))
    }
}

