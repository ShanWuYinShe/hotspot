use axum::{routing::get, Router};

use crate::{state::AppState, websocket::handler::ws_handler};

pub fn websocket_routes() -> Router<AppState> {
    Router::new().route("/ws/v1", get(ws_handler))
}
