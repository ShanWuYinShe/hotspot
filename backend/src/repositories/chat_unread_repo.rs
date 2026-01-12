// src/repositories/chat_unread_repo.rs
// 聊天未读计数Repository（统一管理私聊和群聊的未读计数）

use crate::error::AppError;
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, sqlx::FromRow)]
struct UnreadCountRow {
    chat_id: String,
    count: i64,
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
            chat_id,
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
        // 获取最后已读时间
        let record = sqlx::query!(
            r#"
            SELECT last_read_at
            FROM chat_unread_counts
            WHERE chat_type = $1 AND chat_id = $2 AND user_id = $3
            "#,
            chat_type,
            chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        // 如果没有读取记录，使用 Unix 纪元作为默认值（统计所有消息）
        let read_at = record
            .and_then(|r| r.last_read_at)
            .unwrap_or(chrono::DateTime::UNIX_EPOCH);

        // 根据聊天类型查询未读消息数
        let count = match chat_type {
            "private_chat" => {
                // private_messages uses chat_id (BIGINT)
                let chat_id_i64 = chat_id.parse::<i64>().unwrap_or(0);
                sqlx::query_scalar!(
                    r#"
                    SELECT COUNT(*) as "count!"
                    FROM private_messages
                    WHERE chat_id = $1 
                      AND sender_id != $2 
                      AND sent_at > $3
                      AND is_deleted = FALSE
                    "#,
                    chat_id_i64,
                    user_id,
                    read_at
                )
                .fetch_one(&self.pool)
                .await?
            }
            "group_chat" => {
                // group_messages uses group_chat_id (UUID)
                let chat_uuid = Uuid::parse_str(chat_id).unwrap_or_default();
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

    /// 批量获取未读计数 (返回 Map<chat_id_str, count>)
    /// chat_ids: 统一传入 String 类型 ID
    pub async fn get_unread_count_batch(
        &self,
        chat_type: &str,
        chat_ids: &[String],
        user_id: i64,
    ) -> Result<std::collections::HashMap<String, i64>, AppError> {
        if chat_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = match chat_type {
            "private_chat" => {
                // private_messages uses chat_id (BIGINT)
                let chat_ids_i64: Vec<i64> = chat_ids.iter()
                    .filter_map(|id| id.parse::<i64>().ok())
                    .collect();

                if chat_ids_i64.is_empty() {
                    return Ok(std::collections::HashMap::new());
                }

                sqlx::query_as!(
                    UnreadCountRow,
                    r#"
                    SELECT CAST(m.chat_id AS TEXT) as "chat_id!", COUNT(*) as "count!"
                    FROM private_messages m
                    LEFT JOIN chat_unread_counts c 
                        ON c.chat_id = CAST(m.chat_id AS TEXT)
                        AND c.user_id = $2 
                        AND c.chat_type = 'private_chat'
                    WHERE m.chat_id = ANY($1)
                      AND m.sender_id != $2
                      AND m.is_deleted = FALSE
                      AND m.sent_at > COALESCE(c.last_read_at, '1970-01-01')
                    GROUP BY m.chat_id
                    "#,
                    &chat_ids_i64,
                    user_id
                )
                .fetch_all(&self.pool)
                .await?
            }
            "group_chat" => {
                // group_messages uses group_chat_id (UUID)
                let chat_uuids: Vec<Uuid> = chat_ids.iter()
                    .filter_map(|id| Uuid::parse_str(id).ok())
                    .collect();

                if chat_uuids.is_empty() {
                    return Ok(std::collections::HashMap::new());
                }

                sqlx::query_as!(
                    UnreadCountRow,
                    r#"
                    SELECT CAST(m.group_chat_id AS TEXT) as "chat_id!", COUNT(*) as "count!"
                    FROM group_messages m
                    LEFT JOIN chat_unread_counts c 
                        ON c.chat_id = CAST(m.group_chat_id AS TEXT)
                        AND c.user_id = $2 
                        AND c.chat_type = 'group_chat'
                    WHERE m.group_chat_id = ANY($1)
                      AND m.sender_id != $2
                      AND m.is_deleted = FALSE
                      AND m.sent_at > COALESCE(c.last_read_at, '1970-01-01')
                    GROUP BY m.group_chat_id
                    "#,
                    &chat_uuids,
                    user_id
                )
                .fetch_all(&self.pool)
                .await?
            }
            _ => return Ok(std::collections::HashMap::new()),
        };

        Ok(rows.into_iter().map(|r| (r.chat_id, r.count)).collect())
    }
}
