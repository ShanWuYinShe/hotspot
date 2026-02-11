//! 会话状态仓库层实现 (Inbox)
//! 
//! 适配新 Schema: chat_conversations_state
//! 统一管理私聊和群聊的未读数、置顶、免打扰等状态

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

/// 会话状态行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ChatConversationStateRow {
    pub user_id: i64,
    pub chat_type: String, // 'private', 'group'
    pub chat_id: String,   // UUID string or BIGINT string
    pub unread_count: i32,
    pub last_read_message_id: Option<Uuid>,
    pub is_pinned: bool,
    pub is_muted: bool,
    pub is_hidden: bool,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct ChatConversationStateRepository {
    pool: PgPool,
}

impl ChatConversationStateRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取单个会话状态
    pub async fn get_state(
        &self,
        user_id: i64,
        chat_type: &str,
        chat_id: &str,
    ) -> Result<Option<ChatConversationStateRow>> {
        let row = sqlx::query_as!(
            ChatConversationStateRow,
            r#"
            SELECT user_id as "user_id!", chat_type as "chat_type!", chat_id as "chat_id!",
                   unread_count as "unread_count!", last_read_message_id,
                   is_pinned as "is_pinned!", is_muted as "is_muted!", is_hidden as "is_hidden!",
                   updated_at as "updated_at!"
            FROM chat_conversations_state
            WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3
            "#,
            user_id,
            chat_type,
            chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 列出用户的所有活跃会话 (用于 Inbox 列表)
    pub async fn list_by_user(&self, user_id: i64) -> Result<Vec<ChatConversationStateRow>> {
        let rows = sqlx::query_as!(
            ChatConversationStateRow,
            r#"
            SELECT user_id as "user_id!", chat_type as "chat_type!", chat_id as "chat_id!",
                   unread_count as "unread_count!", last_read_message_id,
                   is_pinned as "is_pinned!", is_muted as "is_muted!", is_hidden as "is_hidden!",
                   updated_at as "updated_at!"
            FROM chat_conversations_state
            WHERE user_id = $1 AND is_hidden = FALSE
            ORDER BY is_pinned DESC, updated_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 标记已读 (重置未读数为 0)
    pub async fn mark_as_read(
        &self,
        user_id: i64,
        chat_type: &str,
        chat_id: &str,
        last_message_id: Uuid,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE chat_conversations_state
            SET unread_count = 0,
                last_read_message_id = $4,
                updated_at = NOW()
            WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3
            "#,
            user_id,
            chat_type,
            chat_id,
            last_message_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 设置置顶状态
    pub async fn set_pinned(&self, user_id: i64, chat_type: &str, chat_id: &str, pinned: bool) -> Result<()> {
        sqlx::query!(
            "UPDATE chat_conversations_state SET is_pinned = $4, updated_at = NOW() WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3",
            user_id, chat_type, chat_id, pinned
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
