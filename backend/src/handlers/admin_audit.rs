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
    user: AuthenticatedUser,
    Json(req): Json<AuditLogListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<AuditLogItem>>>> {
    // 检查用户是否具备 admin.audit.read_ip 权限
    let has_read_ip_permission = state
        .admin_rbac_service
        .has_permission(user.user_id, "admin.audit.read_ip")
        .await
        .unwrap_or(false);

    let response = state
        .admin_audit_service
        .list_logs(req, has_read_ip_permission)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn detail(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AuditLogDetailRequest>,
) -> Result<Json<ApiResponse<AuditLogDetail>>> {
    // 检查用户是否具备 admin.audit.read_ip 权限
    let has_read_ip_permission = state
        .admin_rbac_service
        .has_permission(user.user_id, "admin.audit.read_ip")
        .await
        .unwrap_or(false);

    let response = state
        .admin_audit_service
        .get_detail(req, has_read_ip_permission)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
