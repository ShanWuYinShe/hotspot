// src/repositories/chat_unread_repo.rs
// 聊天未读计数Repository（统一管理私聊和群聊的未读计数）

use crate::error::AppError;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

/// 聊天未读计数Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ChatUnreadCountRow {
    pub id: i64,
    pub chat_type: String, // 'private_chat' 或 'group_chat'
    pub chat_id: Uuid,     // private_chat_id 或 group_chat_id（UUID）
    pub user_id: i64,      // BIGINT（引用 users.user_id）

    // 最后已读信息
    pub last_read_at: Option<DateTime<Utc>>,
    pub last_read_message_id: Option<Uuid>,

    // 时间戳
    pub updated_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct ChatUnreadRepository {
    pool: PgPool,
}

impl ChatUnreadRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 标记已读
    pub async fn mark_as_read(
        &self,
        chat_type: &str,
        chat_id: &str,
        user_id: i64,
        last_message_id: Option<Uuid>,
    ) -> Result<(), AppError> {
        let chat_uuid = Uuid::parse_str(chat_id)
            .map_err(|_| AppError::BadRequest("Invalid chat_id".to_string()))?;

        // 使用 UPSERT 模式
        sqlx::query!(
            r#"
            INSERT INTO chat_unread_counts (chat_type, chat_id, user_id, last_read_message_id, last_read_at)
            VALUES ($1, $2, $3, $4, NOW())
            ON CONFLICT (chat_type, chat_id, user_id)
            DO UPDATE SET 
                last_read_message_id = COALESCE($4, chat_unread_counts.last_read_message_id),
                last_read_at = NOW(),
                updated_at = NOW()
            "#,
            chat_type,
            chat_uuid,
            user_id,
            last_message_id
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }

    /// 获取未读计数
    pub async fn get_unread_count(
        &self,
        chat_type: &str,
        chat_id: &str,
        user_id: i64,
    ) -> Result<i64, AppError> {
        let chat_uuid = Uuid::parse_str(chat_id)
            .map_err(|_| AppError::BadRequest("Invalid chat_id".to_string()))?;

        // 获取最后已读时间
        let record = sqlx::query!(
            r#"
            SELECT last_read_at
            FROM chat_unread_counts
            WHERE chat_type = $1 AND chat_id = $2 AND user_id = $3
            "#,
            chat_type,
            chat_uuid,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        // 如果没有读取记录，使用 Unix 纪元作为默认值（统计所有消息）
        let read_at = record
            .and_then(|r| r.last_read_at)
            .unwrap_or_else(|| chrono::DateTime::UNIX_EPOCH);

        // 根据聊天类型查询未读消息数
        let count = match chat_type {
            "private_chat" => {
                sqlx::query_scalar!(
                    r#"
                    SELECT COUNT(*) as "count!"
                    FROM private_messages
                    WHERE private_chat_id = $1 
                      AND sender_id != $2 
                      AND sent_at > $3
                      AND is_deleted = FALSE
                    "#,
                    chat_uuid,
                    user_id,
                    read_at
                )
                .fetch_one(&self.pool)
                .await?
            }
            "group_chat" => {
                sqlx::query_scalar!(
                    r#"
                    SELECT COUNT(*) as "count!"
                    FROM group_messages
                    WHERE group_chat_id = $1 
                      AND sender_id != $2 
                      AND sent_at > $3
                      AND is_deleted = FALSE
                    "#,
                    chat_uuid,
                    user_id,
                    read_at
                )
                .fetch_one(&self.pool)
                .await?
            }
            _ => 0,
        };

        Ok(count)
    }

    /// 获取用户所有未读聊天列表
    pub async fn list_unread_chats_by_user(&self, user_id: i64) -> Result<Vec<ChatUnreadCountRow>, AppError> {
        let rows = sqlx::query_as!(
            ChatUnreadCountRow,
            r#"
            SELECT id, chat_type, chat_id, user_id, last_read_at, last_read_message_id, updated_at
            FROM chat_unread_counts
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }
}
