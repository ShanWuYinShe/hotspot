use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

/// Public 用户路由
/// 
/// 这些接口不需要认证，任何人都可以访问
pub fn routes() -> Router<AppState> {
    Router::new()
        // 用户搜索接口（公开）
        .route("/search_by_name", post(user::search_by_name))
        .route("/search_by_id", post(user::search_by_id))
        .route("/search_by_location", post(user::search_by_location))
}










