use axum::{routing::post, Router};

use crate::{handlers::group, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/discover", post(group::discover))
        .route("/open_the_door", post(group::open_the_door))
        .route("/members/get_list", post(group::get_members))
}
