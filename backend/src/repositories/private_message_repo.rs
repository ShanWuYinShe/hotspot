use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;

/// 私聊消息Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessageRow {
    pub id: i64,
    pub message_id: Uuid,
    pub cursor_id: Uuid,
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
pub struct PrivateMessageRepository {
    pool: PgPool,
}

impl PrivateMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建私聊消息
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
            RETURNING id, message_id, cursor_id, private_chat_id, sender_id,
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
            SELECT id, message_id, cursor_id, private_chat_id, sender_id,
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

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM private_messages WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 获取私聊消息历史（分页）
    pub async fn list_by_chat(
        &self,
        private_chat_id: Uuid,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<PrivateMessageRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;

        let rows = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT id, message_id, cursor_id, private_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id,
                   file_name, file_size, thumbnail_url, duration, sent_at, edited_at,
                   is_recalled, recalled_at, is_deleted, deleted_at
            FROM private_messages
            WHERE private_chat_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            private_chat_id,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
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
    pub async fn get_last_message(&self, private_chat_id: Uuid) -> Result<Option<PrivateMessageRow>> {
        let row = sqlx::query_as!(
            PrivateMessageRow,
            r#"
            SELECT id, message_id, cursor_id, private_chat_id, sender_id,
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

    /// 获取私聊未读消息数
    pub async fn get_unread_count(&self, private_chat_id: Uuid, user_id: i64, last_read_at: Option<DateTime<Utc>>) -> Result<i64> {
        // 如果没有读取记录，使用 Unix 纪元作为默认值（统计所有消息）
        let read_at = last_read_at.unwrap_or(DateTime::UNIX_EPOCH);
        
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM private_messages
            WHERE private_chat_id = $1 
              AND sender_id != $2 
              AND sent_at > $3
              AND is_deleted = FALSE
            "#,
            private_chat_id,
            user_id,
            read_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(count)
    }
}

