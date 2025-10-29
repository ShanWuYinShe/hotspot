use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{GetConversationSettingsRequest, UpdateConversationSettingsRequest},
        response::{ApiResponse, ConversationSettingsResponse},
    },
    error::AppError,
    services::conversation_settings_service::ConversationSettingsService,
    state::AppState,
};

pub struct ConversationSettingsHandler;

impl ConversationSettingsHandler {
    /// 获取会话设置
    pub async fn get_settings(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetConversationSettingsRequest>,
    ) -> Result<Json<ApiResponse<ConversationSettingsResponse>>, AppError> {
        let service = ConversationSettingsService::new(state.db.clone());
        let result = service.get_settings(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 更新会话设置
    pub async fn update_settings(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<UpdateConversationSettingsRequest>,
    ) -> Result<Json<ApiResponse<ConversationSettingsResponse>>, AppError> {
        let service = ConversationSettingsService::new(state.db.clone());
        let result = service.update_settings(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }
}

