use axum::{routing::post, Router};

use crate::handlers::map as map_handlers;
use crate::state::AppState;

pub fn routes() -> Router<AppState> {
    Router::new().route("/render", post(map_handlers::render))
}
