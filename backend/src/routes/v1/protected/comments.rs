use axum::{routing::post, Router};

use crate::{handlers::comments, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/add", post(comments::add))
        .route("/remove", post(comments::remove))
        .route("/like_comment", post(comments::like_comment))
        .route("/unlike_comment", post(comments::unlike_comment))
}
