use axum::{extract::State, Json};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AdminRbacAdminsListRequest, AdminRbacGrantAdminRequest, AdminRbacMeRequest,
            AdminRbacPermissionsRequest, AdminRbacRevokeAdminRequest, AdminRbacRoleCreateRequest,
            AdminRbacRoleDeleteRequest, AdminRbacRoleSetPermissionsRequest,
            AdminRbacRoleUpdateRequest, AdminRbacRolesListRequest,
        },
        response::{
            AdminAdminItem, AdminAdminsData, AdminMeData, AdminPermissionItem,
            AdminPermissionsData, AdminRoleBasic, AdminRoleItem, AdminRolesData, ApiResponse,
            UserInfo,
        },
    },
    error::{AppError, Result},
    middleware::auth::AuthenticatedUser,
    services::admin_audit_service::AuditLogInput,
    state::AppState,
};

pub async fn me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(_req): Json<AdminRbacMeRequest>,
) -> Result<Json<ApiResponse<AdminMeData>>> {
    let (_is_super, permissions, roles) = state.admin_rbac_service.get_me(user.user_id).await?;

    Ok(Json(ApiResponse::success(AdminMeData {
        permissions,
        roles: roles
            .into_iter()
            .map(|r| AdminRoleBasic {
                role_id: r.role_id.to_string(),
                name: r.name,
            })
            .collect(),
    })))
}

pub async fn list_permissions(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminRbacPermissionsRequest>,
) -> Result<Json<ApiResponse<AdminPermissionsData>>> {
    let items = state
        .admin_rbac_service
        .list_permissions()
        .await?
        .into_iter()
        .map(|(key, description)| AdminPermissionItem { key, description })
        .collect();

    Ok(Json(ApiResponse::success(AdminPermissionsData { items })))
}

pub async fn list_roles(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminRbacRolesListRequest>,
) -> Result<Json<ApiResponse<AdminRolesData>>> {
    let items = state
        .admin_rbac_service
        .list_roles()
        .await?
        .into_iter()
        .map(|(role, permissions)| AdminRoleItem {
            role_id: role.role_id.to_string(),
            name: role.name,
            description: role.description,
            permissions,
        })
        .collect();

    Ok(Json(ApiResponse::success(AdminRolesData { items })))
}

pub async fn create_role(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminRbacRoleCreateRequest>,
) -> Result<Json<ApiResponse<AdminRoleItem>>> {
    let role_id = state
        .admin_rbac_service
        .create_role(&req.name, req.description.as_deref())
        .await?;
    let details = serde_json::json!({
        "description": req.description,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "role.create".to_string(),
            operator_id: user.user_id,
            target_type: "role".to_string(),
            target_id: role_id.to_string(),
            target_name: Some(req.name.clone()),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;

    Ok(Json(ApiResponse::success(AdminRoleItem {
        role_id: role_id.to_string(),
        name: req.name,
        description: req.description,
        permissions: vec![],
    })))
}

pub async fn update_role(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminRbacRoleUpdateRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let role_id = Uuid::parse_str(&req.role_id)
        .map_err(|_| AppError::BadRequest("无效的角色ID".to_string()))?;
    let desc = if req.clear_description.unwrap_or(false) {
        Some(None)
    } else {
        req.description.as_deref().map(Some)
    };
    state
        .admin_rbac_service
        .update_role(role_id, req.name.as_deref(), desc)
        .await?;
    let details = serde_json::json!({
        "name": req.name,
        "description": req.description,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "role.update".to_string(),
            operator_id: user.user_id,
            target_type: "role".to_string(),
            target_id: role_id.to_string(),
            target_name: req.name.clone(),
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn delete_role(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminRbacRoleDeleteRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let role_id = Uuid::parse_str(&req.role_id)
        .map_err(|_| AppError::BadRequest("无效的角色ID".to_string()))?;
    state.admin_rbac_service.delete_role(role_id).await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "role.delete".to_string(),
            operator_id: user.user_id,
            target_type: "role".to_string(),
            target_id: role_id.to_string(),
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

pub async fn set_role_permissions(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<AdminRbacRoleSetPermissionsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let role_id = Uuid::parse_str(&req.role_id)
        .map_err(|_| AppError::BadRequest("无效的角色ID".to_string()))?;
    state
        .admin_rbac_service
        .set_role_permissions(role_id, &req.permission_keys)
        .await?;
    let details = serde_json::json!({
        "permission_keys": req.permission_keys,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "role.update".to_string(),
            operator_id: user.user_id,
            target_type: "role".to_string(),
            target_id: role_id.to_string(),
            target_name: None,
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn list_admins(
    State(state): State<AppState>,
    _user: AuthenticatedUser,
    Json(_req): Json<AdminRbacAdminsListRequest>,
) -> Result<Json<ApiResponse<AdminAdminsData>>> {
    let rows = state.admin_rbac_service.list_admins().await?;
    let items = rows
        .into_iter()
        .map(|row| {
            let permissions = if row.is_super {
                vec!["*".to_string()]
            } else {
                vec![]
            };

            AdminAdminItem {
                user: UserInfo {
                    user_id: row.user.user_id,
                    nickname: row.user.nickname,
                    avatar_url: row.user.avatar_url,
                    user_type: row.user.user_type,
                },
                permissions,
                roles: row
                    .roles
                    .into_iter()
                    .map(|r| AdminRoleBasic {
                        role_id: r.role_id.to_string(),
                        name: r.name,
                    })
                    .collect(),
            }
        })
        .collect();

    Ok(Json(ApiResponse::success(AdminAdminsData { items })))
}

pub async fn grant_admin(
    State(state): State<AppState>,
    operator: AuthenticatedUser,
    Json(req): Json<AdminRbacGrantAdminRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let role_ids: Vec<Uuid> = req
        .role_ids
        .iter()
        .map(|id| Uuid::parse_str(id).map_err(|_| AppError::BadRequest("无效的角色ID".to_string())))
        .collect::<Result<Vec<_>>>()?;

    state
        .admin_rbac_service
        .grant_admin(
            operator.user_id,
            req.user_id,
            req.grant_all_permissions.unwrap_or(false),
            &role_ids,
        )
        .await?;
    let details = serde_json::json!({
        "grant_all_permissions": req.grant_all_permissions,
        "role_ids": req.role_ids,
    });
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "permission.grant".to_string(),
            operator_id: operator.user_id,
            target_type: "user".to_string(),
            target_id: req.user_id.to_string(),
            target_name: None,
            details: Some(details),
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        })
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn revoke_admin(
    State(state): State<AppState>,
    operator: AuthenticatedUser,
    Json(req): Json<AdminRbacRevokeAdminRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .admin_rbac_service
        .revoke_admin(operator.user_id, req.user_id)
        .await?;
    state
        .admin_audit_service
        .record(AuditLogInput {
            action: "permission.revoke".to_string(),
            operator_id: operator.user_id,
            target_type: "user".to_string(),
            target_id: req.user_id.to_string(),
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
