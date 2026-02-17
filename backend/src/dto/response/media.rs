//! 文件上传相关响应类型

use serde::Serialize;

/// 上传媒体数据
#[derive(Debug, Serialize)]
pub struct UploadMediaData {
    pub file_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    pub file_size: i64,
    pub mime_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
}

/// 预签名上传数据 - API 文档: PresignUploadData
#[derive(Debug, Serialize)]
pub struct PresignUploadData {
    pub upload_id: String,
    pub upload_url: String,
    pub headers: std::collections::HashMap<String, String>,
    pub expires_at: String,
}

/// 批量预签名上传响应 - API 文档: BatchPresignedUpload
#[derive(Debug, Serialize)]
pub struct BatchPresignedUpload {
    pub uploads: Vec<PresignUploadData>,
}
