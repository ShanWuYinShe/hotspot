use axum::{routing::post, Router};

use crate::{handlers::leaderboard, state::AppState};

/// Public 排行榜路由
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_creators", post(leaderboard::get_creators))
        .route("/get_moments", post(leaderboard::get_moments))
}
