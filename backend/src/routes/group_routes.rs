use axum::{Router, routing::post};

use crate::{controllers::group_controller, middleware::auth::AppState};
use std::sync::Arc;

pub fn group_routes() -> Router<Arc<AppState>> {
    Router::new()
        .route("/create", post(group_controller::create_group))
        .route("/search_by_location", post(group_controller::search_group_by_location))
        .route("/search_by_name", post(group_controller::search_group_by_name))
        .route("/search_by_id", post(group_controller::search_group_by_id))
        .route("/join", post(group_controller::join_group))
        .route("/leave", post(group_controller::leave_group))
        .route("/members", post(group_controller::group_members))
        .route("/transfer_ownership", post(group_controller::transfer_ownership))
        .route("/kick", post(group_controller::kick_member))
        .route("/user_groups", post(group_controller::get_user_groups))
        .route("/owned_groups", post(group_controller::get_owned_groups))
}
