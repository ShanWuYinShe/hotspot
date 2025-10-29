use axum::{routing::post, Router};

use crate::{handlers::system, state::AppState};

pub fn system_routes() -> Router<AppState> {
    Router::new()
        .route("/api/v1/public/system/health", post(system::health))
        .route("/api/v1/public/system/config", post(system::config))
        .route("/api/v1/protected/system/stats", post(system::stats))
}

