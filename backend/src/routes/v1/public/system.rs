use axum::{routing::post, Router};

use crate::{handlers::system, state::AppState};

/// Public 系统路由
///
/// 提供系统健康检查等公开接口
pub fn routes() -> Router<AppState> {
    Router::new()
        // 系统健康检查
        .route("/health", post(system::health))
}
