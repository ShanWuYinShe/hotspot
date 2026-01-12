use axum::{routing::post, Router};

use crate::{
    handlers::{admin_rbac, announcements, group_chats, reports, system_config, tags, user},
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
        // 举报审核（管理员）
        .route("/reports", post(reports::get_list))
        .route("/reports/handle", post(reports::handle))
        // Global Settings (Strongly Typed)
        .route(
            "/global_settings",
            post(system_config::admin_get_global_settings),
        )
        .route(
            "/global_settings/update_upload",
            post(system_config::admin_update_upload_config),
        )
        .route(
            "/global_settings/update_defaults",
            post(system_config::admin_update_defaults_config),
        )
        .route(
            "/global_settings/update_features",
            post(system_config::admin_update_features_config),
        )
        .route(
            "/global_settings/update_contact",
            post(system_config::admin_update_contact_config),
        )
        .route(
            "/global_settings/update_limits",
            post(system_config::admin_update_limits_config),
        )
        // 用户管理
        .route("/users/search", post(user::admin_search))
        .route("/users/ban", post(user::admin_ban))
        .route("/users/unban", post(user::admin_unban))
        // 群组管理
        .route("/groups/search", post(group_chats::admin_search))
        .route("/groups/dismiss", post(group_chats::admin_dismiss))
        // 标签管理
        .route("/tags/delete", post(tags::admin_delete))
        // 公告管理
        .route("/announcements/create", post(announcements::admin_create))
        .route("/announcements/list", post(announcements::admin_list))
        .route("/announcements/delete", post(announcements::admin_delete))
}
