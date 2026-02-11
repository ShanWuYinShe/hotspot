use axum::{
    extract::{Request, State},
    middleware::Next,
    response::Response,
};

use crate::{error::AppError, state::AppState};

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

    // 配置管理（统一路径 admin/app_config/*）
    if path.starts_with("/api/v1/admin/app_config") {
        if path.contains("/update_") {
            return Some("admin.app_config.write");
        }
        return Some("admin.app_config.read");
    }

    // 用户管理
    if path.starts_with("/api/v1/admin/users") {
        if path.ends_with("/ban") || path.ends_with("/unban") {
            return Some("admin.users.write");
        }
        return Some("admin.users.read");
    }

    // 群组管理
    if path.starts_with("/api/v1/admin/groups") {
        if path.ends_with("/dismiss") {
            return Some("admin.groups.write");
        }
        return Some("admin.groups.read");
    }

    // 标签管理
    if path.starts_with("/api/v1/admin/tags") {
        return Some("admin.tags.write");
    }

    // 公告管理
    if path.starts_with("/api/v1/admin/announcements") {
        if path.ends_with("/create") || path.ends_with("/delete") {
            return Some("admin.announcements.write");
        }
        return Some("admin.announcements.read");
    }

    // 审计日志
    if path.starts_with("/api/v1/admin/audit") {
        if path.contains("/export") {
            return Some("admin.audit.export");
        }
        return Some("admin.audit.read");
    }

    // 未来新增 admin 路由：默认只要求 "是管理员"
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
    // 1. 提取用户ID（由 auth_middleware 注入到 request extensions）
    let user_id = req
        .extensions()
        .get::<i64>()
        .copied()
        .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))?;

    // 2. 必须是管理员
    if !state.admin_rbac_service.is_admin(user_id).await? {
        return Err(AppError::NotAdmin);
    }

    // 3. 按路由校验权限（超级管理员自动放行）
    if let Some(permission) = required_permission(req.uri().path()) {
        state
            .admin_rbac_service
            .require_permission(user_id, permission)
            .await?;
    }

    // 4. 继续处理请求
    Ok(next.run(req).await)
}
