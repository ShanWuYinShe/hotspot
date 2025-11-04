use axum::{
    routing::post,
    Router,
};

use crate::{
    handlers::system_config,
    state::AppState,
};

pub fn admin_routes() -> Router<AppState> {
    Router::new()
        // 系统配置管理
        .route("/config/get", post(system_config::get_config))
        .route("/config/list", post(system_config::list_configs))
        .route("/config/create", post(system_config::create_config))
        .route("/config/update", post(system_config::update_config))
        .route("/config/delete", post(system_config::delete_config))
        .route("/config/history", post(system_config::get_config_history))
        .route("/config/changes", post(system_config::list_all_changes))
}

