use axum::{Router, routing::post};

use crate::{controllers::checkin_controller, middleware::auth::AppState};
use std::sync::Arc;

pub fn checkin_routes() -> Router<Arc<AppState>> {
    Router::new()
        .route("/create", post(checkin_controller::create_checkin))
        .route("/delete", post(checkin_controller::delete_checkin))
        .route("/history", post(checkin_controller::history))
        .route("/search_by_location", post(checkin_controller::search_checkin_by_location))
        .route("/search_by_tags", post(checkin_controller::search_checkin_by_tags))
        .route("/search_by_id", post(checkin_controller::search_checkin_by_id))
        .route("/like", post(checkin_controller::like_checkin))
        .route("/unlike", post(checkin_controller::unlike_checkin))
}
