//! 文件与媒体上传 API

use serde::Deserialize;

use crate::types::enums::MediaCategory;

#[derive(Debug, Deserialize)]
pub struct PresignUploadRequest {
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    pub category: MediaCategory,
    pub md5: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CompleteUploadRequest {
    pub upload_id: String,
}

/// 批量预签名上传请求 - POST /api/v1/media/batch_presign_upload
#[derive(Debug, Deserialize)]
pub struct BatchPresignUploadRequest {
    pub files: Vec<PresignUploadRequest>,
}
