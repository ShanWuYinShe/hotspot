//! 群聊消息仓库层实现
//! 
//! 适配新 Schema: message_id (UUID), group_id (BIGINT), 分区表 group_messages_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeIdCursor};

/// 群聊消息行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMessageRow {
    pub message_id: Uuid,
    pub group_id: i64,
    pub sender_id: i64,
    pub r#type: MessageType,
    pub content: Option<String>,
    pub metadata: serde_json::Value,
    pub reply_to_id: Option<Uuid>,
    pub is_recalled: bool,
    pub reaction_counts: serde_json::Value,
    pub mentions: serde_json::Value,
    pub read_count: i32,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub recalled_at: Option<DateTime<Utc>>,
    pub recalled_by: Option<i64>,
}

#[derive(Debug, Clone)]
pub struct GroupMessageRepository {
    pool: PgPool,
}

impl GroupMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 发送群聊消息
    pub async fn create(
        &self,
        group_id: i64,
        sender_id: i64,
        msg_type: MessageType,
        content: Option<String>,
        metadata: Option<serde_json::Value>,
        reply_to_id: Option<Uuid>,
        at_users: Option<Vec<i64>>,
    ) -> Result<GroupMessageRow> {
        let row = sqlx::query_as_unchecked!(
            GroupMessageRow,
            r#"
            INSERT INTO group_messages_partitioned (
                group_id, sender_id, message_type, content, metadata, reply_to_id, at_users
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING message_id, group_id, sender_id, message_type as "type: MessageType", content, metadata, 
                      reply_to_id, is_recalled, reaction_counts, at_users as "mentions!", read_count, 
                      sent_at, edited_at, recalled_at, recalled_by
            "#,
            group_id,
            sender_id,
            msg_type.to_string(),
            content,
            metadata.unwrap_or(serde_json::json!({})),
            reply_to_id,
            serde_json::to_value(at_users.unwrap_or_default()).unwrap_or(serde_json::json!([]))
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据 ID 查找
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<GroupMessageRow>> {
        let row = sqlx::query_as_unchecked!(
            GroupMessageRow,
            r#"
            SELECT message_id, group_id, sender_id, message_type as "type: MessageType", content, metadata,
                   reply_to_id, is_recalled, reaction_counts, at_users as "mentions!", read_count,
                   sent_at, edited_at, recalled_at, recalled_by
            FROM group_messages
            WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
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
                FROM group_messages
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

    /// 列出群聊消息历史
    pub async fn list_by_group(
        &self,
        group_id: i64,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
        include_system: bool,
    ) -> Result<(Vec<GroupMessageRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as_unchecked!(
            GroupMessageRow,
            r#"
            SELECT message_id, group_id, sender_id, message_type as "type: MessageType", content, metadata,
                   reply_to_id, is_recalled, reaction_counts, at_users as "mentions!", read_count,
                   sent_at, edited_at, recalled_at, recalled_by
            FROM group_messages
            WHERE group_id = $1
              AND (sent_at, message_id) < ($3, $4)
              AND ($2::boolean OR message_type <> 'system')
              AND NOT EXISTS (
                  SELECT 1 FROM group_message_deletions d
                  WHERE d.message_id = group_messages.message_id
                    AND d.user_id = $5
              )
            ORDER BY sent_at DESC, message_id DESC
            LIMIT $6
            "#,
            group_id,
            include_system,
            last_ts,
            last_id,
            user_id,
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

    pub async fn find_by_ids(&self, message_ids: &[Uuid]) -> Result<Vec<GroupMessageRow>> {
        if message_ids.is_empty() {
            return Ok(vec![]);
        }

        let rows = sqlx::query_as_unchecked!(
            GroupMessageRow,
            r#"
            SELECT message_id, group_id, sender_id, message_type as "type: MessageType", content, metadata,
                   reply_to_id, is_recalled, reaction_counts, at_users as "mentions!", read_count,
                   sent_at, edited_at, recalled_at, recalled_by
            FROM group_messages
            WHERE message_id = ANY($1)
            "#,
            message_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn get_read_counts(&self, message_ids: &[Uuid]) -> Result<std::collections::HashMap<Uuid, i64>> {
        if message_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT message_id, read_count
            FROM group_messages_partitioned
            WHERE message_id = ANY($1)
            "#,
            message_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.message_id, r.read_count.unwrap_or(0) as i64))
            .collect())
    }

    pub async fn get_meta(&self, message_id: Uuid) -> Result<Option<GroupMessageRow>> {
        self.find_by_id(message_id).await
    }

    pub async fn update_content(&self, message_id: Uuid, content: String) -> Result<()> {
        sqlx::query!(
            "UPDATE group_messages_partitioned SET content = $1, edited_at = NOW() WHERE message_id = $2",
            content,
            message_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 撤回群消息
    pub async fn recall(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE group_messages_partitioned SET is_recalled = TRUE, recalled_at = NOW(), recalled_by = $2, content = NULL, metadata = '{}'::jsonb WHERE message_id = $1",
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn delete_for_user(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO group_message_deletions (message_id, user_id)
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

    pub async fn delete_for_everyone(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE group_messages_partitioned SET is_recalled = TRUE, recalled_at = NOW(), recalled_by = $2, content = NULL, metadata = '{}'::jsonb WHERE message_id = $1",
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
