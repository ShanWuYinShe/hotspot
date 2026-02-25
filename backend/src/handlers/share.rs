//! 分享处理器
//!
//! 处理分享相关的 HTTP 请求

use crate::dto::request::{
    CreateShareRequest, GetShareRequest, RevokeShareRequest, ShareListRequest,
};
use crate::dto::response::{ApiResponse, PaginatedData, ShareCreated, ShareDetail, ShareListItem};
use crate::error::AppError;
use crate::middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser};
use crate::state::AppState;
use axum::{
    extract::{FromRequestParts, State},
    http::request::Parts,
    Json,
};

#[derive(Debug, Clone)]
pub struct ClientIp(pub String);

impl<S> FromRequestParts<S> for ClientIp
where
    S: Send + Sync,
{
    type Rejection = AppError;

    fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> impl std::future::Future<Output = Result<Self, Self::Rejection>> + Send {
        let client_ip = crate::utils::client_ip::get_client_ip_from_headers(&parts.headers);

        async move { Ok(ClientIp(client_ip)) }
    }
}

/// 创建分享
///
/// POST /api/v1/share/create
pub async fn create_share(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<CreateShareRequest>,
) -> Result<Json<ApiResponse<ShareCreated>>, AppError> {
    let data = state.share_service.create_share(user_id, req).await?;

    Ok(Json(ApiResponse::success(data)))
}

/// 获取分享详情
///
/// POST /api/v1/share/get
pub async fn get_share(
    State(state): State<AppState>,
    client_ip: ClientIp,
    OptionalAuthenticatedUser { user_id }: OptionalAuthenticatedUser,
    Json(req): Json<GetShareRequest>,
) -> Result<Json<ApiResponse<ShareDetail>>, AppError> {
    let ClientIp(client_ip) = client_ip;
    let data = state
        .share_service
        .get_share(req, user_id, &client_ip)
        .await?;

    Ok(Json(ApiResponse::success(data)))
}

/// 撤销分享
///
/// POST /api/v1/share/revoke
pub async fn revoke_share(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<RevokeShareRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    state.share_service.revoke_share(user_id, req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// 获取当前用户创建的分享列表
pub async fn list_my_shares(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<ShareListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<ShareListItem>>>, AppError> {
    let data = state.share_service.list_my_shares(user_id, req).await?;
    Ok(Json(ApiResponse::success(data)))
}
