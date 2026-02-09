use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, AdminUpdateAnnouncementRequest,
            CreateAnnouncementRequest, DeleteAnnouncementRequest, PublicAnnouncementListRequest,
        },
        response::{ApiResponse, CreateAnnouncementData, PaginatedData, SystemAnnouncementItem},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

// ==================== 管理员公告管理 ====================

pub async fn admin_create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateAnnouncementRequest>,
) -> Result<Json<ApiResponse<CreateAnnouncementData>>> {
    let title = req.title.clone();
    let details = serde_json::json!({
        "severity": req.severity,
        "action_url": req.action_url,
        "send_push": req.send_push,
    });
    let response = state
        .announcement_service
        .create_announcement(user.user_id, req)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "announcement.create".to_string(),
            operator_id: user.user_id,
            target_type: "announcement".to_string(),
            target_id: response.announcement_id.clone(),
            target_name: Some(title),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
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

/// TODO: 需要在 AnnouncementService 中实现 update_announcement 方法
pub async fn admin_update(
    State(_state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminUpdateAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    Err(crate::error::AppError::InternalError(
        "公告更新功能尚未实现".to_string(),
    ))
}

pub async fn admin_delete(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DeleteAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let announcement_id = req.announcement_id.clone();
    state.announcement_service.delete_announcement(req).await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "announcement.delete".to_string(),
            operator_id: user.user_id,
            target_type: "announcement".to_string(),
            target_id: announcement_id,
            target_name: None,
            details: None,
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
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
