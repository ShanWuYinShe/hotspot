use axum::{routing::post, Router};

use crate::{handlers::inbox::InboxHandler, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/list", post(InboxHandler::get_inbox))
        .route("/mark_read", post(InboxHandler::mark_as_read))
        .route("/delete", post(InboxHandler::delete_items))
        .route("/unread_summary", post(InboxHandler::get_unread_count))
        .route("/update_device_token", post(InboxHandler::update_device_token))
}

