use axum::{routing::post, Router};

use crate::{handlers::feed::FeedHandler, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/list", post(FeedHandler::get_feed))
        .route("/feedback", post(FeedHandler::submit_feedback))
}

