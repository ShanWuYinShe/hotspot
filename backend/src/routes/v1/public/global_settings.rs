use axum::{routing::post, Router};

use crate::{handlers::system_config, state::AppState};

/// Public Global Settings Routes
pub fn routes() -> Router<AppState> {
    Router::new().route("/fetch", post(system_config::fetch_global_settings))
}
