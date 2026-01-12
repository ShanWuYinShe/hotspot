//! 媒体服务
//!
//! 基于 S3 (MinIO) 的 Presigned URL 上传流程
//! 同时也支持 S3 Multipart Upload (如果需要)

use aws_sdk_s3::Client as S3Client;
use aws_sdk_s3::presigning::PresigningConfig;
use aws_sdk_s3::operation::head_object::HeadObjectError;
use sqlx::PgPool;
use std::time::Duration;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CheckChunkedUploadRequest, CompleteUploadRequest, FinishChunkedUploadRequest,
            InitChunkedUploadRequest, PresignUploadRequest,
        },
        response::{
            InitiateUploadData, PresignUploadData, UploadMediaData,
        },
    },
    error::{AppError, Result},
    repositories::file_repo::FileRepository,
};

#[derive(Clone)]
pub struct MediaService {
    s3_internal: S3Client,
    s3_public: S3Client,
    bucket: String,
    file_repo: FileRepository,
}

impl MediaService {
    pub fn new(
        db: &PgPool,
        s3_internal: S3Client,
        s3_public: S3Client,
        bucket: String,
    ) -> Self {
        Self {
            file_repo: FileRepository::new(db.clone()),
            s3_internal,
            s3_public,
            bucket,
        }
    }

    /// 申请上传预签名 URL (Simple PUT)
    pub async fn presign_upload(
        &self,
        user_id: i64,
        req: PresignUploadRequest,
    ) -> Result<PresignUploadData> {
        // 简单校验
        if req.file_size > 100 * 1024 * 1024 {
             return Err(AppError::BadRequest("File too large".to_string()));
        }
        // Category 校验 (可选)
        if req.category.is_empty() {
             return Err(AppError::BadRequest("Category required".to_string()));
        }

        // Suppress unused md5 warning
        let _ = req.md5;

        let file_id = Uuid::new_v4();
        let extension = req
            .file_name
            .rsplit('.')
            .next()
            .unwrap_or("bin")
            .to_lowercase();
        
        // S3 Key: uploads/{user_id}/{uuid}.{ext}
        // 使用 user_id 分隔目录便于管理
        let key = format!("uploads/{}/{}.{}", user_id, file_id, extension);

        // 生成预签名 URL (使用 public client，确保域名对客户端可达)
        let expires_in = Duration::from_secs(3600); // 1小时有效
        let presigned_req = self
            .s3_public
            .put_object()
            .bucket(&self.bucket)
            .key(&key)
            .content_type(&req.mime_type)
            // .content_md5(req.md5.as_deref().unwrap_or("")) // 如果前端传了MD5，可以校验
            .presigned(PresigningConfig::expires_in(expires_in).map_err(|e| {
                AppError::InternalError(format!("Failed to config presign: {}", e))
            })?)
            .await
            .map_err(|e| AppError::InternalError(format!("Failed to presign upload: {}", e)))?;

        // 记录上传意图到 Redis (可选，用于 complete 时的校验，或者防滥用)
        // key: upload_pending:{upload_id} -> { metadata }
        // 这里为了简单，暂不强校验 Redis，complete 时直接查 S3

        // 构造 headers map
        let mut headers = std::collections::HashMap::new();
        headers.insert("Content-Type".to_string(), req.mime_type.clone());

        Ok(PresignUploadData {
            upload_id: key, // 这里直接用 S3 Key 作为 upload_id，或者存 Redis 映射
            upload_url: presigned_req.uri().to_string(),
            headers,
            expires_at: (chrono::Utc::now() + chrono::Duration::from_std(expires_in)
                .map_err(|e| AppError::InternalError(format!("Invalid duration: {}", e)))?)
                .to_rfc3339(),
        })
    }

    /// 完成上传
    pub async fn complete_upload(
        &self,
        user_id: i64,
        req: CompleteUploadRequest,
    ) -> Result<UploadMediaData> {
        let key = req.upload_id; // upload_id 即为 S3 Key

        // 1. 检查 S3 对象是否存在 (使用 internal client)
        let head_output = self
            .s3_internal
            .head_object()
            .bucket(&self.bucket)
            .key(&key)
            .send()
            .await
            .map_err(|e| match e.into_service_error() {
                HeadObjectError::NotFound(_) => {
                    AppError::FileNotFound("File not found in storage".to_string())
                }
                err => AppError::InternalError(format!("S3 error: {}", err)),
            })?;

        let size = head_output.content_length.ok_or_else(|| AppError::InternalError("File size missing".to_string()))?;
        let content_type = head_output.content_type.unwrap_or_default();
        let etag = head_output.e_tag.unwrap_or_default().replace("\"", "");

        // 2. 解析文件名/扩展名
        let file_name = key.rsplit('/').next().unwrap_or("file.bin").to_string();
        let _extension = file_name.rsplit('.').next().unwrap_or("").to_string();

        // 3. 生成 file_id (UUID)
        // 注意：之前的 presign 阶段生成的 uuid 是 key 的一部分，这里作为 file_id
        // key format: uploads/{user_id}/{uuid}.{ext}
        let uuid_part = file_name.rsplit('.').skip(1).next().unwrap_or(&file_name);
        let file_id = Uuid::parse_str(uuid_part).unwrap_or_else(|_| Uuid::new_v4());

        // 4. 创建 DB 记录
        // URL 策略：这里存相对路径或绝对路径？
        // 建议存相对路径或完整 Key，展示时拼接 CDN 前缀
        // 这里存 S3 Key
        let storage_path = key.clone();
        
        // 外部访问 URL (可以是 presigned GET，或者 public read URL)
        // 假设 Bucket 是 Public Read，直接拼接
        // 需要知道 S3_PUBLIC_URL 基础部分。
        // 这里暂时存 Key，转换逻辑在 DTO 转换层或前端。
        // 为了兼容旧逻辑，这里构造一个 URL string
        let url = format!("/{}", storage_path); // 简单处理

        let category = "general"; // TODO: 从 path 或 redis 恢复 category

        let file_record = self
            .file_repo
            .create_file_with_id(
                file_id,
                user_id,
                category,
                Some(&file_name),
                &storage_path,
                &url,
                None, // thumbnail
                &content_type,
                size,
                None,
                None,
                None,
                Some(&etag),
            )
            .await?;

        Ok(UploadMediaData {
            file_id: file_record.file_id.to_string(),
            thumbnail_url: None,
            file_size: size,
            mime_type: content_type,
            duration: None,
            width: None,
            height: None,
        })
    }

    // ==================== Chunked Upload (TODO) ====================
    // 这里可以实现 S3 Multipart Upload 的封装

    pub async fn init_chunked_upload(
        &self,
        _user_id: i64,
        req: InitChunkedUploadRequest,
    ) -> Result<InitiateUploadData> {
        // Use fields to avoid warnings
        let _ = req.file_name;
        let _ = req.file_size;
        let _ = req.mime_type;
        let _ = req.category;
        let _ = req.chunk_count;
        // TODO: Implement S3 CreateMultipartUpload
        Err(AppError::InternalError("Not implemented yet".to_string()))
    }

    pub async fn finish_chunked_upload(
        &self,
        _user_id: i64,
        req: FinishChunkedUploadRequest,
    ) -> Result<UploadMediaData> {
        let _ = req.upload_id;
        // TODO: Implement S3 CompleteMultipartUpload
        Err(AppError::InternalError("Not implemented yet".to_string()))
    }

    pub async fn check_chunked_upload(
        &self,
        _user_id: i64,
        req: CheckChunkedUploadRequest,
    ) -> Result<crate::dto::response::CheckChunkedUploadData> {
        let _ = req.upload_id;
        // TODO: Implement S3 ListParts
        Err(AppError::InternalError("Not implemented yet".to_string()))
    }
}
