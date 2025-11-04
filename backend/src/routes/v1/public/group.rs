use axum::{routing::post, Router};

use crate::{handlers::group, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/search_by_name", post(group::search_by_name))
        .route("/search_by_location", post(group::search_by_location))
        .route("/search_by_id", post(group::search_by_id))
}

