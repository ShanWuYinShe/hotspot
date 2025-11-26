use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;

/// 群聊消息Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMessageRow {
    pub id: i64,
    pub message_id: Uuid,
    pub cursor_id: Uuid,
    pub group_chat_id: Uuid,
    pub sender_id: i64,
    pub r#type: MessageType,
    pub content: String,
    pub reply_to_message_id: Option<Uuid>,
    pub mentions: Option<Vec<i64>>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub is_recalled: bool,
    pub recalled_by: Option<i64>,
    pub recalled_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
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
        group_chat_id: Uuid,
        sender_id: i64,
        r#type: MessageType,
        content: String,
        reply_to_message_id: Option<Uuid>,
        mentions: Option<Vec<i64>>,
    ) -> Result<GroupMessageRow> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            INSERT INTO group_messages (group_chat_id, sender_id, type, content, reply_to_message_id, mentions)
            VALUES ($1, $2, $3::message_type, $4, $5, $6)
            RETURNING id, message_id, cursor_id, group_chat_id, sender_id,
                      type as "type: MessageType", content, reply_to_message_id, mentions,
                      file_name, file_size, thumbnail_url, duration,
                      sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            "#,
            group_chat_id,
            sender_id,
            r#type as MessageType,
            content,
            reply_to_message_id,
            mentions.as_deref()
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找消息
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<GroupMessageRow>> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, cursor_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取群聊消息历史（分页）
    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM group_messages WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// cursor: 游标（cursor_id 的 UUID），用于分页
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<GroupMessageRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;

        let rows = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, cursor_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE group_chat_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            group_chat_id,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 更新消息内容
    pub async fn update_content(&self, message_id: Uuid, content: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET content = $2, edited_at = NOW()
            WHERE message_id = $1
            "#,
            message_id,
            content
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid, recalled_by: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET is_recalled = TRUE, recalled_by = $2, recalled_at = NOW(), content = '[消息已撤回]'
            WHERE message_id = $1
            "#,
            message_id,
            recalled_by
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除消息
    pub async fn soft_delete(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE message_id = $1
            "#,
            message_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取群聊历史消息（旧方法，保留兼容性）
    pub async fn get_history(&self, group_chat_id: Uuid, limit: i64) -> Result<Vec<GroupMessageRow>> {
        self.list_by_group(group_chat_id, limit, None).await
    }

    /// 获取消息已读用户列表
    pub async fn get_read_status(&self, message_id: Uuid) -> Result<Vec<i64>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT user_id
            FROM group_message_read_status
            WHERE message_id = $1
            ORDER BY read_at DESC
            "#,
            message_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 获取群聊的最后一条消息
    pub async fn get_last_message(&self, group_chat_id: Uuid) -> Result<Option<GroupMessageRow>> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, cursor_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE group_chat_id = $1 AND is_deleted = FALSE
            ORDER BY sent_at DESC
            LIMIT 1
            "#,
            group_chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }
}
