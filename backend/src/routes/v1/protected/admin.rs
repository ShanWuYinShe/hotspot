use axum::{routing::post, Router};

use crate::{
    handlers::{admin_rbac, reports, system_config},
    state::AppState,
};

pub fn admin_routes() -> Router<AppState> {
    Router::new()
        // RBAC（管理员权限管理）
        .route("/rbac/me", post(admin_rbac::me))
        .route("/rbac/permissions", post(admin_rbac::list_permissions))
        .route("/rbac/roles/list", post(admin_rbac::list_roles))
        .route("/rbac/roles/create", post(admin_rbac::create_role))
        .route("/rbac/roles/update", post(admin_rbac::update_role))
        .route("/rbac/roles/delete", post(admin_rbac::delete_role))
        .route(
            "/rbac/roles/set_permissions",
            post(admin_rbac::set_role_permissions),
        )
        .route("/rbac/admins/list", post(admin_rbac::list_admins))
        .route("/rbac/admins/grant", post(admin_rbac::grant_admin))
        .route("/rbac/admins/revoke", post(admin_rbac::revoke_admin))
        // 系统配置管理
        .route("/config/create", post(system_config::create))
        // list
        .route("/config", post(system_config::get_list))
        // detail
        .route("/config/detail", post(system_config::get_config))
        .route("/config/update", post(system_config::update))
        .route("/config/delete", post(system_config::delete))
        // history
        .route("/config/history", post(system_config::get_config_history))
        // changes
        .route("/config/changes", post(system_config::get_changes))
        // 举报审核（管理员）
        .route("/reports", post(reports::get_list))
        .route("/reports/handle", post(reports::handle))
}
