use axum::{routing::post, Router};

use crate::{handlers::comment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/add", post(comment::add))
        .route("/delete", post(comment::delete))
        .route("/like", post(comment::like))
        .route("/unlike", post(comment::unlike))
}

