use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated};

/// 私聊Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatRow {
    pub chat_id: Uuid,
    pub user1_id: i64,
    pub user2_id: i64,
    pub user1_cleared_at: Option<DateTime<Utc>>,
    pub user2_cleared_at: Option<DateTime<Utc>>,
    pub last_message_id: Option<Uuid>,
    pub last_message_content: Option<String>,
    pub last_message_type: Option<String>,
    pub last_message_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
struct PrivateChatListQueryRow {
    pub chat_id: Uuid,
    pub user1_id: i64,
    pub user2_id: i64,
    pub user1_cleared_at: Option<DateTime<Utc>>,
    pub user2_cleared_at: Option<DateTime<Utc>>,
    pub last_message_id: Option<Uuid>,
    pub last_message_content: Option<String>,
    pub last_message_type: Option<String>,
    pub last_message_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub sort_is_pinned: i32,
    pub sort_activity_at: DateTime<Utc>,
    pub sort_other_user_id: i64,
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
            RETURNING chat_id, user1_id, user2_id, user1_cleared_at, user2_cleared_at,
                      last_message_id, last_message_content, last_message_type,
                      last_message_at, created_at, updated_at
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
            SELECT chat_id, user1_id, user2_id, user1_cleared_at, user2_cleared_at,
                   last_message_id, last_message_content, last_message_type,
                   last_message_at, created_at, updated_at
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
    ) -> Paginated<PrivateChatRow> {
        type PrivateChatCursor = (i32, DateTime<Utc>, i64);
        let cursor_value = cursor
            .as_deref()
            .and_then(|c| decode_cursor::<PrivateChatCursor>(c).ok());

        let mut rows = if let Some((cursor_pinned, cursor_activity_at, cursor_other_user_id)) =
            cursor_value
        {
            sqlx::query_as!(
                PrivateChatListQueryRow,
                r#"
                SELECT
                    pc.chat_id as "chat_id!", pc.user1_id as "user1_id!", pc.user2_id as "user2_id!",
                    pc.user1_cleared_at, pc.user2_cleared_at,
                    pc.last_message_id, pc.last_message_content, pc.last_message_type,
                    pc.last_message_at, pc.created_at as "created_at!", pc.updated_at as "updated_at!",
                    COALESCE(ccs.is_pinned, FALSE)::int as "sort_is_pinned!",
                    COALESCE(pc.last_message_at, pc.updated_at) as "sort_activity_at!",
                    (CASE WHEN $1 = pc.user1_id THEN pc.user2_id ELSE pc.user1_id END) as "sort_other_user_id!"
                FROM private_chats pc
                LEFT JOIN chat_conversations_state ccs
                  ON ccs.user_id = $1
                 AND ccs.chat_type = 'private'
                 AND ccs.chat_id = pc.chat_id::text
                WHERE (pc.user1_id = $1 OR pc.user2_id = $1)
                  AND COALESCE(ccs.is_hidden, FALSE) = FALSE
                  AND (
                    (CASE WHEN $1 = pc.user1_id THEN pc.user1_cleared_at ELSE pc.user2_cleared_at END) IS NULL
                    OR (
                        pc.last_message_at IS NOT NULL
                        AND pc.last_message_at > (CASE WHEN $1 = pc.user1_id THEN pc.user1_cleared_at ELSE pc.user2_cleared_at END)
                    )
                  )
                  AND (
                    COALESCE(ccs.is_pinned, FALSE)::int < $2
                    OR (
                      COALESCE(ccs.is_pinned, FALSE)::int = $2
                      AND (
                        COALESCE(pc.last_message_at, pc.updated_at) < $3
                        OR (
                          COALESCE(pc.last_message_at, pc.updated_at) = $3
                          AND (CASE WHEN $1 = pc.user1_id THEN pc.user2_id ELSE pc.user1_id END) > $4
                        )
                      )
                    )
                  )
                ORDER BY
                    COALESCE(ccs.is_pinned, FALSE) DESC,
                    COALESCE(pc.last_message_at, pc.updated_at) DESC,
                    (CASE WHEN $1 = pc.user1_id THEN pc.user2_id ELSE pc.user1_id END) ASC
                LIMIT $5
                "#,
                user_id,
                cursor_pinned,
                cursor_activity_at,
                cursor_other_user_id,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                PrivateChatListQueryRow,
                r#"
                SELECT
                    pc.chat_id as "chat_id!", pc.user1_id as "user1_id!", pc.user2_id as "user2_id!",
                    pc.user1_cleared_at, pc.user2_cleared_at,
                    pc.last_message_id, pc.last_message_content, pc.last_message_type,
                    pc.last_message_at, pc.created_at as "created_at!", pc.updated_at as "updated_at!",
                    COALESCE(ccs.is_pinned, FALSE)::int as "sort_is_pinned!",
                    COALESCE(pc.last_message_at, pc.updated_at) as "sort_activity_at!",
                    (CASE WHEN $1 = pc.user1_id THEN pc.user2_id ELSE pc.user1_id END) as "sort_other_user_id!"
                FROM private_chats pc
                LEFT JOIN chat_conversations_state ccs
                  ON ccs.user_id = $1
                 AND ccs.chat_type = 'private'
                 AND ccs.chat_id = pc.chat_id::text
                WHERE (pc.user1_id = $1 OR pc.user2_id = $1)
                  AND COALESCE(ccs.is_hidden, FALSE) = FALSE
                  AND (
                    (CASE WHEN $1 = pc.user1_id THEN pc.user1_cleared_at ELSE pc.user2_cleared_at END) IS NULL
                    OR (
                        pc.last_message_at IS NOT NULL
                        AND pc.last_message_at > (CASE WHEN $1 = pc.user1_id THEN pc.user1_cleared_at ELSE pc.user2_cleared_at END)
                    )
                  )
                ORDER BY
                    COALESCE(ccs.is_pinned, FALSE) DESC,
                    COALESCE(pc.last_message_at, pc.updated_at) DESC,
                    (CASE WHEN $1 = pc.user1_id THEN pc.user2_id ELSE pc.user1_id END) ASC
                LIMIT $2
                "#,
                user_id,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        };

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let items: Vec<PrivateChatRow> = rows
            .iter()
            .map(|r| PrivateChatRow {
                chat_id: r.chat_id,
                user1_id: r.user1_id,
                user2_id: r.user2_id,
                user1_cleared_at: r.user1_cleared_at,
                user2_cleared_at: r.user2_cleared_at,
                last_message_id: r.last_message_id,
                last_message_content: r.last_message_content.clone(),
                last_message_type: r.last_message_type.clone(),
                last_message_at: r.last_message_at,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect();

        let next_cursor = if has_more {
            items
                .last()
                .and_then(|last_item| {
                    rows.iter()
                        .find(|r| r.chat_id == last_item.chat_id)
                        .map(|r| {
                            encode_cursor(&(
                                r.sort_is_pinned,
                                r.sort_activity_at,
                                r.sort_other_user_id,
                            ))
                        })
                })
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 单边清空会话：仅写入当前用户的 clear timeline，不影响对方
    pub async fn mark_cleared_by_user(&self, chat_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_chats
            SET
                user1_cleared_at = CASE WHEN user1_id = $2 THEN NOW() ELSE user1_cleared_at END,
                user2_cleared_at = CASE WHEN user2_id = $2 THEN NOW() ELSE user2_cleared_at END,
                updated_at = NOW()
            WHERE chat_id = $1
              AND ($2 = user1_id OR $2 = user2_id)
            "#,
            chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
