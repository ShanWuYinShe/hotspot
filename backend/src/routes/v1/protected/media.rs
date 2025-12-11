use axum::{routing::post, Router};

use crate::{handlers::media, state::AppState};

/// Protected 媒体文件路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/upload_file", post(media::upload_file))
        .route("/initiate_upload", post(media::initiate_upload))
        .route("/upload_chunk", post(media::upload_chunk))
        .route("/complete_upload", post(media::complete_upload))
        .route("/get_upload_status", post(media::get_upload_status))
        .route("/abort_upload", post(media::abort_upload))
}
