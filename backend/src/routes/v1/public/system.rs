use axum::{routing::post, Router};

use crate::{handlers::system, state::AppState};

/// Public 系统路由
/// 
/// 提供系统健康检查等公开接口
pub fn routes() -> Router<AppState> {
    Router::new()
        // 系统健康检查
        .route("/health", post(system::health))
        // 系统配置（公开部分）
        .route("/config", post(system::config))
        // 系统统计（需要管理员权限，但放在public供调试使用）
        .route("/stats", post(system::stats))
}










