/// 文件上传控制器
/// 
/// 处理头像上传、图片上传等文件上传功能
use axum::{
    extract::{Multipart, State},
    http::StatusCode,
    response::{Response, IntoResponse},
    Extension, Json,
};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use tokio::fs;
use tokio::io::AsyncWriteExt;
use tracing::{debug, error, info, warn};
use uuid::Uuid;

use crate::presentation::http::auth::CurrentUser;
use crate::presentation::http::controllers::ok_response;
use serde_json::json;

/// 文件上传控制器
#[derive(Clone)]
pub struct UploadController {
    /// 上传文件存储目录
    upload_dir: PathBuf,
    /// 最大文件大小（字节）
    max_file_size: usize,
    /// 允许的文件类型
    allowed_types: Vec<String>,
}

impl Default for UploadController {
    fn default() -> Self {
        Self::new()
    }
}

impl UploadController {
    /// 创建简单错误响应
    fn simple_error_response(message: &str) -> Response {
        let response = json!({
            "code": 1000,
            "msg": message,
            "data": null
        });
        (StatusCode::OK, Json(response)).into_response()
    }

    /// 创建新的上传控制器
    pub fn new() -> Self {
        Self {
            upload_dir: PathBuf::from("uploads"),
            max_file_size: 10 * 1024 * 1024, // 10MB
            allowed_types: vec![
                "image/jpeg".to_string(),
                "image/png".to_string(),
                "image/gif".to_string(),
                "image/webp".to_string(),
            ],
        }
    }
    
    /// 配置上传目录
    pub fn with_upload_dir(mut self, dir: PathBuf) -> Self {
        self.upload_dir = dir;
        self
    }
    
    /// 配置最大文件大小
    pub fn with_max_file_size(mut self, size: usize) -> Self {
        self.max_file_size = size;
        self
    }
    
    /// 配置允许的文件类型
    pub fn with_allowed_types(mut self, types: Vec<String>) -> Self {
        self.allowed_types = types;
        self
    }
    
    /// 确保上传目录存在
    async fn ensure_upload_dir(&self) -> Result<(), std::io::Error> {
        if !self.upload_dir.exists() {
            fs::create_dir_all(&self.upload_dir).await?;
            info!("创建上传目录: {:?}", self.upload_dir);
        }
        Ok(())
    }
    
    /// 生成唯一文件名
    fn generate_filename(&self, original_name: &str) -> String {
        let uuid = Uuid::new_v4();
        let extension = std::path::Path::new(original_name)
            .extension()
            .and_then(|ext| ext.to_str())
            .unwrap_or("bin");
        format!("{}_{}.{}", 
                chrono::Utc::now().format("%Y%m%d_%H%M%S"),
                uuid.simple(),
                extension)
    }
    
    /// 验证文件类型
    fn is_allowed_type(&self, content_type: &str) -> bool {
        self.allowed_types.iter().any(|allowed| allowed == content_type)
    }
    
    /// 获取文件的公开URL
    fn get_file_url(&self, filename: &str) -> String {
        format!("/uploads/{}", filename)
    }
}

/// 文件上传响应
#[derive(Debug, Serialize)]
pub struct UploadResponse {
    /// 文件名
    pub filename: String,
    /// 文件URL
    pub url: String,
    /// 文件大小
    pub size: u64,
    /// 文件类型
    pub content_type: String,
}

/// 批量上传响应
#[derive(Debug, Serialize)]
pub struct BatchUploadResponse {
    /// 成功上传的文件
    pub files: Vec<UploadResponse>,
    /// 上传失败的文件
    pub errors: Vec<UploadError>,
}

/// 上传错误
#[derive(Debug, Serialize)]
pub struct UploadError {
    /// 文件名
    pub filename: String,
    /// 错误信息
    pub error: String,
}

impl UploadController {
    /// 上传头像
    pub async fn upload_avatar(
        State(controller): State<UploadController>,
        Extension(current_user): Extension<CurrentUser>,
        mut multipart: Multipart,
    ) -> Response {
        info!("用户{}上传头像", current_user.user_id);
        
        // 确保上传目录存在
        if let Err(e) = controller.ensure_upload_dir().await {
            error!("创建上传目录失败: {:?}", e);
            return Self::simple_error_response("服务器配置错误");
        }
        
        // 处理单个文件上传
        while let Some(field) = multipart.next_field().await.unwrap_or(None) {
            let name = field.name().unwrap_or("").to_string();
            if name != "avatar" {
                continue;
            }
            
            let filename = field.file_name().unwrap_or("avatar").to_string();
            let content_type = field.content_type().unwrap_or("application/octet-stream").to_string();
            
            // 验证文件类型
            if !controller.is_allowed_type(&content_type) {
                warn!("不支持的文件类型: {}", content_type);
                return Self::simple_error_response("不支持的文件类型");
            }
            
            // 读取文件数据
            let data = match field.bytes().await {
                Ok(data) => data,
                Err(e) => {
                    error!("读取文件数据失败: {:?}", e);
                    return Self::simple_error_response("读取文件失败");
                }
            };
            
            // 检查文件大小
            if data.len() > controller.max_file_size {
                warn!("文件大小超限: {} bytes", data.len());
                return Self::simple_error_response("文件大小超过限制");
            }
            
            // 生成文件名
            let new_filename = controller.generate_filename(&filename);
            let file_path = controller.upload_dir.join(&new_filename);
            
            // 保存文件
            match fs::File::create(&file_path).await {
                Ok(mut file) => {
                    if let Err(e) = file.write_all(&data).await {
                        error!("写入文件失败: {:?}", e);
                        return Self::simple_error_response("保存文件失败");
                    }
                }
                Err(e) => {
                    error!("创建文件失败: {:?}", e);
                    return Self::simple_error_response("创建文件失败");
                }
            }
            
            info!("头像上传成功: {} -> {}", filename, new_filename);
            
            let response = UploadResponse {
                filename: new_filename.clone(),
                url: controller.get_file_url(&new_filename),
                size: data.len() as u64,
                content_type,
            };
            
            return ok_response(response);
        }
        
        Self::simple_error_response("未找到有效的头像文件")
    }
    
    /// 批量上传图片
    pub async fn upload_images(
        State(controller): State<UploadController>,
        Extension(current_user): Extension<CurrentUser>,
        mut multipart: Multipart,
    ) -> Response {
        info!("用户{}批量上传图片", current_user.user_id);
        
        // 确保上传目录存在
        if let Err(e) = controller.ensure_upload_dir().await {
            error!("创建上传目录失败: {:?}", e);
            return Self::simple_error_response("服务器配置错误");
        }
        
        let mut uploaded_files = Vec::new();
        let mut upload_errors = Vec::new();
        
        // 处理多个文件上传
        while let Some(field) = multipart.next_field().await.unwrap_or(None) {
            let name = field.name().unwrap_or("").to_string();
            if name != "images" {
                continue;
            }
            
            let filename = field.file_name().unwrap_or("image").to_string();
            let content_type = field.content_type().unwrap_or("application/octet-stream").to_string();
            
            // 验证文件类型
            if !controller.is_allowed_type(&content_type) {
                warn!("不支持的文件类型: {} - {}", filename, content_type);
                upload_errors.push(UploadError {
                    filename: filename.clone(),
                    error: "不支持的文件类型".to_string(),
                });
                continue;
            }
            
            // 读取文件数据
            let data = match field.bytes().await {
                Ok(data) => data,
                Err(e) => {
                    error!("读取文件数据失败: {} - {:?}", filename, e);
                    upload_errors.push(UploadError {
                        filename: filename.clone(),
                        error: "读取文件失败".to_string(),
                    });
                    continue;
                }
            };
            
            // 检查文件大小
            if data.len() > controller.max_file_size {
                warn!("文件大小超限: {} - {} bytes", filename, data.len());
                upload_errors.push(UploadError {
                    filename: filename.clone(),
                    error: "文件大小超过限制".to_string(),
                });
                continue;
            }
            
            // 生成文件名
            let new_filename = controller.generate_filename(&filename);
            let file_path = controller.upload_dir.join(&new_filename);
            
            // 保存文件
            match fs::File::create(&file_path).await {
                Ok(mut file) => {
                    if let Err(e) = file.write_all(&data).await {
                        error!("写入文件失败: {} - {:?}", filename, e);
                        upload_errors.push(UploadError {
                            filename: filename.clone(),
                            error: "保存文件失败".to_string(),
                        });
                        continue;
                    }
                }
                Err(e) => {
                    error!("创建文件失败: {} - {:?}", filename, e);
                    upload_errors.push(UploadError {
                        filename: filename.clone(),
                        error: "创建文件失败".to_string(),
                    });
                    continue;
                }
            }
            
            debug!("图片上传成功: {} -> {}", filename, new_filename);
            
            uploaded_files.push(UploadResponse {
                filename: new_filename.clone(),
                url: controller.get_file_url(&new_filename),
                size: data.len() as u64,
                content_type,
            });
        }
        
        info!("批量上传完成: 成功{}个, 失败{}个", uploaded_files.len(), upload_errors.len());
        
        let response = BatchUploadResponse {
            files: uploaded_files,
            errors: upload_errors,
        };
        
        ok_response(response)
    }
    
    /// 删除文件
    pub async fn delete_file(
        State(controller): State<UploadController>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<DeleteFileRequest>,
    ) -> Response {
        info!("用户{}删除文件: {}", current_user.user_id, request.filename);
        
        // 验证文件名安全性（防止路径遍历攻击）
        if request.filename.contains("..") || request.filename.contains("/") || request.filename.contains("\\") {
            warn!("不安全的文件名: {}", request.filename);
            return Self::simple_error_response("无效的文件名");
        }
        
        let file_path = controller.upload_dir.join(&request.filename);
        
        // 检查文件是否存在
        if !file_path.exists() {
            return Self::simple_error_response("文件不存在");
        }
        
        // 删除文件
        match fs::remove_file(&file_path).await {
            Ok(_) => {
                info!("文件删除成功: {}", request.filename);
                ok_response(serde_json::json!({"success": true}))
            }
            Err(e) => {
                error!("删除文件失败: {} - {:?}", request.filename, e);
                Self::simple_error_response("删除文件失败")
            }
        }
    }
}

/// 删除文件请求
#[derive(Debug, Deserialize)]
pub struct DeleteFileRequest {
    pub filename: String,
}

#[async_trait::async_trait]
impl crate::presentation::http::controllers::Controller for UploadController {
    fn name(&self) -> &'static str {
        "UploadController"
    }
}
