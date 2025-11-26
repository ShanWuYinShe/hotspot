use axum::{routing::post, Router};

use crate::{handlers::subscription, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_config", post(subscription::get_config))
        .route("/", post(subscription::update))
}
