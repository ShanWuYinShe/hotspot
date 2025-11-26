use axum::{routing::post, Router};

use crate::{handlers::comment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_list", post(comment::get_list))
        .route("/get_replies", post(comment::get_replies))
        .route("/get_likes", post(comment::get_likes))
}
