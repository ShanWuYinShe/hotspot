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
        sqlx::query!(
            r#"
            INSERT INTO chat_conversations_state (
                user_id, chat_type, chat_id, unread_count, last_read_message_id, updated_at
            )
            VALUES ($1, $2, $3, 0, $4, NOW())
            ON CONFLICT (user_id, chat_type, chat_id)
            DO UPDATE SET
                unread_count = 0,
                last_read_message_id = COALESCE($4, chat_conversations_state.last_read_message_id),
                updated_at = NOW()
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

    /// 标记指定类型下当前用户所有会话为已读
    pub async fn mark_all_as_read(
        &self,
        chat_type: &str,
        user_id: i64,
    ) -> Result<(), AppError> {
        sqlx::query!(
            r#"
            UPDATE chat_conversations_state
            SET unread_count = 0, updated_at = NOW()
            WHERE user_id = $1 AND chat_type = $2
            "#,
            user_id,
            chat_type
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
        let row = sqlx::query!(
            r#"
            SELECT COALESCE(
                (SELECT unread_count
                 FROM chat_conversations_state
                 WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3),
                0
            ) as "count!"
            "#,
            user_id,
            chat_type,
            chat_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.count as i64)
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

        let rows = sqlx::query_as!(
            UnreadCountRow,
            r#"
            SELECT chat_id as "chat_id!", unread_count::bigint as "count!"
            FROM chat_conversations_state
            WHERE user_id = $1 AND chat_type = $2 AND chat_id = ANY($3)
            "#,
            user_id,
            chat_type,
            chat_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.chat_id, r.count)).collect())
    }
}
