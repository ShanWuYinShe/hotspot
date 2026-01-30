//! 认证路由模块
//!
//! 统一的认证 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: 无需 Token 的路由（登录、注册、刷新等）
//! - token_required_routes: 需要 Token 的路由（登出、修改密码、会话管理等）

use axum::{routing::post, Router};

use crate::{handlers::auth, state::AppState};

/// Token 可选的认证路由
/// 实际上这些路由大多不需要 Token（登录、注册等）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        .route("/register", post(auth::register))
        .route("/login", post(auth::login))
        .route("/guest", post(auth::create_guest))
        .route("/refresh_token", post(auth::refresh_token))
        .route("/check_status", post(auth::check_status))
        .route(
            "/send_verification_code",
            post(auth::send_verification_code),
        )
        .route("/verify_code", post(auth::verify_code))
        .route(
            "/request_password_reset",
            post(auth::request_password_reset),
        )
        .route("/verify_reset_code", post(auth::verify_reset_code))
        .route("/reset_password", post(auth::reset_password))
}

/// Token 必须的认证路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由 - 基础认证操作
    let main_routes = Router::new().route("/logout", post(auth::logout));

    // /me 子路由 - 当前用户认证相关操作
    let me_routes = Router::new()
        .route("/change_password", post(auth::change_password))
        .route("/sessions", post(auth::list_sessions))
        .route("/sessions/revoke", post(auth::revoke_session));

    main_routes.nest("/me", me_routes)
}
