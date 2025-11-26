use axum::{routing::post, Router};

use crate::{handlers::leaderboard, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/my_creator_rank", post(leaderboard::my_creator_rank))
        .route("/my_moment_rank", post(leaderboard::my_moment_rank))
}


