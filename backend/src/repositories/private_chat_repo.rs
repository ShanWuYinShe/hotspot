use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// 私聊Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatRow {
    pub chat_id: Uuid,
    pub user1_id: i64,
    pub user2_id: i64,
    pub last_message_id: Option<Uuid>,
    pub last_message_content: Option<String>,
    pub last_message_type: Option<String>,
    pub last_message_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct PrivateChatRepository {
    pool: PgPool,
}

impl PrivateChatRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建或获取已存在的私聊
    pub async fn get_or_create(&self, u1: i64, u2: i64) -> Result<PrivateChatRow> {
        let (id1, id2) = if u1 < u2 { (u1, u2) } else { (u2, u1) };

        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            INSERT INTO private_chats (user1_id, user2_id)
            VALUES ($1, $2)
            ON CONFLICT (user1_id, user2_id) DO UPDATE SET updated_at = NOW()
            RETURNING chat_id, user1_id, user2_id, last_message_id, last_message_content, 
                      last_message_type, last_message_at, created_at, updated_at
            "#,
            id1,
            id2
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按双方用户查找
    pub async fn find_by_users(&self, u1: i64, u2: i64) -> Result<Option<PrivateChatRow>> {
        let (id1, id2) = if u1 < u2 { (u1, u2) } else { (u2, u1) };

        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT chat_id, user1_id, user2_id, last_message_id, last_message_content, 
                   last_message_type, last_message_at, created_at, updated_at
            FROM private_chats
            WHERE user1_id = $1 AND user2_id = $2
            "#,
            id1,
            id2
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 列出用户的私聊会话
    /// 这里的"列表"通常是从 Inbox (chat_conversations_state) 查询
    /// 但也可以直接从 private_chats 查询 (如果只想看私聊)
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateChatRow>, Option<String>)> {
        let last_ts = if let Some(c) = cursor {
            decode_cursor::<DateTime<Utc>>(&c)?
        } else {
            DateTime::<Utc>::MAX_UTC
        };

        let rows = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT chat_id, user1_id, user2_id, last_message_id, last_message_content, 
                   last_message_type, last_message_at, created_at, updated_at
            FROM private_chats
            WHERE (user1_id = $1 OR user2_id = $1)
              AND updated_at < $2
            ORDER BY updated_at DESC
            LIMIT $3
            "#,
            user_id,
            last_ts,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&r.updated_at))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
