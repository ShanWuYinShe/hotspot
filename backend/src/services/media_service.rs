use redis::aio::ConnectionManager as RedisConnectionManager;
use redis::AsyncCommands;
use sha2::{Digest, Sha256};
use sqlx::PgPool;
use std::path::{Path, PathBuf};
use uuid::Uuid;

use crate::{
    constants::UPLOAD_SESSION_HOURS,
    dto::{
        request::{
            AbortUploadRequest, CompleteUploadRequest, InitiateUploadRequest, UploadStatusRequest,
        },
        response::{
            InitiateUploadData, UploadChunkData, UploadMediaData, UploadPartUrlData,
            UploadStatusData,
        },
    },
    error::{AppError, Result},
    repositories::file_repo::FileRepository,
};

#[derive(Clone)]
pub struct MediaService {
    redis: RedisConnectionManager,
    file_repo: FileRepository,
    upload_dir: PathBuf,
    max_file_size: usize,
}

impl MediaService {
    pub fn new(db: &PgPool, redis: &RedisConnectionManager) -> Self {
        // 从环境变量读取配置，或使用默认值
        let upload_dir = std::env::var("UPLOAD_DIR").unwrap_or_else(|_| "./uploads".to_string());
        let max_file_size = std::env::var("MAX_UPLOAD_SIZE")
            .unwrap_or_else(|_| "10485760".to_string()) // 10MB
            .parse()
            .unwrap_or(10485760);

        Self {
            file_repo: FileRepository::new(db.clone()),
            redis: redis.clone(),
            upload_dir: PathBuf::from(upload_dir),
            max_file_size,
        }
    }

    /// 验证文件类型
    fn validate_file_type(&self, filename: &str, allowed_types: &[&str]) -> Result<String> {
        let extension = filename
            .rsplit('.')
            .next()
            .ok_or_else(|| AppError::BadRequest("文件没有扩展名".to_string()))?
            .to_lowercase();

        if !allowed_types.contains(&extension.as_str()) {
            return Err(AppError::BadRequest(format!(
                "不支持的文件类型: {}",
                extension
            )));
        }

        Ok(extension)
    }

    /// 生成唯一文件名
    fn generate_filename(&self, extension: &str) -> String {
        format!("{}.{}", Uuid::new_v4(), extension)
    }

    /// 获取文件保存路径
    fn get_file_path(&self, filename: &str, subdir: &str) -> PathBuf {
        self.upload_dir.join(subdir).join(filename)
    }

    /// 确保目录存在
    fn ensure_directory(&self, path: &Path) -> Result<()> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| AppError::InternalError(format!("创建目录失败: {}", e)))?;
        }
        Ok(())
    }

    /// 上传文件（≤100MB）
    pub async fn upload_file(
        &self,
        user_id: i64,
        filename: &str,
        category: &str,
        data: Vec<u8>,
    ) -> Result<UploadMediaData> {
        // 验证文件大小
        if data.len() > self.max_file_size {
            return Err(AppError::BadRequest(format!(
                "文件大小超过限制: {} bytes",
                self.max_file_size
            )));
        }

        // 验证文件类型（支持图片和文档）
        let allowed_types = [
            "jpg", "jpeg", "png", "gif", "webp", "pdf", "doc", "docx", "xls", "xlsx", "ppt",
            "pptx", "txt", "zip", "rar", "mp4", "mov", "mp3", "aac", "ogg",
        ];
        let extension = self.validate_file_type(filename, &allowed_types)?;

        // 计算文件 checksum 用于去重
        let mut hasher = Sha256::new();
        hasher.update(&data);
        let checksum = format!("{:x}", hasher.finalize());

        let file_size = data.len() as i64;
        let mime_type = self.get_mime_type(&extension);

        // 检查是否有相同内容的文件（物理文件去重）
        // 注意：无论是否复用物理文件，每次上传都会生成新的 file_id 和 url
        let (storage_path, need_save) =
            if let Some(existing_file) = self.file_repo.find_any_by_checksum(&checksum).await? {
                // 物理文件已存在，复用 storage_path，跳过磁盘写入
                (existing_file.storage_path, false)
            } else {
                // 全新文件，生成新路径，需要保存到磁盘
                let new_filename = self.generate_filename(&extension);
                let storage_path = format!("files/{}", new_filename);
                (storage_path, true)
            };

        // 只有全新文件才需要保存到磁盘
        if need_save {
            let file_path = self.get_file_path(&storage_path.replace("files/", ""), "files");
            self.ensure_directory(&file_path)?;
            tokio::fs::write(&file_path, &data)
                .await
                .map_err(|e| AppError::InternalError(format!("保存文件失败: {}", e)))?;
        }

        // 每次上传都创建新的 file_id 记录
        // file_id 是给客户端的标识符，url 是实际物理文件路径，两者无关联
        // 客户端发送消息时只传 file_id，服务端查表转换为 url
        let file_id = Uuid::new_v4();
        let url = format!("/uploads/{}", storage_path);

        let file_record = self
            .file_repo
            .create_file_with_id(
                file_id,
                user_id,
                category,
                Some(filename),
                &storage_path,
                &url,
                None, // thumbnail_url
                &mime_type,
                file_size,
                None, // width
                None, // height
                None, // duration
                Some(&checksum),
            )
            .await?;

        Ok(UploadMediaData {
            file_id: file_record.file_id.to_string(),
            thumbnail_url: None,
            file_size,
            duration: None,
        })
    }

    /// 初始化大文件上传（>100MB）
    pub async fn initiate_upload(
        &self,
        user_id: i64,
        req: &InitiateUploadRequest,
    ) -> Result<InitiateUploadData> {
        // 生成上传会话ID
        let upload_id = Uuid::new_v4().to_string();

        // 计算分块数量
        let chunk_size = req.chunk_size.unwrap_or(5 * 1024 * 1024) as i64; // 默认5MB
        let chunk_count = ((req.file_size as f64) / (chunk_size as f64)).ceil() as i32;

        // 上传会话过期时间（用于客户端展示/恢复）
        let expires_at =
            (chrono::Utc::now() + chrono::Duration::hours(UPLOAD_SESSION_HOURS)).to_rfc3339();

        // 分块上传 URL 列表
        // - 分块很大时一次性返回会导致响应体过大；因此做一个上限，超过则返回空数组，客户端用 upload_part_url 按需获取
        const MAX_URLS_IN_INIT_RESPONSE: i32 = 200;
        let upload_urls: Vec<String> = if chunk_count <= MAX_URLS_IN_INIT_RESPONSE {
            (0..chunk_count)
                .map(|i| {
                    format!(
                        "/api/v1/protected/media/upload_chunk?upload_id={}&part_number={}",
                        upload_id,
                        i + 1
                    )
                })
                .collect()
        } else {
            vec![]
        };

        // 将上传会话信息存储到Redis
        let session_key = format!("upload_session:{}", upload_id);
        let session_data = serde_json::json!({
            "user_id": user_id,
            "file_name": req.file_name,
            "file_size": req.file_size,
            "mime_type": req.mime_type,
            "category": req.category,
            "chunk_size": chunk_size,
            "chunk_count": chunk_count,
            "uploaded_parts": Vec::<i32>::new(),
            "created_at": chrono::Utc::now().to_rfc3339(),
        });

        let mut redis = self.redis.clone();
        let _: () = redis
            .set_ex(&session_key, session_data.to_string(), 86400)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(InitiateUploadData {
            upload_id,
            upload_target: "backend".to_string(),
            chunk_size,
            chunk_count,
            upload_urls,
            expires_at,
        })
    }

    /// 按需获取分块上传 URL（用于分块数量较大时，避免 initiate_upload 返回超大响应体）
    pub async fn get_upload_part_url(
        &self,
        user_id: i64,
        upload_id: &str,
        part_number: i32,
    ) -> Result<UploadPartUrlData> {
        let session_key = format!("upload_session:{}", upload_id);
        let mut redis = self.redis.clone();

        let session_str: Option<String> = redis
            .get(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        let session_str =
            session_str.ok_or_else(|| AppError::NotFound("上传会话不存在或已过期".to_string()))?;
        let session: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| AppError::InternalError(format!("解析会话数据失败: {}", e)))?;

        if session["user_id"].as_i64() != Some(user_id) {
            return Err(AppError::Forbidden("无权访问此上传会话".to_string()));
        }

        let chunk_count = session["chunk_count"].as_i64().unwrap_or(0) as i32;
        if part_number < 1 || part_number > chunk_count {
            return Err(AppError::BadRequest("part_number 超出范围".to_string()));
        }

        // 过期时间与 upload_status 的计算保持一致：created_at + UPLOAD_SESSION_HOURS
        let created_at = session["created_at"].as_str().unwrap_or("").to_string();
        let expires_at = if let Ok(dt) = chrono::DateTime::parse_from_rfc3339(&created_at) {
            (dt + chrono::Duration::hours(UPLOAD_SESSION_HOURS)).to_rfc3339()
        } else {
            (chrono::Utc::now() + chrono::Duration::hours(UPLOAD_SESSION_HOURS)).to_rfc3339()
        };

        let upload_url = format!(
            "/api/v1/protected/media/upload_chunk?upload_id={}&part_number={}",
            upload_id, part_number
        );

        Ok(UploadPartUrlData {
            upload_id: upload_id.to_string(),
            part_number,
            upload_url,
            expires_at,
        })
    }

    /// 上传分块
    pub async fn upload_chunk(
        &self,
        user_id: i64,
        upload_id: &str,
        part_number: i32,
        data: Vec<u8>,
    ) -> Result<UploadChunkData> {
        // 从Redis获取上传会话信息
        let session_key = format!("upload_session:{}", upload_id);
        let mut redis = self.redis.clone();

        let session_str: Option<String> = redis
            .get(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        let session_str =
            session_str.ok_or_else(|| AppError::NotFound("上传会话不存在或已过期".to_string()))?;
        let mut session: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| AppError::InternalError(format!("解析会话数据失败: {}", e)))?;

        // 验证用户ID
        if session["user_id"].as_i64() != Some(user_id) {
            return Err(AppError::Forbidden("无权访问此上传会话".to_string()));
        }

        // 保存分块文件
        let chunk_dir = self.upload_dir.join("chunks").join(upload_id);
        std::fs::create_dir_all(&chunk_dir)
            .map_err(|e| AppError::InternalError(format!("创建分块目录失败: {}", e)))?;

        let chunk_path = chunk_dir.join(format!("part_{}", part_number));
        tokio::fs::write(&chunk_path, &data)
            .await
            .map_err(|e| AppError::InternalError(format!("保存分块失败: {}", e)))?;

        // 计算ETag（使用SHA256）
        let mut hasher = Sha256::new();
        hasher.update(&data);
        let etag = format!("{:x}", hasher.finalize());

        // 更新已上传分块列表
        if let Some(parts) = session["uploaded_parts"].as_array_mut() {
            if !parts.iter().any(|p| p.as_i64() == Some(part_number as i64)) {
                parts.push(serde_json::json!(part_number));
            }
        }

        // 保存更新后的会话
        let _: () = redis
            .set_ex(&session_key, session.to_string(), 86400)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(UploadChunkData { part_number, etag })
    }

    /// 完成大文件上传
    pub async fn complete_upload(
        &self,
        user_id: i64,
        req: &CompleteUploadRequest,
    ) -> Result<UploadMediaData> {
        // 从Redis获取上传会话信息
        let session_key = format!("upload_session:{}", req.upload_id);
        let mut redis = self.redis.clone();

        let session_str: Option<String> = redis
            .get(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        let session_str =
            session_str.ok_or_else(|| AppError::NotFound("上传会话不存在或已过期".to_string()))?;
        let session: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| AppError::InternalError(format!("解析会话数据失败: {}", e)))?;

        // 验证用户ID
        if session["user_id"].as_i64() != Some(user_id) {
            return Err(AppError::Forbidden("无权访问此上传会话".to_string()));
        }

        let chunk_count = session["chunk_count"].as_i64().unwrap_or(0) as i32;
        let file_name = session["file_name"].as_str().unwrap_or("file");
        let mime_type = session["mime_type"]
            .as_str()
            .unwrap_or("application/octet-stream")
            .to_string();
        let category = session["category"]
            .as_str()
            .unwrap_or("chat_file")
            .to_string();

        // 验证所有分块已上传
        if req.parts.len() != chunk_count as usize {
            return Err(AppError::BadRequest(format!(
                "分块数量不匹配: 期望 {}, 实际 {}",
                chunk_count,
                req.parts.len()
            )));
        }

        // 合并分块文件
        let chunk_dir = self.upload_dir.join("chunks").join(&req.upload_id);
        let extension = file_name.rsplit('.').next().unwrap_or("bin");
        let new_filename = self.generate_filename(extension);
        let storage_path = format!("files/{}", new_filename);
        let final_path = self.get_file_path(&new_filename, "files");

        self.ensure_directory(&final_path)?;

        let mut final_file = tokio::fs::File::create(&final_path)
            .await
            .map_err(|e| AppError::InternalError(format!("创建文件失败: {}", e)))?;

        let mut total_size: i64 = 0;
        let mut hasher = Sha256::new();

        for part in &req.parts {
            let chunk_path = chunk_dir.join(format!("part_{}", part.part_number));
            let chunk_data = tokio::fs::read(&chunk_path)
                .await
                .map_err(|e| AppError::InternalError(format!("读取分块失败: {}", e)))?;

            // 校验分块 ETag（upload_chunk 返回的是 SHA256 hex）
            let mut part_hasher = Sha256::new();
            part_hasher.update(&chunk_data);
            let expected_etag = format!("{:x}", part_hasher.finalize());
            if expected_etag != part.etag {
                return Err(AppError::BadRequest(format!(
                    "分块ETag不匹配: part_number={}",
                    part.part_number
                )));
            }

            total_size += chunk_data.len() as i64;
            hasher.update(&chunk_data);

            use tokio::io::AsyncWriteExt;
            final_file
                .write_all(&chunk_data)
                .await
                .map_err(|e| AppError::InternalError(format!("写入文件失败: {}", e)))?;
        }

        let checksum = format!("{:x}", hasher.finalize());

        // 检查是否有相同内容的文件（物理文件去重）
        // 如果有，删除刚合并的文件，复用已存在的 storage_path
        let final_storage_path =
            if let Some(existing_file) = self.file_repo.find_any_by_checksum(&checksum).await? {
                // 删除刚合并的文件（因为已有相同内容的文件）
                let _ = tokio::fs::remove_file(&final_path).await;
                existing_file.storage_path
            } else {
                storage_path
            };

        // 清理临时分块
        let _ = tokio::fs::remove_dir_all(&chunk_dir).await;

        // 删除Redis会话
        let _: () = redis
            .del(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 每次上传都生成新的 file_id
        // file_id 是标识符，url 是实际物理文件路径，两者无关联
        let file_id = Uuid::new_v4();
        let url = format!("/uploads/{}", final_storage_path);

        // 持久化到数据库
        let file_record = self
            .file_repo
            .create_file_with_id(
                file_id,
                user_id,
                &category,
                Some(file_name),
                &final_storage_path,
                &url,
                None, // thumbnail_url
                &mime_type,
                total_size,
                None, // width
                None, // height
                None, // duration
                Some(&checksum),
            )
            .await?;

        Ok(UploadMediaData {
            file_id: file_record.file_id.to_string(),
            thumbnail_url: None,
            file_size: total_size,
            duration: None,
        })
    }

    /// 查询上传状态
    pub async fn get_upload_status(
        &self,
        user_id: i64,
        req: &UploadStatusRequest,
    ) -> Result<UploadStatusData> {
        // 从Redis获取上传会话信息
        let session_key = format!("upload_session:{}", req.upload_id);
        let mut redis = self.redis.clone();

        let session_str: Option<String> = redis
            .get(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        let session_str =
            session_str.ok_or_else(|| AppError::NotFound("上传会话不存在或已过期".to_string()))?;
        let session: serde_json::Value = serde_json::from_str(&session_str)
            .map_err(|e| AppError::InternalError(format!("解析会话数据失败: {}", e)))?;

        // 验证用户ID
        if session["user_id"].as_i64() != Some(user_id) {
            return Err(AppError::Forbidden("无权访问此上传会话".to_string()));
        }

        let uploaded_parts: Vec<i32> = session["uploaded_parts"]
            .as_array()
            .map(|arr| {
                arr.iter()
                    .filter_map(|v| v.as_i64().map(|n| n as i32))
                    .collect()
            })
            .unwrap_or_default();

        let chunk_count = session["chunk_count"].as_i64().unwrap_or(0) as i32;
        let created_at = session["created_at"].as_str().unwrap_or("").to_string();

        // 计算过期时间
        let expires_at = if let Ok(dt) = chrono::DateTime::parse_from_rfc3339(&created_at) {
            (dt + chrono::Duration::hours(UPLOAD_SESSION_HOURS)).to_rfc3339()
        } else {
            (chrono::Utc::now() + chrono::Duration::hours(UPLOAD_SESSION_HOURS)).to_rfc3339()
        };

        // 判断状态
        let status = if uploaded_parts.len() == chunk_count as usize {
            "completed".to_string()
        } else {
            "in_progress".to_string()
        };

        Ok(UploadStatusData {
            upload_id: req.upload_id.clone(),
            status,
            uploaded_parts,
            total_parts: chunk_count,
            created_at,
            expires_at,
        })
    }

    /// 取消大文件上传
    pub async fn abort_upload(&self, user_id: i64, req: &AbortUploadRequest) -> Result<()> {
        // 从Redis获取上传会话信息
        let session_key = format!("upload_session:{}", req.upload_id);
        let mut redis = self.redis.clone();

        let session_str: Option<String> = redis
            .get(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        if let Some(session_str) = session_str {
            let session: serde_json::Value = serde_json::from_str(&session_str)
                .map_err(|e| AppError::InternalError(format!("解析会话数据失败: {}", e)))?;

            // 验证用户ID
            if session["user_id"].as_i64() != Some(user_id) {
                return Err(AppError::Forbidden("无权访问此上传会话".to_string()));
            }
        }

        // 清理已上传的分块文件
        let chunk_dir = self.upload_dir.join("chunks").join(&req.upload_id);
        let _ = tokio::fs::remove_dir_all(&chunk_dir).await;

        // 从Redis删除上传会话信息
        let _: () = redis
            .del(&session_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        tracing::debug!("取消上传: upload_id={}", req.upload_id);

        Ok(())
    }

    /// 获取MIME类型
    fn get_mime_type(&self, extension: &str) -> String {
        match extension {
            "jpg" | "jpeg" => "image/jpeg",
            "png" => "image/png",
            "gif" => "image/gif",
            "webp" => "image/webp",
            "pdf" => "application/pdf",
            "doc" => "application/msword",
            "docx" => "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            "xls" => "application/vnd.ms-excel",
            "xlsx" => "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            "ppt" => "application/vnd.ms-powerpoint",
            "pptx" => "application/vnd.openxmlformats-officedocument.presentationml.presentation",
            "txt" => "text/plain",
            "zip" => "application/zip",
            "rar" => "application/x-rar-compressed",
            _ => "application/octet-stream",
        }
        .to_string()
    }
}
