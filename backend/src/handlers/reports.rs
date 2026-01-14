use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{CreateReportRequest, HandleReportRequest, MyReportsRequest, ReportsRequest},
        response::{ApiResponse, CreateReportData, PaginatedData, ReportItem},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
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
    Json(req): Json<ReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    let result = state.report_service.get_list(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn handle(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<HandleReportRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.report_service.handle(user.user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
