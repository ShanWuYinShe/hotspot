use axum::{Router, routing::post};

use crate::{
    controllers::user_controller, 
    middleware::auth::AppState
};
use std::sync::Arc;

// 合并的用户路由（认证由中间件统一判断）
pub fn user_routes() -> Router<Arc<AppState>> {
    Router::new()
        // 公开路由（不需要认证，由auth_middleware自动判断）
        .route("/register", post(user_controller::register))
        .route("/login", post(user_controller::login))
        .route("/temp", post(user_controller::create_temp_user))
        .route("/refresh_token", post(user_controller::refresh_token))
        // 需要认证的路由
        .route("/search_by_name", post(user_controller::search_user_by_name))
        .route("/search_by_id", post(user_controller::search_user_by_id))
        .route("/search_by_location", post(user_controller::search_user_by_location))
        .route("/update_location", post(user_controller::update_location))
        .route("/set_password", post(user_controller::set_password))
        .route("/auth/status", post(user_controller::check_auth_status))
}
