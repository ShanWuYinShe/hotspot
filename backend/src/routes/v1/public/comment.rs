use axum::{routing::post, Router};

use crate::{handlers::comment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/list", post(comment::list))
        .route("/replies", post(comment::replies))
}

