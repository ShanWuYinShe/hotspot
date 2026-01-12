use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, CreateAnnouncementRequest, DeleteAnnouncementRequest,
            PublicAnnouncementListRequest,
        },
        response::{
            ApiResponse, CreateAnnouncementData, PaginatedData, SystemAnnouncementItem,
        },
    },
    error::Result,
    state::AppState,
};

// ==================== 管理员公告管理 ====================

pub async fn admin_create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateAnnouncementRequest>,
) -> Result<Json<ApiResponse<CreateAnnouncementData>>> {
    let response = state
        .announcement_service
        .create_announcement(user_id, req)
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
