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
    pub is_revoked: bool,
    pub revoked_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, sqlx::FromRow)]
pub struct ShareListRow {
    pub share_id: Uuid,
    pub share_code: String,
    pub share_type: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub is_revoked: bool,
    pub payload: serde_json::Value,
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
        expires_at: Option<DateTime<Utc>>,
    ) -> Result<Uuid> {
        let row = sqlx::query!(
            r#"
            INSERT INTO shares (share_code, share_type, payload, sharer_id, expires_at)
            VALUES ($1, $2::text::share_type, $3, $4, $5)
            RETURNING share_id
            "#,
            share_code,
            share_type,
            payload,
            creator_id,
            expires_at
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
                   is_revoked,
                   revoked_at,
                   created_at as "created_at!"
            FROM shares
            WHERE share_code = $1 AND is_revoked = FALSE
            "#,
            share_code
        )
        .fetch_optional(pool)
        .await?;

        Ok(row)
    }

    /// 按 share_id 获取分享记录
    pub async fn get_by_id(pool: &PgPool, share_id: Uuid) -> Result<Option<ShareRow>> {
        let row = sqlx::query_as!(
            ShareRow,
            r#"
            SELECT share_id as "share_id!",
                   sharer_id as "creator_id!",
                   share_code as "share_code!",
                   share_type::text as "share_type!",
                   payload as "payload!",
                   expires_at,
                   is_revoked,
                   revoked_at,
                   created_at as "created_at!"
            FROM shares
            WHERE share_id = $1
            "#,
            share_id
        )
        .fetch_optional(pool)
        .await?;

        Ok(row)
    }

    /// 按 share_id 删除分享（撤销）
    pub async fn revoke_by_id(pool: &PgPool, share_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            UPDATE shares
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE share_id = $1 AND is_revoked = FALSE
            "#,
            share_id
        )
        .execute(pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    pub async fn list_by_creator(
        pool: &PgPool,
        creator_id: i64,
        share_type: Option<&str>,
        cursor: Option<(DateTime<Utc>, Uuid)>,
        limit: i64,
    ) -> Result<Vec<ShareListRow>> {
        let (cursor_time, cursor_id) = cursor
            .map(|(ts, id)| (Some(ts), Some(id)))
            .unwrap_or((None, None));

        let rows = sqlx::query_as!(
            ShareListRow,
            r#"
            SELECT share_id as "share_id!",
                   share_code as "share_code!",
                   share_type::text as "share_type!",
                   created_at as "created_at!",
                   expires_at,
                   is_revoked,
                   payload as "payload!"
            FROM shares
            WHERE sharer_id = $1
              AND ($2::text IS NULL OR share_type::text = $2::text)
              AND (
                    $3::timestamptz IS NULL
                 OR created_at < $3
                 OR (created_at = $3 AND ($4::uuid IS NULL OR share_id < $4))
              )
            ORDER BY created_at DESC, share_id DESC
            LIMIT $5
            "#,
            creator_id,
            share_type,
            cursor_time,
            cursor_id,
            limit
        )
        .fetch_all(pool)
        .await?;

        Ok(rows)
    }
}
