//! 文件仓库 - 管理上传文件的数据库操作

use sqlx::PgPool;
use uuid::Uuid;

use crate::error::{AppError, Result};

/// 上传文件数据 - 用于 create_file 返回
/// 只包含业务需要的字段
#[derive(Debug, Clone)]
pub struct UploadedFileData {
    pub file_id: Uuid,
    pub url: String,
    pub thumbnail_url: Option<String>,
    pub mime_type: String,
    pub file_size: i64,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub duration: Option<i32>,
}

/// 文件验证结果 - validate_file 返回的 URL
pub type FileValidationResult = String;

/// 已存在文件信息 - 用于物理文件去重，复用 storage_path
#[derive(Debug, Clone)]
pub struct ExistingFileInfo {
    pub storage_path: String,
}

// 注：分块上传使用 Redis 存储临时会话状态，数据库表 upload_sessions 和 upload_chunks 
// 作为备用方案保留，当前未使用

#[derive(Clone)]
pub struct FileRepository {
    pool: PgPool,
}

impl FileRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建上传文件记录（指定 file_id）
    /// 每次上传都生成新的 file_id 和 url，客户端无法判断文件是否已存在
    pub async fn create_file_with_id(
        &self,
        file_id: Uuid,
        uploader_id: i64,
        category: &str,
        original_name: Option<&str>,
        storage_path: &str,
        url: &str,
        thumbnail_url: Option<&str>,
        mime_type: &str,
        file_size: i64,
        width: Option<i32>,
        height: Option<i32>,
        duration: Option<i32>,
        checksum: Option<&str>,
    ) -> Result<UploadedFileData> {
        let row = sqlx::query!(
            r#"
            INSERT INTO uploaded_files (
                file_id, uploader_id, category, original_name, storage_path, url,
                thumbnail_url, mime_type, file_size, width, height, duration, checksum
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
            RETURNING file_id, url, thumbnail_url, mime_type, file_size, width, height, duration
            "#,
            file_id,
            uploader_id,
            category,
            original_name,
            storage_path,
            url,
            thumbnail_url,
            mime_type,
            file_size,
            width,
            height,
            duration,
            checksum
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(UploadedFileData {
            file_id: row.file_id,
            url: row.url,
            thumbnail_url: row.thumbnail_url,
            mime_type: row.mime_type,
            file_size: row.file_size,
            width: row.width,
            height: row.height,
            duration: row.duration,
        })
    }

    /// 验证文件所有权和分类，返回文件 URL
    pub async fn validate_file(&self, file_id: Uuid, uploader_id: i64, expected_category: &str) -> Result<FileValidationResult> {
        let row = sqlx::query!(
            r#"
            SELECT uploader_id, category, url
            FROM uploaded_files
            WHERE file_id = $1 AND is_deleted = FALSE
            "#,
            file_id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| AppError::NotFound("文件不存在".to_string()))?;

        if row.uploader_id != uploader_id {
            return Err(AppError::Forbidden("无权操作此文件".to_string()));
        }

        if row.category != expected_category {
            return Err(AppError::BadRequest(format!(
                "文件分类不匹配，期望 {}，实际 {}", expected_category, row.category
            )));
        }

        Ok(row.url)
    }

    /// 标记文件已使用
    pub async fn mark_as_used(&self, file_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE uploaded_files
            SET is_used = TRUE, used_at = NOW()
            WHERE file_id = $1 AND is_deleted = FALSE
            "#,
            file_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 通过 checksum 查找任意文件（物理文件去重）
    /// 只返回 storage_path，用于复用物理文件
    pub async fn find_any_by_checksum(&self, checksum: &str) -> Result<Option<ExistingFileInfo>> {
        let row = sqlx::query!(
            r#"
            SELECT storage_path
            FROM uploaded_files
            WHERE checksum = $1 AND is_deleted = FALSE
            LIMIT 1
            "#,
            checksum
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.map(|r| ExistingFileInfo {
            storage_path: r.storage_path,
        }))
    }

    /// 验证文件所有权并获取完整文件信息（用于消息发送时转换 file_id → 文件数据）
    /// 验证：文件存在、属于当前用户、分类匹配
    pub async fn get_file_for_message(
        &self,
        file_id: Uuid,
        uploader_id: i64,
        expected_categories: &[&str],
    ) -> Result<UploadedFileData> {
        let row = sqlx::query!(
            r#"
            SELECT file_id, uploader_id, category, url, thumbnail_url, mime_type, 
                   file_size, width, height, duration
            FROM uploaded_files
            WHERE file_id = $1 AND is_deleted = FALSE
            "#,
            file_id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| AppError::NotFound("文件不存在".to_string()))?;

        if row.uploader_id != uploader_id {
            return Err(AppError::Forbidden("无权使用此文件".to_string()));
        }

        if !expected_categories.contains(&row.category.as_str()) {
            return Err(AppError::BadRequest(format!(
                "文件分类不匹配，期望 {:?}，实际 {}", expected_categories, row.category
            )));
        }

        Ok(UploadedFileData {
            file_id: row.file_id,
            url: row.url,
            thumbnail_url: row.thumbnail_url,
            mime_type: row.mime_type,
            file_size: row.file_size,
            width: row.width,
            height: row.height,
            duration: row.duration,
        })
    }
}
