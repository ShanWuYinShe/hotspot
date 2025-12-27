use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{CreateReportRequest, HandleReportRequest, MyReportsRequest, ReportsRequest},
        response::{ApiResponse, CreateReportData, PaginatedData, ReportItem},
    },
    error::Result,
    state::AppState,
};

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateReportRequest>,
) -> Result<Json<ApiResponse<CreateReportData>>> {
    let result = state.report_service.create_report(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn me(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    let result = state.report_service.get_my_reports(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn get_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    let result = state.report_service.get_list(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}

pub async fn handle(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<HandleReportRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.report_service.handle(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
