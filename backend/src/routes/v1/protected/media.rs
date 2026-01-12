use axum::{routing::post, Router};

use crate::{handlers::media, state::AppState};

/// Protected 媒体文件路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/presign_upload", post(media::presign_upload))
        .route("/complete_upload", post(media::complete_upload))
        // Chunked Upload (Optional/Enhanced)
        .route("/chunked/init", post(media::init_chunked_upload))
        .route("/chunked/finish", post(media::finish_chunked_upload))
        .route("/chunked/check", post(media::check_chunked_upload))
}
