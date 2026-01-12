use axum::{
    extract::{Request, State},
    middleware::Next,
    response::Response,
};

use crate::{error::AppError, middleware::auth::extract_user_id, state::AppState};

fn required_permission(path: &str) -> Option<&'static str> {
    // RBAC 自检：只要是管理员即可访问（用于前端判断可见性/权限）
    if path == "/api/v1/admin/rbac/me" {
        return None;
    }

    // RBAC 管理：默认要求 rbac.manage（超级管理员自动放行）
    if path.starts_with("/api/v1/admin/rbac") {
        return Some("admin.rbac.manage");
    }

    // 举报管理
    if path.starts_with("/api/v1/admin/reports") {
        if path.ends_with("/handle") {
            return Some("admin.reports.write");
        }
        return Some("admin.reports.read");
    }

    // 系统配置管理
    if path.starts_with("/api/v1/admin/config") {
        if path.ends_with("/create") || path.ends_with("/update") || path.ends_with("/delete") {
            return Some("admin.config.write");
        }
        return Some("admin.config.read");
    }

    // 未来新增 admin 路由：默认只要求 “是管理员”
    None
}

/// 管理员权限验证中间件
/// 必须在 auth_middleware 之后使用
/// 基于 RBAC（admin_users + admin_user_roles + admin_role_permissions）
pub async fn admin_auth_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 1. 提取用户ID（由 auth_middleware 注入）
    let user_id = extract_user_id(&req)?;

    // 2. 必须是管理员
    if !state.admin_rbac_service.is_admin(user_id).await? {
        return Err(AppError::NotAdmin);
    }

    // 3. 按路由校验权限（超级管理员自动放行）
    if let Some(permission) = required_permission(req.uri().path()) {
        state.admin_rbac_service.require_permission(user_id, permission).await?;
    }

    // 4. 继续处理请求
    Ok(next.run(req).await)
}
