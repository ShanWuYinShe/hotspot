//! 媒体文件路由模块
//!
//! 统一的媒体上传 API 路由
//! 所有媒体接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::media, state::AppState};

/// Token 必须的媒体路由
pub fn token_required_routes() -> Router<AppState> {
    Router::new()
        .route("/presign_upload", post(media::presign_upload))
        .route("/complete_upload", post(media::complete_upload))
}
