use axum::{routing::post, Router};

use crate::{handlers::media, state::AppState};

pub fn media_routes() -> Router<AppState> {
    Router::new()
        .route("/api/v1/protected/media/upload_image", post(media::upload_image))
        .route("/api/v1/protected/media/upload_file", post(media::upload_file))
        .route("/api/v1/protected/media/upload_token", post(media::upload_token))
}

