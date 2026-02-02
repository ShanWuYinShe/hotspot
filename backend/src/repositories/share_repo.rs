//! 分享数据仓库
//!
//! 负责分享的数据库访问

use crate::error::AppError;
use crate::types::enums::ShareType;
use sqlx::PgPool;
use uuid::Uuid;

/// 分享仓库
pub struct ShareRepository;

impl ShareRepository {
    /// 创建分享
    pub async fn create(
        pool: &PgPool,
        share_code: &str,
        share_type: ShareType,
        creator_id: i64,
        payload: &serde_json::Value,
    ) -> Result<Uuid, AppError> {
        let share_id = sqlx::query_scalar!(
            r#"
            INSERT INTO shares (share_code, share_type, creator_id, payload)
            VALUES ($1, $2, $3, $4)
            RETURNING share_id
            "#,
            share_code,
            share_type.as_str(),
            creator_id,
            payload
        )
        .fetch_one(pool)
        .await
        .map_err(|e| {
            tracing::error!("创建分享失败: {}", e);
            AppError::internal(e.to_string())
        })?;

        Ok(share_id)
    }

    /// 根据 share_code 获取分享
    pub async fn get_by_code(pool: &PgPool, share_code: &str) -> Result<Option<DbShare>, AppError> {
        let share = sqlx::query_as!(
            DbShare,
            r#"
            SELECT share_id, share_code, share_type, creator_id, payload, created_at
            FROM shares
            WHERE share_code = $1
            "#,
            share_code
        )
        .fetch_optional(pool)
        .await
        .map_err(|e| {
            tracing::error!("获取分享失败: {}", e);
            AppError::internal(e.to_string())
        })?;

        Ok(share)
    }

    /// 检查 share_code 是否已存在
    pub async fn exists_by_code(pool: &PgPool, share_code: &str) -> Result<bool, AppError> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(SELECT 1 FROM shares WHERE share_code = $1) AS "exists!"
            "#,
            share_code
        )
        .fetch_one(pool)
        .await
        .map_err(|e| {
            tracing::error!("检查分享码是否存在失败: {}", e);
            AppError::internal(e.to_string())
        })?;

        Ok(exists)
    }
}

/// 数据库分享记录
#[derive(Debug, Clone)]
pub struct DbShare {
    pub share_id: Uuid,
    pub share_code: String,
    pub share_type: String,
    pub creator_id: i64,
    pub payload: serde_json::Value,
    pub created_at: chrono::DateTime<chrono::Utc>,
}
