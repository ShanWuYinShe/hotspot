use axum::{
    routing::post,
    Router,
};

use crate::{handlers::message, state::AppState};

pub fn message_routes() -> Router<AppState> {
    Router::new()
        .route("/api/v1/protected/message/send", post(message::send))
        .route("/api/v1/protected/message/edit", post(message::edit))
        .route("/api/v1/protected/message/recall", post(message::recall))
        .route("/api/v1/protected/message/delete", post(message::delete))
        .route("/api/v1/protected/message/list", post(message::list))
        .route("/api/v1/protected/message/mark_read", post(message::mark_read))
        .route("/api/v1/protected/message/add_reaction", post(message::add_reaction))
        .route("/api/v1/protected/message/remove_reaction", post(message::remove_reaction))
        .route("/api/v1/protected/message/unread_summary", post(message::unread_summary))
}
