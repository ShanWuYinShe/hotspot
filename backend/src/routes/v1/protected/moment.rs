use axum::{routing::post, Router};

use crate::{handlers::moment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(moment::create))
        .route("/update", post(moment::update))
        .route("/delete", post(moment::delete))
        .route("/toggle_like", post(moment::toggle_like))
        .route("/toggle_favorite", post(moment::toggle_favorite))
        .route("/get_favorites", post(moment::get_favorites))
}
