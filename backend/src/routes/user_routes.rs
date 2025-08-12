use axum::{Router, routing::post};

use crate::{
    controllers::user_controller, 
    middleware::auth::AppState
};
use std::sync::Arc;

/// 用户数据管理路由
/// 处理用户资料、设置、搜索等用户数据相关功能（需要认证）
pub fn user_routes() -> Router<Arc<AppState>> {
    Router::new()
        // 用户搜索和发现功能
        .route("/search_by_name", post(user_controller::search_user_by_name))
        .route("/search_by_id", post(user_controller::search_user_by_id))
        .route("/search_by_location", post(user_controller::search_user_by_location))
        
        // 用户位置管理
        .route("/update_location", post(user_controller::update_location))
        
        // 用户资料管理
        .route("/update_profile", post(user_controller::update_nickname))
        .route("/upgrade_account", post(user_controller::upgrade_account))
        .route("/delete_account", post(user_controller::delete_account))
        
        // 用户设置管理
        .route("/get_settings", post(user_controller::get_user_settings))
        .route("/update_settings", post(user_controller::update_user_settings))
}
