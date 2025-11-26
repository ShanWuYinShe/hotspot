use axum::{routing::post, Router};

use crate::{handlers::system_config, state::AppState};

/// Protected 系统配置管理路由
/// 
/// 这些接口需要认证和管理员权限
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_list", post(system_config::get_list))  // ✅ 获取配置列表（管理员）
        .route("/update", post(system_config::update))  // ✅ 更新配置（管理员）
}






