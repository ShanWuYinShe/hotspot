use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            DeleteInboxItemsRequest, GetInboxRequest, MarkInboxReadRequest,
            UpdateDeviceTokenRequest,
        },
        response::{ApiResponse, InboxItemResponse, PaginatedResponse, UpdateDeviceTokenResponse},
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

    /// 更新设备Token（注册或注销）
    pub async fn update_device_token(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<UpdateDeviceTokenRequest>,
    ) -> Result<Json<ApiResponse<UpdateDeviceTokenResponse>>, AppError> {
        let inbox_service = InboxService::new(state.db.clone());
        inbox_service.update_device_token(user_id, req.clone()).await?;
        
        let message = match req.action.as_str() {
            "register" => "设备Token注册成功",
            "unregister" => "设备Token注销成功",
            _ => "操作成功",
        };
        
        let response = UpdateDeviceTokenResponse {
            success: true,
            message: message.to_string(),
        };
        
        Ok(Json(ApiResponse::success_data(response)))
    }
}

