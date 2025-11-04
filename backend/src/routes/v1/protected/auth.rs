use axum::{routing::post, Router};

use crate::{handlers::auth, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/logout", post(auth::logout))
        .route("/check_status", post(auth::check_status))
        .route("/set_password", post(auth::set_password))
        .route("/list_sessions", post(auth::list_sessions))
        .route("/revoke_session", post(auth::revoke_session))
}

