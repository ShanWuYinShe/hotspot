use axum::{routing::post, Router};

use crate::handlers::leaderboard::LeaderboardHandler;

pub fn leaderboard_routes() -> Router {
    Router::new().nest(
        "/protected/leaderboard",
        Router::new().route("/get", post(LeaderboardHandler::get_leaderboard)),
    )
}

