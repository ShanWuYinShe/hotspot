//! 管理员路由模块
//!
//! 统一的管理员 API 路由
//! 所有管理员接口都需要 Token + 管理员权限

use axum::{routing::post, Router};

use crate::{
    handlers::{
        admin_audit, admin_rbac, announcements, group_chats, reports, system_config, tags, user,
    },
    state::AppState,
};

/// 管理员路由（需要 Token + 管理员权限）
pub fn admin_routes() -> Router<AppState> {
    Router::new()
        // RBAC（管理员权限管理）
        .route("/rbac/get_me", post(admin_rbac::me))
        .route("/rbac/list_permissions", post(admin_rbac::list_permissions))
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
        // 审计日志
        .route("/audit/list", post(admin_audit::list))
        .route("/audit/open_the_door", post(admin_audit::detail))
        // 举报审核（管理员）
        .route("/reports/search", post(reports::get_list))
        .route("/reports/handle", post(reports::handle))
        // Server Config
        .route(
            "/server_config/get",
            post(system_config::admin_get_global_settings),
        )
        .route(
            "/server_config/update_upload",
            post(system_config::admin_update_upload_config),
        )
        .route(
            "/server_config/update_defaults",
            post(system_config::admin_update_defaults_config),
        )
        .route(
            "/server_config/update_rate_limits",
            post(system_config::admin_update_rate_limits_config),
        )
        .route(
            "/server_config/update_response_defaults",
            post(system_config::admin_update_response_defaults_config),
        )
        // Client Config
        .route(
            "/client_config/get",
            post(system_config::admin_get_client_config),
        )
        .route(
            "/client_config/update_features",
            post(system_config::admin_update_features_config),
        )
        .route(
            "/client_config/update_limits",
            post(system_config::admin_update_limits_config),
        )
        .route(
            "/client_config/update_contact",
            post(system_config::admin_update_contact_config),
        )
        // 用户管理
        .route("/users/search", post(user::admin_search))
        .route("/users/open_the_door", post(user::admin_detail))
        .route("/users/ban", post(user::admin_ban))
        .route("/users/unban", post(user::admin_unban))
        // 群组管理
        .route("/groups/search", post(group_chats::admin_search))
        .route("/groups/open_the_door", post(group_chats::admin_detail))
        .route("/groups/dismiss", post(group_chats::admin_dismiss))
        // 标签管理
        .route("/tags/search", post(tags::admin_search))
        .route("/tags/delete", post(tags::admin_delete))
        // 公告管理
        .route("/announcements/search", post(announcements::admin_list))
        .route("/announcements/create", post(announcements::admin_create))
        .route("/announcements/update", post(announcements::admin_update))
        .route("/announcements/delete", post(announcements::admin_delete))
}
