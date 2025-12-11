use axum::{routing::post, Router};

use crate::{handlers::inbox, state::AppState};

/// 推送设备管理路由
/// 
/// 挂载路径: /api/v1/protected/device_tokens
/// API 规范路径:
/// - POST /api/v1/protected/device_tokens/me/register
/// - POST /api/v1/protected/device_tokens/me/unregister
/// - POST /api/v1/protected/device_tokens/me/get_list
pub fn routes() -> Router<AppState> {
    // /me 子路由 - 当前用户设备管理
    let me_routes = Router::new()
        .route("/register", post(inbox::register_device_token))
        .route("/unregister", post(inbox::unregister_device_token))
        .route("/get_list", post(inbox::get_device_tokens));

    Router::new().nest("/me", me_routes)
}
