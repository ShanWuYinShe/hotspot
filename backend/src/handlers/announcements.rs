use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, AdminUpdateAnnouncementRequest,
            CreateAnnouncementRequest, DeleteAnnouncementRequest, PublicAnnouncementListRequest,
        },
        response::{
            ApiResponse, CreateAnnouncementData, PaginatedData, SystemAnnouncementItem,
            SystemAnnouncementListData,
        },
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

pub async fn admin_update(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(req): Json<AdminUpdateAnnouncementRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.announcement_service.update_announcement(req).await?;
    Ok(Json(ApiResponse::success_empty()))
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
    Json(_req): Json<PublicAnnouncementListRequest>,
) -> Result<Json<ApiResponse<SystemAnnouncementListData>>> {
    // 从 ResponseDefaults 配置获取返回条数，默认 10
    let count = state
        .system_config_service
        .get_config_value("response_defaults.system_announcements_count")
        .await?
        .and_then(|s| s.parse::<i64>().ok())
        .unwrap_or(10);

    let items = state
        .announcement_service
        .list_public_announcements(count)
        .await?;
    Ok(Json(ApiResponse::success(SystemAnnouncementListData {
        items,
    })))
}
