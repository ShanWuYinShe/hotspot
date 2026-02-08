//! 私聊消息仓库层实现
//! 
//! 适配新 Schema: message_id (UUID), sender/receiver (BIGINT), 分区表 private_messages_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeIdCursor};

/// 私聊消息行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessageRow {
    pub message_id: Uuid,
    pub chat_id: Uuid,
    pub sender_id: i64,
    pub receiver_id: i64,
    pub r#type: MessageType,
    pub content: Option<String>,
    pub metadata: serde_json::Value,
    pub reply_to_message_id: Option<Uuid>,
    pub is_recalled: bool,
    pub reaction_counts: serde_json::Value,
    pub thumbnail_url: Option<String>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub duration: Option<f64>,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub mime_type: Option<String>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub recalled_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct PrivateMessageRepository {
    pool: PgPool,
}

impl PrivateMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 发送私聊消息
    pub async fn create(
        &self,
        chat_id: Uuid,
        sender_id: i64,
        receiver_id: i64,
        msg_type: MessageType,
        content: Option<String>,
        metadata: Option<serde_json::Value>,
        reply_to_id: Option<Uuid>,
    ) -> Result<PrivateMessageRow> {
        let row = sqlx::query_as_unchecked!(
            PrivateMessageRow,
            r#"
            INSERT INTO private_messages_partitioned (
                chat_id, sender_id, receiver_id, message_type, content, metadata, reply_to_id
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING message_id, chat_id, sender_id, receiver_id, 
                      message_type as "type: MessageType", content, metadata, reply_to_id as "reply_to_message_id", 
                      is_recalled, reaction_counts,
                      metadata->>'thumbnail_url' as "thumbnail_url?",
                      metadata->>'file_name' as "file_name?",
                      (metadata->>'size')::bigint as "file_size?",
                      (metadata->>'duration')::float8 as "duration?",
                      (metadata->>'width')::int as "width?",
                      (metadata->>'height')::int as "height?",
                      metadata->>'mime_type' as "mime_type?",
                      sent_at, edited_at, recalled_at
            "#,
            chat_id,
            sender_id,
            receiver_id,
            msg_type.to_string(),
            content,
            metadata.unwrap_or(serde_json::json!({})),
            reply_to_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 列出会话消息历史
    pub async fn list_by_chat(
        &self,
        chat_id: Uuid,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateMessageRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as_unchecked!(
            PrivateMessageRow,
            r#"
            SELECT message_id, chat_id, sender_id, receiver_id, message_type as "type: MessageType",
                   content, metadata, reply_to_id as "reply_to_message_id", is_recalled, reaction_counts,
                   metadata->>'thumbnail_url' as "thumbnail_url?",
                   metadata->>'file_name' as "file_name?",
                   (metadata->>'size')::bigint as "file_size?",
                   (metadata->>'duration')::float8 as "duration?",
                   (metadata->>'width')::int as "width?",
                   (metadata->>'height')::int as "height?",
                   metadata->>'mime_type' as "mime_type?",
                   sent_at, edited_at, recalled_at
            FROM private_messages
            WHERE chat_id = $1
              AND (sent_at, message_id) < ($3, $4)
              AND NOT EXISTS (
                  SELECT 1 FROM private_message_deletions d
                  WHERE d.message_id = private_messages.message_id
                    AND d.user_id = $2
              )
            ORDER BY sent_at DESC, message_id DESC
            LIMIT $5
            "#,
            chat_id,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            items.last().map(|r| encode_cursor(&(r.sent_at, r.message_id))).transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE private_messages_partitioned SET is_recalled = TRUE, recalled_at = NOW(), content = NULL, metadata = '{}'::jsonb WHERE message_id = $1",
            message_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn delete_for_user(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO private_message_deletions (message_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (message_id, user_id) DO NOTHING
            "#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn find_by_ids(&self, message_ids: &[Uuid]) -> Result<Vec<PrivateMessageRow>> {
        if message_ids.is_empty() {
            return Ok(vec![]);
        }

        let rows = sqlx::query_as_unchecked!(
            PrivateMessageRow,
            r#"
            SELECT message_id, chat_id, sender_id, receiver_id, message_type as "type: MessageType",
                   content, metadata, reply_to_id as "reply_to_message_id", is_recalled, reaction_counts,
                   metadata->>'thumbnail_url' as "thumbnail_url?",
                   metadata->>'file_name' as "file_name?",
                   (metadata->>'size')::bigint as "file_size?",
                   (metadata->>'duration')::float8 as "duration?",
                   (metadata->>'width')::int as "width?",
                   (metadata->>'height')::int as "height?",
                   metadata->>'mime_type' as "mime_type?",
                   sent_at, edited_at, recalled_at
            FROM private_messages
            WHERE message_id = ANY($1)
            "#,
            message_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<PrivateMessageRow>> {
        self.get_meta(message_id).await
    }

    pub async fn list_message_ids_after(
        &self,
        last_row_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, Uuid)>> {
        let rows = sqlx::query!(
            r#"
            SELECT row_id as "row_id!", message_id as "message_id!"
            FROM (
                SELECT row_number() OVER (ORDER BY sent_at ASC, message_id ASC)::bigint as row_id,
                       message_id
                FROM private_messages
            ) ranked
            WHERE row_id > $1
            ORDER BY row_id ASC
            LIMIT $2
            "#,
            last_row_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.row_id, r.message_id)).collect())
    }

    pub async fn get_last_message_batch(
        &self,
        chat_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, PrivateMessageRow>> {
        if chat_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query_as_unchecked!(
            PrivateMessageRow,
            r#"
            SELECT DISTINCT ON (chat_id)
                   message_id, chat_id, sender_id, receiver_id, message_type as "type: MessageType",
                   content, metadata, reply_to_id as "reply_to_message_id", is_recalled, reaction_counts,
                   metadata->>'thumbnail_url' as "thumbnail_url?",
                   metadata->>'file_name' as "file_name?",
                   (metadata->>'size')::bigint as "file_size?",
                   (metadata->>'duration')::float8 as "duration?",
                   (metadata->>'width')::int as "width?",
                   (metadata->>'height')::int as "height?",
                   metadata->>'mime_type' as "mime_type?",
                   sent_at, edited_at, recalled_at
            FROM private_messages
            WHERE chat_id = ANY($1)
            ORDER BY chat_id, sent_at DESC, message_id DESC
            "#,
            chat_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.chat_id, r)).collect())
    }

    pub async fn get_meta(&self, message_id: Uuid) -> Result<Option<PrivateMessageRow>> {
        let row = sqlx::query_as_unchecked!(
            PrivateMessageRow,
            r#"
            SELECT message_id, chat_id, sender_id, receiver_id, message_type as "type: MessageType",
                   content, metadata, reply_to_id as "reply_to_message_id", is_recalled, reaction_counts,
                   metadata->>'thumbnail_url' as "thumbnail_url?",
                   metadata->>'file_name' as "file_name?",
                   (metadata->>'size')::bigint as "file_size?",
                   (metadata->>'duration')::float8 as "duration?",
                   (metadata->>'width')::int as "width?",
                   (metadata->>'height')::int as "height?",
                   metadata->>'mime_type' as "mime_type?",
                   sent_at, edited_at, recalled_at
            FROM private_messages
            WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn update_content(&self, message_id: Uuid, content: String) -> Result<()> {
        sqlx::query!(
            "UPDATE private_messages_partitioned SET content = $1, edited_at = NOW() WHERE message_id = $2",
            content,
            message_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
