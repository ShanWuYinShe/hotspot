use axum::{routing::post, Router};

use crate::{handlers::chat_messages, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/send", post(chat_messages::send))
        .route("/edit", post(chat_messages::edit))
        .route("/recall", post(chat_messages::recall))
        .route("/delete", post(chat_messages::delete))
        .route("/list", post(chat_messages::list))
        .route("/mark_read", post(chat_messages::mark_read))
        .route("/mark_all_read", post(chat_messages::mark_all_read))
        .route("/unread_summary", post(chat_messages::unread_summary))
        .route("/unread_count", post(chat_messages::unread_count))
        .route("/react", post(chat_messages::add_reaction))
        .route("/reactions", post(chat_messages::list_reactions))
        .route("/remove_reaction", post(chat_messages::remove_reaction))
}

