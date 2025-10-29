use axum::{routing::post, Router};

use crate::handlers::auth;

pub fn public_routes() -> Router {
    Router::new()
        .route("/register", post(auth::register))
        .route("/login", post(auth::login))
        .route("/create_temp", post(auth::create_temp))
        .route("/refresh_token", post(auth::refresh_token))
}

pub fn protected_routes() -> Router {
    Router::new()
        .route("/logout", post(auth::logout))
    // TODO: 添加其他受保护的认证端点
}

