use axum::{routing::post, Router};

use crate::{handlers::media, state::AppState};

/// Protected 媒体文件路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        // POST /api/v1/protected/media/upload_file - 上传文件（≤100MB）
        .route("/upload_file", post(media::upload_file))
        // POST /api/v1/protected/media/upload_image - 上传图片
        .route("/upload_image", post(media::upload_image))
        // POST /api/v1/protected/media/upload_token - 获取上传令牌
        .route("/upload_token", post(media::upload_token))
        // POST /api/v1/protected/media/initiate_upload - 初始化大文件上传
        .route("/initiate_upload", post(media::initiate_upload))
        // POST /api/v1/protected/media/complete_upload - 完成大文件上传
        .route("/complete_upload", post(media::complete_upload))
        // POST /api/v1/protected/media/get_upload_status - 查询上传状态
        .route("/get_upload_status", post(media::get_upload_status))
        // POST /api/v1/protected/media/abort_upload - 取消大文件上传
        .route("/abort_upload", post(media::abort_upload))
}
