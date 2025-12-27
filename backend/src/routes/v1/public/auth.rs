use axum::{routing::post, Router};

use crate::{handlers::auth, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/register", post(auth::register))
        .route("/login", post(auth::login))
        .route("/guest", post(auth::create_guest))
        .route("/refresh_token", post(auth::refresh_token))
        .route(
            "/request_password_reset",
            post(auth::request_password_reset),
        )
        .route("/verify_reset_code", post(auth::verify_reset_code))
        .route("/reset_password", post(auth::reset_password))
}
