//! 聊天未读计数 Repository — 统一管理私聊和群聊的未读计数

use crate::error::ApiError;
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, sqlx::FromRow)]
struct UnreadCountRow {
    chat_id: String,
    count: i64,
}

#[derive(Debug, sqlx::FromRow)]
struct ChatSettingsRow {
    chat_id: String,
    is_muted: bool,
    is_pinned: bool,
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
    ) -> Result<(), ApiError> {
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
    pub async fn mark_all_as_read(&self, chat_type: &str, user_id: i64) -> Result<(), ApiError> {
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

    /// 更新私聊会话设置 (is_muted, is_pinned)
    pub async fn update_private_chat_settings(
        &self,
        user_id: i64,
        chat_id: &str,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
    ) -> Result<(), ApiError> {
        sqlx::query!(
            r#"
            INSERT INTO chat_conversations_state (
                user_id, chat_type, chat_id, unread_count, is_muted, is_pinned, updated_at
            )
            VALUES ($1, 'private', $2, 0, COALESCE($3, FALSE), COALESCE($4, FALSE), NOW())
            ON CONFLICT (user_id, chat_type, chat_id)
            DO UPDATE SET
                is_muted = COALESCE($3, chat_conversations_state.is_muted),
                is_pinned = COALESCE($4, chat_conversations_state.is_pinned),
                updated_at = NOW()
            "#,
            user_id,
            chat_id,
            is_muted,
            is_pinned
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 隐藏私聊会话（单边）：用于会话删除后从 Inbox 移除
    pub async fn hide_private_chat(&self, user_id: i64, chat_id: &str) -> Result<(), ApiError> {
        sqlx::query!(
            r#"
            INSERT INTO chat_conversations_state (
                user_id, chat_type, chat_id, unread_count, is_hidden, updated_at
            )
            VALUES ($1, 'private', $2, 0, TRUE, NOW())
            ON CONFLICT (user_id, chat_type, chat_id)
            DO UPDATE SET
                unread_count = 0,
                is_hidden = TRUE,
                updated_at = NOW()
            "#,
            user_id,
            chat_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 取消隐藏私聊会话（单边）：当用户再次发送/收到新消息时恢复可见
    pub async fn unhide_private_chat(&self, user_id: i64, chat_id: &str) -> Result<(), ApiError> {
        sqlx::query!(
            r#"
            INSERT INTO chat_conversations_state (
                user_id, chat_type, chat_id, unread_count, is_hidden, updated_at
            )
            VALUES ($1, 'private', $2, 0, FALSE, NOW())
            ON CONFLICT (user_id, chat_type, chat_id)
            DO UPDATE SET
                is_hidden = FALSE,
                updated_at = NOW()
            "#,
            user_id,
            chat_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 获取指定用户在某个私聊中的最后已读消息 ID
    pub async fn get_last_read_message_id(
        &self,
        user_id: i64,
        chat_id: &str,
    ) -> Result<Option<Uuid>, ApiError> {
        let row = sqlx::query!(
            r#"
            SELECT last_read_message_id
            FROM chat_conversations_state
            WHERE user_id = $1 AND chat_type = 'private' AND chat_id = $2
            "#,
            user_id,
            chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.and_then(|r| r.last_read_message_id))
    }

    /// 获取未读计数
    pub async fn get_unread_count(
        &self,
        chat_type: &str,
        chat_id: &str,
        user_id: i64,
    ) -> Result<i64, ApiError> {
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
    ) -> Result<std::collections::HashMap<String, i64>, ApiError> {
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

    /// 批量获取私聊会话设置 (is_muted, is_pinned)
    pub async fn get_private_chat_settings_batch(
        &self,
        chat_ids: &[String],
        user_id: i64,
    ) -> Result<std::collections::HashMap<String, (bool, bool)>, ApiError> {
        if chat_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query_as!(
            ChatSettingsRow,
            r#"
            SELECT chat_id as "chat_id!", is_muted as "is_muted!", is_pinned as "is_pinned!"
            FROM chat_conversations_state
            WHERE user_id = $1 AND chat_type = 'private' AND chat_id = ANY($2)
            "#,
            user_id,
            chat_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.chat_id, (r.is_muted, r.is_pinned)))
            .collect())
    }

    /// 获取单个私聊会话设置
    pub async fn get_private_chat_settings(
        &self,
        chat_id: &str,
        user_id: i64,
    ) -> Result<(bool, bool), ApiError> {
        let row = sqlx::query!(
            r#"
            SELECT is_muted as "is_muted!", is_pinned as "is_pinned!"
            FROM chat_conversations_state
            WHERE user_id = $1 AND chat_type = 'private' AND chat_id = $2
            "#,
            user_id,
            chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row
            .map(|r| (r.is_muted, r.is_pinned))
            .unwrap_or((false, false)))
    }
}
