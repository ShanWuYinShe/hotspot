use axum::{routing::post, Router};

use crate::{handlers::inbox, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_list", post(inbox::get_list))
        .route("/get_unread_count", post(inbox::get_unread_count))
        .route("/mark_all_as_read", post(inbox::mark_all_as_read))
        .route("/remove", post(inbox::remove))
        .route("/remove_batch", post(inbox::remove_batch))
        .route("/register_device_token", post(inbox::register_device_token))
        .route("/unregister_device_token", post(inbox::unregister_device_token))
}
