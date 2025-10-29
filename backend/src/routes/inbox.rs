use axum::{routing::post, Router};

use crate::handlers::inbox::InboxHandler;

pub fn inbox_routes() -> Router {
    Router::new().nest(
        "/protected/inbox",
        Router::new()
            .route("/get", post(InboxHandler::get_inbox))
            .route("/mark_read", post(InboxHandler::mark_as_read))
            .route("/delete", post(InboxHandler::delete_items))
            .route("/unread_count", post(InboxHandler::get_unread_count))
            .route(
                "/register_device_token",
                post(InboxHandler::register_device_token),
            )
            .route(
                "/unregister_device_token",
                post(InboxHandler::unregister_device_token),
            ),
    )
}

