use axum::{Router, routing::post};

use crate::{controllers::moment_controller, middleware::auth::AppState};
use std::sync::Arc;

pub fn moment_routes() -> Router<Arc<AppState>> {
    Router::new()
        .route("/create", post(moment_controller::create_moment))
        .route("/delete", post(moment_controller::delete_moment))
        .route("/history", post(moment_controller::history))
        .route("/search_by_location", post(moment_controller::search_moment_by_location))
        .route("/search_by_tags", post(moment_controller::search_moment_by_tags))
        .route("/search_by_id", post(moment_controller::search_moment_by_id))
        .route("/like", post(moment_controller::like_moment))
        .route("/unlike", post(moment_controller::unlike_moment))
}
