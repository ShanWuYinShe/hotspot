use crate::types::enums::UploadStatus;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MediaFile {
    pub file_id: uuid::Uuid,
    pub uploader_id: i64,
    pub url: String,
    pub thumbnail_url: Option<String>,
    pub storage_path: String,
    pub meta: Option<serde_json::Value>,
    pub hash_md5: Option<String>,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UploadSession {
    pub upload_id: uuid::Uuid,
    pub user_id: i64,
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    // category enum handled via string or custom type
    pub upload_url: String,
    pub storage_path: String,
    pub status: UploadStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}
