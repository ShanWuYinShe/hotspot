//! 分享处理器
//!
//! 处理分享相关的 HTTP 请求

use crate::dto::request::{
    ApplyShareRequest, CreateShareRequest, GetShareRequest, MySharesRequest, RevokeShareRequest,
};
use crate::dto::response::{ApiResponse, PaginatedData, Share};
use crate::error::AppError;
use crate::middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser};
use crate::services::share_service::ShareService;
use crate::state::AppState;
use axum::{extract::State, Json};

/// 创建分享
///
/// POST /api/v1/share/create
pub async fn create_share(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<CreateShareRequest>,
) -> Result<Json<ApiResponse<Share>>, AppError> {
    let share = ShareService::create_share(&state.db, user_id, req).await?;

    Ok(Json(ApiResponse::success(share)))
}

/// 获取分享详情
///
/// POST /api/v1/share/get
pub async fn get_share(
    State(state): State<AppState>,
    OptionalAuthenticatedUser { .. }: OptionalAuthenticatedUser,
    Json(req): Json<GetShareRequest>,
) -> Result<Json<ApiResponse<Share>>, AppError> {
    let share = ShareService::get_share(&state.db, req).await?;

    Ok(Json(ApiResponse::success(share)))
}

/// 应用分享
///
/// POST /api/v1/share/apply
pub async fn apply_share(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<ApplyShareRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    ShareService::apply_share(&state.db, user_id, req).await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 撤销分享
///
/// POST /api/v1/share/revoke
/// TODO: 需要在 ShareService 中实现 revoke_share 方法
pub async fn revoke_share(
    State(_state): State<AppState>,
    AuthenticatedUser { .. }: AuthenticatedUser,
    Json(_req): Json<RevokeShareRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    Err(AppError::InternalError(
        "分享撤销功能尚未实现".to_string(),
    ))
}

/// 我的分享列表
///
/// POST /api/v1/share/me/list
/// TODO: 需要在 ShareService 中实现 my_shares 方法
pub async fn my_shares(
    State(_state): State<AppState>,
    AuthenticatedUser { .. }: AuthenticatedUser,
    Json(_req): Json<MySharesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<Share>>>, AppError> {
    Err(AppError::InternalError(
        "我的分享列表功能尚未实现".to_string(),
    ))
}
