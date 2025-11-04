use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateReportRequest, GetReportHistoryRequest, GetReportsRequest, HandleReportRequest,
        },
        response::{ApiResponse, PaginatedResponse, ReportResponse},
    },
    error::AppError,
    services::report_service::ReportService,
    state::AppState,
};

pub struct ReportHandler;

impl ReportHandler {
    /// 提交举报
    pub async fn create_report(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<CreateReportRequest>,
    ) -> Result<Json<ApiResponse<ReportResponse>>, AppError> {
        let report_service = ReportService::new(state.db.clone());
        let result = report_service.create_report(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 获取举报历史
    pub async fn get_report_history(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetReportHistoryRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<ReportResponse>>>, AppError> {
        let report_service = ReportService::new(state.db.clone());
        let result = report_service.get_report_history(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 获取所有举报（管理员）
    pub async fn get_all_reports(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<GetReportsRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<ReportResponse>>>, AppError> {
        let report_service = ReportService::new(state.db.clone());
        let result = report_service.get_all_reports(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }

    /// 审核举报（管理员）
    pub async fn handle_report(
        State(state): State<AppState>,
        Extension(user_id): Extension<i64>,
        Json(req): Json<HandleReportRequest>,
    ) -> Result<Json<ApiResponse<ReportResponse>>, AppError> {
        let report_service = ReportService::new(state.db.clone());
        let result = report_service.handle_report(user_id, req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }
}

