use axum::{routing::post, Router};

use crate::handlers::map as map_handlers;
use crate::state::AppState;

pub fn routes() -> Router<AppState> {
    Router::new()
        // users
        .route("/users/cluster", post(map_handlers::cluster_users))
        .route("/users/search_by_bounds", post(map_handlers::search_users_by_bounds))
        // groups
        .route("/groups/cluster", post(map_handlers::cluster_groups))
        .route("/groups/search_by_bounds", post(map_handlers::search_groups_by_bounds))
        // moments
        .route("/moments/cluster", post(map_handlers::cluster_moments))
        .route("/moments/search_by_bounds", post(map_handlers::search_moments_by_bounds))
}

