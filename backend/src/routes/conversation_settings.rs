use axum::{routing::post, Router};

use crate::handlers::conversation_settings::ConversationSettingsHandler;

pub fn conversation_settings_routes() -> Router {
    Router::new().nest(
        "/protected/conversation_settings",
        Router::new()
            .route("/get", post(ConversationSettingsHandler::get_settings))
            .route("/update", post(ConversationSettingsHandler::update_settings)),
    )
}

