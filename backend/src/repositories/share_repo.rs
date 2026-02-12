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
        let result = sqlx::query!("DELETE FROM shares WHERE share_id = $1", share_id)
            .execute(pool)
            .await?;

        Ok(result.rows_affected() > 0)
    }

    /// 按创建者分页列分享
    pub async fn list_by_creator(
        pool: &PgPool,
        creator_id: i64,
        limit: i64,
        cursor: Option<&str>,
    ) -> Result<(Vec<ShareRow>, Option<String>)> {
        use crate::utils::pagination::{decode_cursor, encode_cursor};

        let cursor_ts = cursor.and_then(|c| decode_cursor::<chrono::DateTime<Utc>>(c).ok());

        let rows = if let Some(ts) = cursor_ts {
            sqlx::query_as!(
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
                WHERE sharer_id = $1 AND created_at < $2
                ORDER BY created_at DESC
                LIMIT $3
                "#,
                creator_id,
                ts,
                limit + 1
            )
            .fetch_all(pool)
            .await?
        } else {
            sqlx::query_as!(
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
                WHERE sharer_id = $1
                ORDER BY created_at DESC
                LIMIT $2
                "#,
                creator_id,
                limit + 1
            )
            .fetch_all(pool)
            .await?
        };

        let has_more = rows.len() as i64 > limit;
        let items: Vec<ShareRow> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
