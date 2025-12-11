use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateReportRequest, GetMyReportsRequest, GetReportsRequest, HandleReportRequest,
        },
        response::{ApiResponse, PaginatedData, ReportItem, CreateReportData},
    },
    error::{AppError, Result},
    services::report_service::ReportService,
    state::AppState,
};

/// 检查管理员权限
async fn check_admin(state: &AppState, user_id: i64) -> Result<()> {
    let is_admin = sqlx::query_scalar::<_, bool>(
        r#"
        SELECT is_admin
        FROM users
        WHERE user_id = $1
        "#,
    )
    .bind(user_id)
    .fetch_optional(&state.db)
    .await?
    .unwrap_or(false);

    if !is_admin {
        return Err(AppError::Forbidden("需要管理员权限".to_string()));
    }

    Ok(())
}


pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateReportRequest>,
) -> Result<Json<ApiResponse<CreateReportData>>> {
    let service = ReportService::new(state.db.clone());
    let result = service.create_report(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}


pub async fn get_my_reports(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetMyReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    let service = ReportService::new(state.db.clone());
    let result = service.get_my_reports(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}


pub async fn get_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetReportsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ReportItem>>>> {
    check_admin(&state, user_id).await?;
    let service = ReportService::new(state.db.clone());
    let result = service.get_list(user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}


pub async fn handle(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<HandleReportRequest>,
) -> Result<Json<ApiResponse<()>>> {
    check_admin(&state, user_id).await?;
    let service = ReportService::new(state.db.clone());
    service.handle(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
