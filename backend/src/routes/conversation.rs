use axum::{
    routing::post,
    Router,
};

use crate::{handlers::conversation, state::AppState};

pub fn conversation_routes() -> Router<AppState> {
    Router::new()
        .route("/api/v1/protected/conversation/list", post(conversation::list))
        .route("/api/v1/protected/conversation/pin", post(conversation::pin))
        .route("/api/v1/protected/conversation/unpin", post(conversation::unpin))
        .route("/api/v1/protected/conversation/mute", post(conversation::mute))
        .route("/api/v1/protected/conversation/unmute", post(conversation::unmute))
        .route("/api/v1/protected/conversation/delete", post(conversation::delete))
}
