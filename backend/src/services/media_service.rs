use redis::Client as RedisClient;
use sqlx::PgPool;
use std::path::PathBuf;
use uuid::Uuid;

use crate::{
    dto::response::UploadResponse,
    error::{AppError, Result},
};

pub struct MediaService<'a> {
    #[allow(dead_code)]
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisClient,
    upload_dir: PathBuf,
    max_file_size: usize,
}

impl<'a> MediaService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisClient) -> Self {
        // 从环境变量读取配置，或使用默认值
        let upload_dir = std::env::var("UPLOAD_DIR")
            .unwrap_or_else(|_| "./uploads".to_string());
        let max_file_size = std::env::var("MAX_UPLOAD_SIZE")
            .unwrap_or_else(|_| "10485760".to_string()) // 10MB
            .parse()
            .unwrap_or(10485760);

        Self {
            db,
            redis,
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
    fn ensure_directory(&self, path: &PathBuf) -> Result<()> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| AppError::InternalError(format!("创建目录失败: {}", e)))?;
        }
        Ok(())
    }

    /// 上传图片
    pub async fn upload_image(
        &self,
        _user_id: i64,
        filename: &str,
        data: Vec<u8>,
    ) -> Result<UploadResponse> {
        // 验证文件大小
        if data.len() > self.max_file_size {
            return Err(AppError::BadRequest(format!(
                "文件大小超过限制: {} bytes",
                self.max_file_size
            )));
        }

        // 验证文件类型
        let allowed_types = ["jpg", "jpeg", "png", "gif", "webp"];
        let extension = self.validate_file_type(filename, &allowed_types)?;

        // 生成新文件名
        let new_filename = self.generate_filename(&extension);
        let file_path = self.get_file_path(&new_filename, "images");

        // 确保目录存在
        self.ensure_directory(&file_path)?;

        // 保存文件
        tokio::fs::write(&file_path, data)
            .await
            .map_err(|e| AppError::InternalError(format!("保存文件失败: {}", e)))?;

        // 生成URL（这里简化处理，实际应该使用CDN或对象存储）
        let url = format!("/uploads/images/{}", new_filename);

        // TODO: 保存文件元数据到数据库
        // TODO: 生成缩略图

        Ok(UploadResponse {
            url,
            filename: new_filename,
            size: data.len() as i64,
            mime_type: format!("image/{}", extension),
        })
    }

    /// 上传文件
    pub async fn upload_file(
        &self,
        _user_id: i64,
        filename: &str,
        data: Vec<u8>,
    ) -> Result<UploadResponse> {
        // 验证文件大小
        if data.len() > self.max_file_size {
            return Err(AppError::BadRequest(format!(
                "文件大小超过限制: {} bytes",
                self.max_file_size
            )));
        }

        // 验证文件类型
        let allowed_types = [
            "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "zip", "rar",
        ];
        let extension = self.validate_file_type(filename, &allowed_types)?;

        // 生成新文件名
        let new_filename = self.generate_filename(&extension);
        let file_path = self.get_file_path(&new_filename, "files");

        // 确保目录存在
        self.ensure_directory(&file_path)?;

        // 保存文件
        tokio::fs::write(&file_path, data)
            .await
            .map_err(|e| AppError::InternalError(format!("保存文件失败: {}", e)))?;

        // 生成URL
        let url = format!("/uploads/files/{}", new_filename);

        // TODO: 保存文件元数据到数据库

        Ok(UploadResponse {
            url,
            filename: new_filename,
            size: data.len() as i64,
            mime_type: self.get_mime_type(&extension),
        })
    }

    /// 获取上传Token（用于客户端直传）
    pub async fn get_upload_token(&self, user_id: i64) -> Result<String> {
        // 生成临时上传Token（有效期10分钟）
        let token = Uuid::new_v4().to_string();
        
        // TODO: 将Token存储到Redis，设置过期时间
        // redis.setex(format!("upload_token:{}", token), 600, user_id)

        tracing::info!("生成上传Token: user_id={}, token={}", user_id, token);

        Ok(token)
    }

    /// 获取MIME类型
    fn get_mime_type(&self, extension: &str) -> String {
        match extension {
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

