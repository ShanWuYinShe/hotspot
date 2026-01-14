use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, CreateAnnouncementRequest, DeleteAnnouncementRequest,
            PublicAnnouncementListRequest,
        },
        response::{ApiResponse, CreateAnnouncementData, PaginatedData, SystemAnnouncementItem},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

// ==================== 管理员公告管理 ====================

pub async fn admin_create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateAnnouncementRequest>,
) -> Result<Json<ApiResponse<CreateAnnouncementData>>> {
    let response = state
        .announcement_service
        .create_announcement(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn admin_list(
    State(state): State<AppState>,
    Json(req): Json<AdminAnnouncementListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<SystemAnnouncementItem>>>> {
    let response = state
        .announcement_service
        .list_admin_announcements(req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn admin_delete(
    State(state): State<AppState>,
    Json(req): Json<DeleteAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.announcement_service.delete_announcement(req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 公开公告列表 ====================

pub async fn public_list(
    State(state): State<AppState>,
    Json(req): Json<PublicAnnouncementListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<SystemAnnouncementItem>>>> {
    let response = state
        .announcement_service
        .list_public_announcements(req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
