use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AuditLogDetailRequest, AuditLogExportRequest, AuditLogExportStatusRequest,
            AuditLogListRequest,
        },
        response::{ApiResponse, AuditLogDetail, AuditLogExportResult, AuditLogItem, PaginatedData},
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

pub async fn export(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AuditLogExportRequest>,
) -> Result<Json<ApiResponse<AuditLogExportResult>>> {
    let response = state.admin_audit_service.export_logs(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn export_status(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(req): Json<AuditLogExportStatusRequest>,
) -> Result<Json<ApiResponse<AuditLogExportResult>>> {
    let response = state.admin_audit_service.export_status(req).await?;
    Ok(Json(ApiResponse::success(response)))
}
