use axum::{routing::post, Router};

use crate::{handlers::comments, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(comments::create))
        .route("/delete", post(comments::delete))
        .route("/like", post(comments::like))
        .route("/unlike", post(comments::unlike))
}
