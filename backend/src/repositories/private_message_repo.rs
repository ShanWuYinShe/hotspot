use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;

use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 私聊消息Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessageRow {
    pub id: i64,
    pub message_id: Uuid,
    // cursor field removed
    pub private_chat_id: Uuid,
    pub sender_id: i64,
    pub r#type: MessageType,
    pub content: String,
    pub reply_to_message_id: Option<Uuid>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub is_recalled: bool,
    pub recalled_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct PrivateMessageMeta {
    pub private_chat_id: Uuid,
    pub sender_id: i64,
    pub is_recalled: bool,
    pub sent_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct PrivateMessageRepository {
    pool: PgPool,
}

impl PrivateMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取消息元数据（用于权限检查）
    pub async fn get_meta(&self, message_id: Uuid) -> Result<Option<PrivateMessageMeta>> {
        let row = sqlx::query_as!(
            PrivateMessageMeta,
            r#"
            SELECT private_chat_id, sender_id, is_recalled, sent_at
            FROM private_messages
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 创建私聊消息
    #[allow(clippy::too_many_arguments)]
    pub async fn create(
        &self,
        private_chat_id: Uuid,
        sender_id: i64,
        message_type: MessageType,
        content: &str,
        reply_to_message_id: Option<Uuid>,
        file_name: Option<&str>,
        file_size: Option<i64>,
        thumbnail_url: Option<&str>,
        duration: Option<i32>,
    ) -> Result<PrivateMessageRow> {
        let row = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            INSERT INTO private_messages (private_chat_id, sender_id, type, content, reply_to_message_id, file_name, file_size, thumbnail_url, duration)
            VALUES ($1, $2, $3::message_type, $4, $5, $6, $7, $8, $9)
            RETURNING id, message_id, private_chat_id, sender_id,
                      type as "type: MessageType", content, reply_to_message_id,
                      file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                      is_recalled, recalled_at, is_deleted, deleted_at
            "#,
            private_chat_id,
            sender_id,
            message_type as MessageType,
            content,
            reply_to_message_id,
            file_name,
            file_size,
            thumbnail_url,
            duration
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按消息ID查找
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<PrivateMessageRow>> {
        let row = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT id, message_id, private_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id,
                   file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                   is_recalled, recalled_at, is_deleted, deleted_at
            FROM private_messages
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取私聊消息历史（分页）
    pub async fn list_by_chat(
        &self,
        private_chat_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateMessageRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT id, message_id, private_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id,
                   file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                   is_recalled, recalled_at, is_deleted, deleted_at
            FROM private_messages
            WHERE private_chat_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            private_chat_id,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((rows, next_cursor))
    }

    /// 更新消息内容
    pub async fn update_content(&self, message_id: Uuid, content: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_messages
            SET content = $2, edited_at = NOW()
            WHERE message_id = $1 AND is_deleted = FALSE AND is_recalled = FALSE
            "#,
            message_id,
            content
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_messages
            SET is_recalled = TRUE, recalled_at = NOW(), content = '[消息已撤回]'
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除消息
    pub async fn soft_delete(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_messages
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE message_id = $1
            "#,
            message_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取私聊的最后一条消息
    pub async fn get_last_message(
        &self,
        private_chat_id: Uuid,
    ) -> Result<Option<PrivateMessageRow>> {
        let row = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT id, message_id, private_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id,
                   file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                   is_recalled, recalled_at, is_deleted, deleted_at
            FROM private_messages
            WHERE private_chat_id = $1 AND is_deleted = FALSE
            ORDER BY sent_at DESC
            LIMIT 1
            "#,
            private_chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查某用户是否在该私聊中发送过消息
    /// 用于判断是否应该解锁对方的消息限制
    pub async fn has_messages_from_user(
        &self,
        private_chat_id: Uuid,
        user_id: i64,
    ) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM private_messages
                WHERE private_chat_id = $1 
                  AND sender_id = $2 
                  AND is_deleted = FALSE
            ) as "exists!"
            "#,
            private_chat_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(exists)
    }

    /// 批量获取私聊的最后一条消息 (Map<private_chat_id, Message>)
    pub async fn get_last_message_batch(
        &self,
        private_chat_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, PrivateMessageRow>> {
        if private_chat_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT DISTINCT ON (private_chat_id)
                   id, message_id, private_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id,
                   file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                   is_recalled, recalled_at, is_deleted, deleted_at
            FROM private_messages
            WHERE private_chat_id = ANY($1) AND is_deleted = FALSE
            ORDER BY private_chat_id, sent_at DESC
            "#,
            private_chat_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.private_chat_id, r)).collect())
    }
}
