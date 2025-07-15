use axum::{Router, routing::post};

use crate::{controllers::message_controller, middleware::auth::AppState};
use std::sync::Arc;

pub fn message_routes() -> Router<Arc<AppState>> {
    Router::new()
        .route("/group/send", post(message_controller::send_group_message))
        .route("/group/history", post(message_controller::group_history))
        .route("/group/delete", post(message_controller::delete_group_message))
        .route("/group/mark-read", post(message_controller::mark_group_messages_read))
        .route("/group/unread_count", post(message_controller::get_group_unread_count))
        .route("/user/send", post(message_controller::send_direct_message))
        .route("/user/history", post(message_controller::user_history))
        .route("/user/conversations", post(message_controller::user_conversations))
        .route("/user/delete", post(message_controller::delete_direct_message))
        .route("/user/mark-read", post(message_controller::mark_direct_messages_read))
        .route("/user/unread_count", post(message_controller::get_direct_message_unread_count))
}
