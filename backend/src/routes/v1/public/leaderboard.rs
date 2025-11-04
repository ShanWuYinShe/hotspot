use axum::{routing::post, Router};

use crate::{handlers::leaderboard, state::AppState};

/// Public 排行榜路由
/// 
/// 提供排行榜查询接口（不需要认证）
pub fn routes() -> Router<AppState> {
    Router::new()
        // 获取排行榜
        .route("/list", post(leaderboard::list))
        // 获取我的排名（需要认证，但通过 handler 内部处理）
        .route("/my_rank", post(leaderboard::my_rank))
}










