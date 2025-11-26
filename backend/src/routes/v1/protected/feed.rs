use axum::{routing::post, Router};

use crate::{handlers::feed, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_list", post(feed::get_list))
        .route("/feedback", post(feed::feedback))
}

