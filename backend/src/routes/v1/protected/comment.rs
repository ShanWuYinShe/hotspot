use axum::{routing::post, Router};

use crate::{handlers::comment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/add", post(comment::add))
        .route("/remove", post(comment::remove))
        .route("/toggle_like", post(comment::toggle_like))
}
