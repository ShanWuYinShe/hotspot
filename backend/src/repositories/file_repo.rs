//! 文件仓库 - 管理上传文件的数据库操作
//!
//! 适配新 Schema：
//! - media_files 表仅存储核心存储信息
//! - 元数据存储在 meta JSONB 字段中
//! - 移除 category, is_deleted 字段

use sqlx::PgPool;
use uuid::Uuid;
use serde_json::json;

use crate::error::{AppError, Result};

/// 上传文件数据 - 用于业务层传输
#[derive(Debug, Clone)]
pub struct UploadedFileData {
    pub file_id: Uuid,
    pub storage_path: String,
    pub thumbnail_url: Option<String>,
    pub file_size: i64,
    pub duration: Option<i32>,
    pub filename: Option<String>,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub mime_type: String,
}

#[derive(Clone)]
pub struct FileRepository {
    pool: PgPool,
}

impl FileRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建上传文件记录
    /// 将分散的元数据打包存入 meta JSONB 字段
    #[allow(clippy::too_many_arguments)]
    pub async fn create_file_with_id(
        &self,
        file_id: Uuid,
        uploader_id: i64,
        _category: &str, // 数据库不再存储 category，仅在 upload_sessions 中保留
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
        // 构建 meta JSONB
        let meta = json!({
            "size": file_size,
            "mime_type": mime_type,
            "width": width,
            "height": height,
            "duration": duration,
            "original_name": original_name
        });

        let row = sqlx::query!(
            r#"
            INSERT INTO media_files (
                file_id, uploader_id, url, thumbnail_url, storage_path, meta, hash_md5
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING file_id, url, storage_path, thumbnail_url, meta, hash_md5
            "#,
            file_id,
            uploader_id,
            url,
            thumbnail_url,
            storage_path,
            meta,
            checksum
        )
        .fetch_one(&self.pool)
        .await?;

        // 解析 meta
        let meta_json = row.meta.unwrap_or(json!({}));
        
        Ok(UploadedFileData {
            file_id: row.file_id,
            storage_path: row.storage_path,
            thumbnail_url: row.thumbnail_url,
            file_size: meta_json["size"].as_i64().unwrap_or(0),
            duration: meta_json["duration"].as_i64().map(|v| v as i32),
            filename: meta_json["original_name"].as_str().map(|s| s.to_string()),
            width: meta_json["width"].as_i64().map(|v| v as i32),
            height: meta_json["height"].as_i64().map(|v| v as i32),
            mime_type: meta_json["mime_type"].as_str().unwrap_or("application/octet-stream").to_string(),
        })
    }

    /// 验证文件所有权并返回信息
    /// 注意：不再验证 category，因为 category 信息不在 media_files 表中
    /// 如果必须验证 category，需要 join upload_sessions 表 (但 session 可能已清除)
    /// 或者信任调用方传入的 file_id 是合法的
    pub async fn validate_file(
        &self,
        file_id: Uuid,
        uploader_id: i64,
        _expected_category: &str, 
    ) -> Result<UploadedFileData> {
        let row = sqlx::query!(
            r#"
            SELECT file_id, uploader_id, url, storage_path, thumbnail_url, meta
            FROM media_files
            WHERE file_id = $1
            "#,
            file_id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| AppError::NotFound("文件不存在".to_string()))?;

        if row.uploader_id != uploader_id {
            return Err(AppError::Forbidden("无权操作此文件".to_string()));
        }

        let meta_json = row.meta.unwrap_or(json!({}));

        // (可选) 如果 meta 中存储了 category，可以在这里验证
        // 目前 schema meta 中未强制存储 category

        Ok(UploadedFileData {
            file_id: row.file_id,
            storage_path: row.storage_path,
            thumbnail_url: row.thumbnail_url,
            file_size: meta_json["size"].as_i64().unwrap_or(0),
            duration: meta_json["duration"].as_i64().map(|v| v as i32),
            filename: meta_json["original_name"].as_str().map(|s| s.to_string()),
            width: meta_json["width"].as_i64().map(|v| v as i32),
            height: meta_json["height"].as_i64().map(|v| v as i32),
            mime_type: meta_json["mime_type"].as_str().unwrap_or("").to_string(),
        })
    }

    /// 标记文件已使用 (Legacy: schema 中移除了 is_used/used_at)
    /// 如果业务需要保留"引用计数"或"使用状态"，应通过关联表查询 (e.g. JOIN moment_images)
    /// 这里的实现视为空操作，或者如果确实需要，应更新 expires_at = NULL (表示永久存储)
    pub async fn mark_as_used(&self, file_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE media_files
            SET expires_at = NULL -- 永久保留
            WHERE file_id = $1
            "#,
            file_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取文件用于消息
    pub async fn get_file_for_message(
        &self,
        file_id: Uuid,
        uploader_id: i64,
        _expected_categories: &[&str],
    ) -> Result<UploadedFileData> {
        // 复用 validate_file 逻辑 (忽略 category 检查)
        self.validate_file(file_id, uploader_id, "").await
    }
}
