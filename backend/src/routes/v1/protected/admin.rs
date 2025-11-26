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
        .route("/config/create", post(system_config::create))
        .route("/config/get_config", post(system_config::get_config))
        .route("/config/get_list", post(system_config::get_list))
        .route("/config/update", post(system_config::update))
        .route("/config/delete", post(system_config::delete))
        .route("/config/get_config_history", post(system_config::get_config_history))
        .route("/config/list_all_changes", post(system_config::list_all_changes))
}
