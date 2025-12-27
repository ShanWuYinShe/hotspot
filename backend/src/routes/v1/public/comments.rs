use axum::{routing::post, Router};

use crate::{handlers::comments, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/", post(comments::get_comments))
        .route("/replies", post(comments::get_replies))
        .route("/likes", post(comments::get_likes))
}
