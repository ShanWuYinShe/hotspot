use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            AdminGetGlobalSettingsRequest, AdminUpdateResponseDefaultsRequest,
            FetchGlobalSettingsRequest, UpdateContactRequest, UpdateDefaultsRequest,
            UpdateFeaturesRequest, UpdateLimitsRequest, UpdateRateLimitsRequest,
            UpdateUploadConfigRequest,
        },
        response::{
            AdminGlobalSettingsData, ApiResponse, ClientConfig, GlobalSettings,
            UpdateContactConfigData, UpdateDefaultsConfigData, UpdateFeaturesConfigData,
            UpdateLimitsConfigData, UpdateRateLimitsConfigData, UpdateUploadConfigData,
        },
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

// ==================== 客户端拉取配置 ====================

pub async fn get_client_config(
    State(state): State<AppState>,
    Json(req): Json<FetchGlobalSettingsRequest>,
) -> Result<Json<ApiResponse<ClientConfig>>> {
    let data = state.system_config_service.get_client_config(req).await?;
    Ok(Json(ApiResponse::success(data)))
}

// ==================== 管理员配置管理 (Admin) ====================

pub async fn admin_get_global_settings(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminGetGlobalSettingsRequest>,
) -> Result<Json<ApiResponse<AdminGlobalSettingsData>>> {
    let data = state
        .system_config_service
        .get_admin_global_settings()
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn admin_update_upload_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateUploadConfigRequest>,
) -> Result<Json<ApiResponse<UpdateUploadConfigData>>> {
    let details = serde_json::json!({
        "section": "upload",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_upload_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "upload".to_string(),
            target_name: Some("upload".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateUploadConfigData {
        upload: settings.upload,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

pub async fn admin_update_defaults_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateDefaultsRequest>,
) -> Result<Json<ApiResponse<UpdateDefaultsConfigData>>> {
    let details = serde_json::json!({
        "section": "defaults",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_defaults_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "defaults".to_string(),
            target_name: Some("defaults".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateDefaultsConfigData {
        defaults: settings.defaults,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

pub async fn admin_update_features_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateFeaturesRequest>,
) -> Result<Json<ApiResponse<UpdateFeaturesConfigData>>> {
    let details = serde_json::json!({
        "section": "features",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_features_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "features".to_string(),
            target_name: Some("features".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateFeaturesConfigData {
        features: settings.features,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

pub async fn admin_update_contact_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateContactRequest>,
) -> Result<Json<ApiResponse<UpdateContactConfigData>>> {
    let details = serde_json::json!({
        "section": "contact",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_contact_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "contact".to_string(),
            target_name: Some("contact".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateContactConfigData {
        contact: settings.contact,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

pub async fn admin_update_limits_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateLimitsRequest>,
) -> Result<Json<ApiResponse<UpdateLimitsConfigData>>> {
    let details = serde_json::json!({
        "section": "limits",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_limits_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "limits".to_string(),
            target_name: Some("limits".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateLimitsConfigData {
        limits: settings.limits,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

pub async fn admin_update_rate_limits_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateRateLimitsRequest>,
) -> Result<Json<ApiResponse<UpdateRateLimitsConfigData>>> {
    let details = serde_json::json!({
        "section": "rate_limits",
        "payload": serde_json::to_value(&req).unwrap_or_else(|_| serde_json::json!({})),
    });
    let settings = state
        .system_config_service
        .update_rate_limits_config(req, user.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "config.update".to_string(),
            operator_id: user.user_id,
            target_type: "config".to_string(),
            target_id: "rate_limits".to_string(),
            target_name: Some("rate_limits".to_string()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success(UpdateRateLimitsConfigData {
        rate_limits: settings.rate_limits,
        updated_at: chrono::Utc::now().to_rfc3339(),
    })))
}

/// 更新响应默认值配置（管理员）
pub async fn admin_update_response_defaults_config(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminUpdateResponseDefaultsRequest>,
) -> Result<Json<ApiResponse<GlobalSettings>>> {
    let settings = state
        .system_config_service
        .update_response_defaults_config(req, user.user_id)
        .await?;
    Ok(Json(ApiResponse::success(settings)))
}
