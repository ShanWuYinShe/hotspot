use axum::{routing::post, Router};

use crate::handlers::feed::FeedHandler;

pub fn feed_routes() -> Router {
    Router::new().nest(
        "/protected/feed",
        Router::new()
            .route("/get", post(FeedHandler::get_feed))
            .route("/feedback", post(FeedHandler::submit_feedback)),
    )
}

