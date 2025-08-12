use axum::{Router, routing::post};

use crate::{
    controllers::user_controller, 
    middleware::auth::AppState
};
use std::sync::Arc;

/// 公开认证路由
/// 处理不需要token验证的认证相关功能：登录、注册、临时用户创建等
pub fn auth_routes() -> Router<Arc<AppState>> {
    Router::new()
        // 公开认证路由（不需要token验证）
        .route("/register", post(user_controller::register))
        .route("/login", post(user_controller::login))
        .route("/create_temp", post(user_controller::create_temp_user))
        .route("/refresh_token", post(user_controller::refresh_token))
        .route("/health", post(user_controller::public_health_check))
}

/// 需要认证的用户路由
/// 处理需要token验证的用户相关功能
pub fn auth_protected_routes() -> Router<Arc<AppState>> {
    Router::new()
        .route("/check_status", post(user_controller::check_auth_status))
        .route("/set_password", post(user_controller::set_password))
        .route("/password_status", post(user_controller::check_password_status))
}
