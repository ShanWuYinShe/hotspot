use axum::{routing::post, Router};

use crate::{handlers::subscriptions, state::AppState};

/// Protected 通知订阅路由 - 当前用户订阅设置
pub fn routes() -> Router<AppState> {
    // /me 子路由 - 当前用户订阅配置
    let me_routes = Router::new()
        .route("/", post(subscriptions::get_me))
        .route("/update", post(subscriptions::update_me));

    Router::new().nest("/me", me_routes)
}
