use axum::{routing::post, Router};

use crate::{handlers::moment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/search_by_location", post(moment::search_by_location))
}

