//! 媒体服务
//!
//! 基于 S3 (MinIO) 的 Presigned URL 上传流程
//! MinIO 完全模拟 AWS S3，使用标准的 presigned URL 直传

use aws_sdk_s3::operation::head_object::HeadObjectError;
use aws_sdk_s3::presigning::PresigningConfig;
use aws_sdk_s3::Client as S3Client;
use moka::future::Cache;
use std::time::Duration;
use tokio::time::{sleep, timeout};
use uuid::Uuid;

use crate::{
    dto::{
        request::{CompleteUploadRequest, PresignUploadRequest},
        response::{PresignUploadData, UploadMediaData},
    },
    error::{ApiError, Result},
    repositories::file_repo::FileRepository,
};

const SIGNED_URL_CACHE_MAX_TTL: Duration = Duration::from_secs(300);
const SIGNED_URL_CACHE_FAILURE_TTL: Duration = Duration::from_secs(30);
const SIGNED_URL_FAILURE_PREFIX: &str = "__FAILED__:";
const SIGNED_URL_GENERATION_TIMEOUT: Duration = Duration::from_millis(1200);

#[derive(Clone)]
pub struct MediaService {
    s3_client: S3Client,
    bucket: String,
    s3_public_base: String,
    file_repo: FileRepository,
    redis: redis::aio::ConnectionManager,
    signed_url_cache: Cache<String, String>,
}

impl MediaService {
    pub fn new(
        file_repo: FileRepository,
        s3_client: S3Client,
        bucket: String,
        s3_public_base: String,
        redis: redis::aio::ConnectionManager,
        signed_url_cache: Cache<String, String>,
    ) -> Self {
        Self {
            file_repo,
            s3_client,
            bucket,
            s3_public_base,
            redis,
            signed_url_cache,
        }
    }

    fn signed_cache_key(key: &str, expires_in: Duration) -> String {
        format!("{}:{}", key, expires_in.as_secs())
    }

    fn build_public_url(&self, presigned_url: &str, path_and_query: &str) -> String {
        let base = self.s3_public_base.trim();
        if base.is_empty() || base.eq_ignore_ascii_case("presigned") {
            return presigned_url.to_string();
        }
        let base_trimmed = base.trim_end_matches('/');
        let bucket = self.bucket.trim_matches('/');
        let bucket_prefix = format!("/{}", bucket);
        let mut normalized_path = path_and_query.to_string();
        // 若 base 已包含 bucket（如 /s3/hotspot），避免拼接出 /hotspot/hotspot/...
        if base_trimmed.ends_with(&bucket_prefix) {
            if normalized_path == bucket_prefix {
                normalized_path = "/".to_string();
            } else if normalized_path.starts_with(&(bucket_prefix.clone() + "/")) {
                normalized_path = normalized_path[bucket_prefix.len()..].to_string();
            }
        }
        format!("{}{}", base_trimmed, normalized_path)
    }

    /// 申请上传（返回 S3 Presigned URL）
    pub async fn presign_upload(
        &self,
        user_id: i64,
        req: PresignUploadRequest,
    ) -> Result<PresignUploadData> {
        // 调试：记录接收到的 category
        tracing::info!(
            "presign_upload: user_id={}, category={:?}, file_name={}, mime_type={}, file_size={}",
            user_id,
            req.category,
            req.file_name,
            req.mime_type,
            req.file_size
        );

        // 按分类验证 MIME 类型
        if !req.category.validate_mime_type(&req.mime_type) {
            return Err(ApiError::InvalidFormat(format!(
                "该分类不支持此 MIME 类型: {} (允许: {:?})",
                req.mime_type,
                req.category.allowed_mime_types()
            )));
        }

        let file_size = u64::try_from(req.file_size)
            .map_err(|_| ApiError::InvalidFormat("文件大小无效".to_string()))?;

        // 按分类验证文件大小
        if !req.category.validate_file_size(file_size) {
            return Err(ApiError::PayloadTooLarge(format!(
                "文件大小超出限制: {} 字节 (最大: {} 字节)",
                req.file_size,
                req.category.max_file_size()
            )));
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
            .map_err(|e| ApiError::InternalError(format!("预签名配置构建失败: {}", e)))?;

        let presigned_request = self
            .s3_client
            .put_object()
            .bucket(&self.bucket)
            .key(&s3_key)
            .content_type(&req.mime_type)
            .presigned(presigned_config)
            .await
            .map_err(|e| ApiError::InternalError(format!("预签名 URL 生成失败: {}", e)))?;

        // 提取 presigned URL 并转换为客户端可访问的 URL
        let upload_url = presigned_request.uri().to_string();

        // 解析 URI，提取 path_and_query
        let uri = upload_url.parse::<http::Uri>().map_err(|e| {
            ApiError::InternalError(format!("Failed to parse presigned URI: {}", e))
        })?;
        let path_and_query = uri.path_and_query().map(|pq| pq.as_str()).unwrap_or("");

        let upload_url = self.build_public_url(&upload_url, path_and_query);

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
            .map_err(|e| ApiError::InternalError(format!("上传会话保存失败: {}", e)))?;

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
                    ApiError::FileNotFound("File not found in storage".to_string())
                }
                err => ApiError::InternalError(format!("存储服务错误: {}", err)),
            })?;

        let size = head_output
            .content_length
            .ok_or_else(|| ApiError::InternalError("File size missing".to_string()))?;
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
            .map_err(|e| ApiError::InternalError(format!("获取上传会话失败: {}", e)))?;

        tracing::debug!(
            "complete_upload: redis_key={}, session_json={:?}",
            redis_key,
            session_json
        );

        let session_str = session_json.ok_or_else(|| {
            ApiError::FileNotFound("Upload session not found or expired".to_string())
        })?;

        let session_data: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| ApiError::InternalError(format!("Failed to parse session data: {}", e)))?;

        let category = session_data["category"]
            .as_str()
            .ok_or_else(|| ApiError::InternalError("Category missing in session".to_string()))?;

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

        if let Err(err) = self.invalidate_signed_url(&storage_path).await {
            tracing::debug!(target: "media_service", error = %err, "failed to invalidate signed url cache after upload");
        }

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

    pub async fn invalidate_signed_url(&self, key: &str) -> Result<()> {
        self.invalidate_signed_url_with_expiry(key, Duration::from_secs(604800))
            .await
    }

    pub async fn invalidate_signed_url_with_expiry(
        &self,
        key: &str,
        expires_in: Duration,
    ) -> Result<()> {
        if key.starts_with("http://") || key.starts_with("https://") {
            return Err(ApiError::BadRequest("无效的存储 key".to_string()));
        }
        let cleaned_key = key.trim_start_matches('/');
        let cache_key = Self::signed_cache_key(cleaned_key, expires_in);
        self.signed_url_cache.invalidate(&cache_key).await;
        Ok(())
    }

    /// 生成下载/访问用的 Presigned URL (GET)
    pub async fn sign_url(&self, key: &str) -> Result<String> {
        self.sign_url_with_expiry(key, Duration::from_secs(604800))
            .await
    }

    fn is_failed_entry(value: &str) -> bool {
        value.starts_with(SIGNED_URL_FAILURE_PREFIX)
    }

    async fn cache_success_signed_url(&self, cache_key: &str, url: &str, expires_in: Duration) {
        let ttl = expires_in.min(SIGNED_URL_CACHE_MAX_TTL);
        self.insert_with_ttl(cache_key, url.to_string(), ttl).await;
    }

    async fn cache_failed_signed_url(&self, cache_key: &str) {
        self.insert_with_ttl(
            cache_key,
            format!("{}{}", SIGNED_URL_FAILURE_PREFIX, cache_key),
            SIGNED_URL_CACHE_FAILURE_TTL,
        )
        .await;
    }

    async fn insert_with_ttl(&self, cache_key: &str, value: String, ttl: Duration) {
        let cache = self.signed_url_cache.clone();
        let key = cache_key.to_string();
        cache.insert(key.clone(), value).await;
        tokio::spawn(async move {
            sleep(ttl).await;
            cache.invalidate(&key).await;
        });
    }

    /// 生成下载/访问用的 Presigned URL (GET) - 可配置过期时间
    pub async fn sign_url_with_expiry(&self, key: &str, expires_in: Duration) -> Result<String> {
        // 严格模式：只接受“对象存储 key”（例如 uploads/{user_id}/{uuid}.ext）
        // 不接受完整 URL，也不做旧格式兼容。
        if key.starts_with("http://") || key.starts_with("https://") {
            return Err(ApiError::BadRequest("无效的存储 key".to_string()));
        }

        // 去掉前导斜杠（防止误传 "/uploads/..."）
        let cleaned_key = key.trim_start_matches('/');
        let cache_key = Self::signed_cache_key(cleaned_key, expires_in);
        if let Some(cached) = self.signed_url_cache.get(&cache_key).await {
            if Self::is_failed_entry(&cached) {
                tracing::debug!(key, "reuse cached failed signed url entry");
                return Err(ApiError::InternalError(
                    "暂时无法生成下载链接，请稍后再试".to_string(),
                ));
            }
            return Ok(cached);
        }

        let presigned_config = PresigningConfig::builder()
            .expires_in(expires_in)
            .build()
            .map_err(|e| ApiError::InternalError(format!("预签名配置构建失败: {}", e)))?;

        let presigned_req = match timeout(
            SIGNED_URL_GENERATION_TIMEOUT,
            self.s3_client
                .get_object()
                .bucket(&self.bucket)
                .key(cleaned_key) // 使用清理后的 key
                .presigned(presigned_config),
        )
        .await
        {
            Ok(Ok(req)) => req,
            Ok(Err(e)) => {
                self.cache_failed_signed_url(&cache_key).await;
                return Err(ApiError::InternalError(format!("下载 URL 签名失败: {}", e)));
            }
            Err(_) => {
                self.cache_failed_signed_url(&cache_key).await;
                return Err(ApiError::InternalError(
                    "下载 URL 签名超时，请稍后重试".to_string(),
                ));
            }
        };

        let presigned_url = presigned_req.uri().to_string();
        let uri_parsed = presigned_url.parse::<http::Uri>().map_err(|e| {
            ApiError::InternalError(format!("Failed to parse presigned URI: {}", e))
        })?;
        let path_and_query = uri_parsed
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("");

        let download_url = self.build_public_url(&presigned_url, path_and_query);

        tracing::debug!(
            "sign_url_with_expiry: key={}, s3_public_base={}, path_and_query={}, final_url={}",
            key,
            self.s3_public_base,
            path_and_query,
            download_url
        );

        self.cache_success_signed_url(&cache_key, &download_url, expires_in)
            .await;

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
                let normalized_key = s.trim().trim_start_matches('/');
                if normalized_key.starts_with("uploads/") {
                    match self.sign_url(normalized_key).await {
                        Ok(url) => Some(url),
                        Err(e) => {
                            tracing::warn!("Failed to sign url for path {}: {}", normalized_key, e);
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
