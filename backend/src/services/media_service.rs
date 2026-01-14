//! 媒体服务
//!
//! 基于 S3 (MinIO) 的 Presigned URL 上传流程
//! MinIO 完全模拟 AWS S3，使用标准的 presigned URL 直传

use aws_sdk_s3::operation::head_object::HeadObjectError;
use aws_sdk_s3::presigning::PresigningConfig;
use aws_sdk_s3::Client as S3Client;
use sqlx::PgPool;
use std::time::Duration;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CheckChunkedUploadRequest, CompleteUploadRequest, FinishChunkedUploadRequest,
            InitChunkedUploadRequest, PresignUploadRequest,
        },
        response::{InitiateUploadData, PresignUploadData, UploadMediaData},
    },
    error::{AppError, Result},
    repositories::file_repo::FileRepository,
};

#[derive(Clone)]
pub struct MediaService {
    s3_client: S3Client,
    bucket: String,
    s3_public_base: String,
    file_repo: FileRepository,
    redis: redis::aio::ConnectionManager,
}

impl MediaService {
    pub fn new(
        db: &PgPool,
        s3_client: S3Client,
        bucket: String,
        s3_public_base: String,
        redis: redis::aio::ConnectionManager,
    ) -> Self {
        Self {
            file_repo: FileRepository::new(db.clone()),
            s3_client,
            bucket,
            s3_public_base,
            redis,
        }
    }

    /// 申请上传（返回 S3 Presigned URL）
    pub async fn presign_upload(
        &self,
        user_id: i64,
        req: PresignUploadRequest,
    ) -> Result<PresignUploadData> {
        // 调试：记录接收到的 category
        tracing::info!(
            "presign_upload: user_id={}, category={:?}, file_name={}",
            user_id,
            req.category,
            req.file_name
        );

        // 简单校验
        if req.file_size > 100 * 1024 * 1024 {
            return Err(AppError::BadRequest("File too large".to_string()));
        }

        // Suppress unused md5 warning
        let _ = req.md5;

        // 生成 upload_id (UUID)
        let upload_id = Uuid::new_v4();

        let extension = req
            .file_name
            .rsplit('.')
            .next()
            .unwrap_or("bin")
            .to_lowercase();

        // S3 Key: uploads/{user_id}/{uuid}.{ext}
        let s3_key = format!("uploads/{}/{}.{}", user_id, upload_id, extension);

        // 生成 S3 Presigned URL（15分钟有效）
        let presigned_config = PresigningConfig::builder()
            .expires_in(Duration::from_secs(900)) // 15分钟
            .build()
            .map_err(|e| {
                AppError::InternalError(format!("Failed to build presigning config: {}", e))
            })?;

        let presigned_request = self
            .s3_client
            .put_object()
            .bucket(&self.bucket)
            .key(&s3_key)
            .content_type(&req.mime_type)
            .presigned(presigned_config)
            .await
            .map_err(|e| {
                AppError::InternalError(format!("Failed to generate presigned URL: {}", e))
            })?;

        // 提取 presigned URL 并转换为客户端可访问的 URL
        let upload_url = presigned_request.uri().to_string();

        // 解析 URI，提取 path_and_query
        let uri = upload_url.parse::<http::Uri>().map_err(|e| {
            AppError::InternalError(format!("Failed to parse presigned URI: {}", e))
        })?;
        let path_and_query = uri.path_and_query().map(|pq| pq.as_str()).unwrap_or("");

        // 简单拼接 s3_public_base 和 path_and_query
        // AWS SDK 生成的 presigned URL 已经是完整可用的，直接使用即可
        let public_base_trimmed = self.s3_public_base.trim_end_matches('/');
        let upload_url = format!("{}{}", public_base_trimmed, path_and_query);

        tracing::debug!(
            "presign_upload: s3_public_base={}, path_and_query={}, final_url={}",
            self.s3_public_base,
            path_and_query,
            upload_url
        );

        let expires_at = (chrono::Utc::now() + chrono::Duration::minutes(15)).to_rfc3339();

        // 构造 headers map（presigned URL 已包含签名，Content-Type 可能需要在请求头中）
        let mut headers = std::collections::HashMap::new();
        headers.insert("Content-Type".to_string(), req.mime_type.clone());

        // 保存上传会话信息到 Redis（15分钟过期）
        let redis_key = format!("upload:pending:{}", s3_key);
        let session_data = serde_json::json!({
            "user_id": user_id,
            "s3_key": s3_key,
            "mime_type": req.mime_type,
            "category": req.category,
            "file_name": req.file_name,
            "file_size": req.file_size,
            "created_at": chrono::Utc::now().to_rfc3339(),
        });

        tracing::debug!(
            "presign_upload: saving to Redis key={}, data={}",
            redis_key,
            session_data
        );

        let mut redis_conn = self.redis.clone();
        let _: () = redis::cmd("SETEX")
            .arg(&redis_key)
            .arg(900) // 15分钟过期
            .arg(session_data.to_string())
            .query_async(&mut redis_conn)
            .await
            .map_err(|e| {
                AppError::InternalError(format!("Failed to save upload session: {}", e))
            })?;

        Ok(PresignUploadData {
            upload_id: s3_key.clone(), // 返回 S3 Key 作为 upload_id，供 complete_upload 使用
            upload_url,
            headers,
            expires_at,
        })
    }

    /// 完成上传
    pub async fn complete_upload(
        &self,
        user_id: i64,
        req: CompleteUploadRequest,
    ) -> Result<UploadMediaData> {
        let key = req.upload_id; // upload_id 即为 S3 Key

        // 1. 检查 S3 对象是否存在
        let head_output = self
            .s3_client
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

        let size = head_output
            .content_length
            .ok_or_else(|| AppError::InternalError("File size missing".to_string()))?;
        let content_type = head_output.content_type.unwrap_or_default();
        let etag = head_output.e_tag.unwrap_or_default().replace("\"", "");

        // 2. 解析文件名/扩展名
        let file_name = key.rsplit('/').next().unwrap_or("file.bin").to_string();
        let _extension = file_name.rsplit('.').next().unwrap_or("").to_string();

        // 3. 生成 file_id (UUID)
        // key format: uploads/{user_id}/{uuid}.{ext}
        let uuid_part = file_name.rsplit('.').skip(1).next().unwrap_or(&file_name);
        let file_id = Uuid::parse_str(uuid_part).unwrap_or_else(|_| Uuid::new_v4());

        // 4. 创建 DB 记录
        let storage_path = key.clone();

        // 数据库 url 字段暂时存储 storage_path（Key），或者留空
        // 不再生成并存储带有效期的 Signed URL
        // 注意：数据库 url 字段目前是 NOT NULL，所以存入 storage_path 作为占位
        let url = storage_path.clone();

        // 从 Redis 恢复上传会话信息
        let redis_key = format!("upload:pending:{}", key);
        let mut redis_conn = self.redis.clone();

        let session_json: Option<String> = redis::cmd("GET")
            .arg(&redis_key)
            .query_async(&mut redis_conn)
            .await
            .map_err(|e| AppError::InternalError(format!("Failed to get upload session: {}", e)))?;

        tracing::debug!(
            "complete_upload: redis_key={}, session_json={:?}",
            redis_key,
            session_json
        );

        let session_str = session_json.ok_or_else(|| {
            AppError::FileNotFound("Upload session not found or expired".to_string())
        })?;

        let session_data: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| AppError::InternalError(format!("Failed to parse session data: {}", e)))?;

        let category = session_data["category"]
            .as_str()
            .ok_or_else(|| AppError::InternalError("Category missing in session".to_string()))?;

        tracing::info!(
            "complete_upload: user_id={}, upload_id={}, category={}",
            user_id,
            key,
            category
        );

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

        // 完成后删除 Redis 会话（避免重复使用）
        let _: () = redis::cmd("DEL")
            .arg(&redis_key)
            .query_async(&mut redis_conn)
            .await
            .unwrap_or_else(|e| {
                tracing::warn!("Failed to delete upload session: {}", e);
            });

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

    /// 生成下载/访问用的 Presigned URL (GET)
    pub async fn sign_url(&self, key: &str) -> Result<String> {
        // 如果是完整 URL (http/https 开头)，直接返回，不处理
        if key.starts_with("http://") || key.starts_with("https://") {
            return Ok(key.to_string());
        }

        // 清理 key：去掉前导斜杠和旧的 /media/ 前缀（兼容旧数据）
        let cleaned_key = key
            .trim_start_matches('/') // 去掉前导斜杠
            .trim_start_matches("media/"); // 去掉旧的 /media/ 前缀

        if key != cleaned_key {
            tracing::warn!(
                "sign_url: cleaned old format key: {} -> {}",
                key,
                cleaned_key
            );
        }

        // 设置较长的过期时间（例如 7 天），以减少客户端频繁刷新的需求
        // 注意：AWS S3 Presigned URL 最长有效期通常为 7 天
        let presigned_config = PresigningConfig::builder()
            .expires_in(Duration::from_secs(604800))
            .build()
            .map_err(|e| {
                AppError::InternalError(format!("Failed to build presigning config: {}", e))
            })?;

        let presigned_req = self
            .s3_client
            .get_object()
            .bucket(&self.bucket)
            .key(cleaned_key) // 使用清理后的 key
            .presigned(presigned_config)
            .await
            .map_err(|e| AppError::InternalError(format!("Failed to sign download URL: {}", e)))?;

        let uri_str = presigned_req.uri();
        let uri_parsed = uri_str.parse::<http::Uri>().map_err(|e| {
            AppError::InternalError(format!("Failed to parse presigned URI: {}", e))
        })?;
        let path_and_query = uri_parsed
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("");

        // 简单拼接 s3_public_base 和 path_and_query
        // AWS SDK 生成的 presigned URL 已经是完整可用的，直接使用即可
        let public_base_trimmed = self.s3_public_base.trim_end_matches('/');
        let download_url = format!("{}{}", public_base_trimmed, path_and_query);

        tracing::debug!(
            "sign_url: key={}, s3_public_base={}, path_and_query={}, final_url={}",
            key,
            self.s3_public_base,
            path_and_query,
            download_url
        );

        Ok(download_url)
    }

    /// 验证文件所有权并返回文件存储路径 (Key)
    /// 注意：返回的不是 Signed URL，而是存储路径，以便写入 DB。
    /// 读取时需调用 sign_url 生成签名链接。
    pub async fn validate_file(
        &self,
        file_id: Uuid,
        user_id: i64,
        expected_category: &str,
    ) -> Result<String> {
        let metadata = self
            .file_repo
            .validate_file(file_id, user_id, expected_category)
            .await?;

        // 返回存储路径 (Key)
        Ok(metadata.storage_path)
    }

    /// 验证消息文件并返回完整文件信息（url 字段包含存储路径）
    pub async fn validate_message_file(
        &self,
        file_id: Uuid,
        user_id: i64,
        expected_categories: &[&str],
    ) -> Result<crate::repositories::file_repo::UploadedFileData> {
        let file_data = self
            .file_repo
            .get_file_for_message(file_id, user_id, expected_categories)
            .await?;

        // 这里的 url 字段暂时存储 storage_path，发送消息时存入 content
        // 读取历史消息时再进行签名
        // file_data.url = file_data.storage_path.clone();

        Ok(file_data)
    }

    /// 如果输入看起来像存储路径（以 uploads/ 开头），则生成签名 URL；否则原样返回
    pub async fn sign_url_if_path(&self, path_or_url: Option<&str>) -> Option<String> {
        match path_or_url {
            Some(s) => {
                if s.starts_with("uploads/") {
                    match self.sign_url(s).await {
                        Ok(url) => Some(url),
                        Err(e) => {
                            tracing::warn!("Failed to sign url for path {}: {}", s, e);
                            Some(s.to_string()) // Fallback
                        }
                    }
                } else {
                    Some(s.to_string())
                }
            }
            None => None,
        }
    }

    /// 标记文件已使用
    pub async fn mark_as_used(&self, file_id: Uuid) -> Result<()> {
        self.file_repo.mark_as_used(file_id).await
    }
}
