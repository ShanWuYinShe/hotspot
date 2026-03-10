use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminReportHandleRequest, AdminReportSearchRequest, CreateReportRequest,
            MyReportsRequest,
        },
        response::{AdminReportItem, ApiResponse, CreateReportData, PaginatedData, ReportItem},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

pub async fn create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateReportRequest>,
) -> Result<Json<ApiResponse<CreateReportData>>> {
    let result = state
        .report_service
        .create_report(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    let result = state
        .report_service
        .get_my_reports(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn get_list(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminReportSearchRequest>,
) -> Result<Json<ApiResponse<PaginatedData<AdminReportItem>>>> {
    let result = state.report_service.get_list(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn handle(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminReportHandleRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.report_service.handle(user.user_id, req.clone()).await?;
    let details = serde_json::json!({
        "action": req.action,
        "note": req.note,
        "ban_duration_days": req.ban_duration_days,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "report.handle".to_string(),
            operator_id: user.user_id,
            target_type: "report".to_string(),
            target_id: req.report_id,
            target_name: None,
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
