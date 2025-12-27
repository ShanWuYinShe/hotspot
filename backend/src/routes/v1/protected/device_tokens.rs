use axum::{routing::post, Router};

use crate::{handlers::device_tokens, state::AppState};

/// 推送设备管理路由
///
/// 挂载路径: /api/v1/protected/device_tokens
///
/// - POST /api/v1/protected/device_tokens/me/register
/// - POST /api/v1/protected/device_tokens/me/unregister
/// - POST /api/v1/protected/device_tokens/me
pub fn routes() -> Router<AppState> {
    let me_routes = Router::new()
        .route("/register", post(device_tokens::register_device_token))
        .route("/unregister", post(device_tokens::unregister_device_token))
        .route("/", post(device_tokens::me));

    Router::new().nest("/me", me_routes)
}
