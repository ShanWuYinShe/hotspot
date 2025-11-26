use axum::{routing::post, Router};

use crate::{handlers::moment, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/search_by_location", post(moment::search_by_location))
        .route("/search_by_tags", post(moment::search_by_tags))
        .route("/open_the_door", post(moment::open_the_door))
        .route("/get_list", post(moment::get_list))
        .route("/get_likes", post(moment::get_likes))
}
