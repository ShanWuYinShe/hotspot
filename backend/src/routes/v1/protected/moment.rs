use axum::{routing::post, Router};

use crate::{handlers::moment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(moment::create))
        .route("/update", post(moment::update))
        .route("/delete", post(moment::delete))
        .route("/like", post(moment::like))
        .route("/unlike", post(moment::unlike))
        .route("/collect", post(moment::collect))
        .route("/uncollect", post(moment::uncollect))
}

