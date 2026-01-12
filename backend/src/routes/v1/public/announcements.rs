use axum::{routing::post, Router};

use crate::{handlers::announcements, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new().route("/", post(announcements::public_list))
}
