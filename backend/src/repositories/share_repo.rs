//! 分享数据仓库
//!
//! 适配新 Schema: share_id, sharer_id, content_type, content_id, platform, created_at

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone, sqlx::FromRow, Serialize, Deserialize)]
pub struct ShareRow {
    pub share_id: Uuid,
    pub creator_id: i64,
    pub share_code: String,
    pub share_type: String,
    pub payload: serde_json::Value,
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

pub struct ShareRepository;

impl ShareRepository {
    /// 创建分享记录
    pub async fn create(
        pool: &PgPool,
        share_code: &str,
        share_type: &str,
        creator_id: i64,
        payload: &serde_json::Value,
    ) -> Result<Uuid> {
        let row = sqlx::query!(
            r#"
            INSERT INTO shares (share_code, share_type, payload, sharer_id)
            VALUES ($1, $2::text::share_type, $3, $4)
            RETURNING share_id
            "#,
            share_code,
            share_type,
            payload,
            creator_id
        )
        .fetch_one(pool)
        .await?;

        Ok(row.share_id)
    }

    /// 获取分享记录
    pub async fn get_by_code(pool: &PgPool, share_code: &str) -> Result<Option<ShareRow>> {
        let row = sqlx::query_as!(
            ShareRow,
            r#"
            SELECT share_id as "share_id!",
                   sharer_id as "creator_id!",
                   share_code as "share_code!",
                   share_type::text as "share_type!",
                   payload as "payload!",
                   expires_at,
                   created_at as "created_at!"
            FROM shares
            WHERE share_code = $1
            "#,
            share_code
        )
        .fetch_optional(pool)
        .await?;

        Ok(row)
    }

    pub async fn exists_by_code(pool: &PgPool, share_code: &str) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(SELECT 1 FROM shares WHERE share_code = $1) as "exists!"
            "#,
            share_code
        )
        .fetch_one(pool)
        .await?;

        Ok(exists)
    }
}
