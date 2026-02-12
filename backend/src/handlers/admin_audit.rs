use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{AuditLogDetailRequest, AuditLogListRequest},
        response::{ApiResponse, AuditLogDetail, AuditLogItem, PaginatedData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

pub async fn list(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(req): Json<AuditLogListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<AuditLogItem>>>> {
    let response = state.admin_audit_service.list_logs(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn detail(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(req): Json<AuditLogDetailRequest>,
) -> Result<Json<ApiResponse<AuditLogDetail>>> {
    let response = state.admin_audit_service.get_detail(req).await?;
    Ok(Json(ApiResponse::success(response)))
}
