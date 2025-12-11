use axum::{routing::post, Router};

use crate::{handlers::comments, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_list", post(comments::get_list))
        .route("/get_replies", post(comments::get_replies))
        .route("/get_likes_list", post(comments::get_likes_list))
}
