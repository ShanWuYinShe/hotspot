use axum::{extract::State, Json};
use serde_json::json;

use crate::{
    dto::{
        request::AdminTagSearchRequest,
        response::{AdminTagItem, ApiResponse, PaginatedData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

// ==================== Admin APIs ====================

/// 搜索标签（管理员）
pub async fn admin_search(
    State(state): State<AppState>,
    Json(req): Json<AdminTagSearchRequest>,
) -> Result<Json<ApiResponse<PaginatedData<AdminTagItem>>>> {
    let response = state.tag_service.admin_search_tags(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 删除/屏蔽标签（管理员）
pub async fn admin_delete(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<crate::dto::request::AdminDeleteTagRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let tag_name = req.name.clone();
    state.tag_service.admin_delete_tag(&req).await?;

    // 记录审计日志
    let audit_input = AuditLogInput {
        action: "tag.delete".to_string(),
        operator_id: user.user_id,
        target_type: "tag".to_string(),
        target_id: tag_name.clone(),
        target_name: Some(tag_name),
        details: Some(json!({ "name": req.name })),
        before_state: None,
        after_state: None,
        ip_address: None,
        user_agent: None,
    };
    if let Err(e) = state.admin_audit_service.record(audit_input).await {
        tracing::error!(error = %e, "Failed to record tag.delete audit log");
    }

    Ok(Json(ApiResponse::success_empty()))
}
