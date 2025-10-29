use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::conversation::Conversation,
};

pub struct ConversationRepository<'a> {
    db: &'a PgPool,
}

impl<'a> ConversationRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建或更新会话
    pub async fn upsert(&self, conversation: &Conversation) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO conversations (
                user_id, target_id, target_type, last_message_id, 
                last_message_at, unread_count, is_pinned, is_muted
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            ON CONFLICT (user_id, target_id, target_type)
            DO UPDATE SET
                last_message_id = EXCLUDED.last_message_id,
                last_message_at = EXCLUDED.last_message_at,
                unread_count = EXCLUDED.unread_count
            RETURNING id
            "#,
            conversation.user_id,
            conversation.target_id,
            conversation.target_type,
            conversation.last_message_id,
            conversation.last_message_at,
            conversation.unread_count,
            conversation.is_pinned,
            conversation.is_muted,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.id)
    }

    /// 查找会话
    pub async fn find_by_user_and_target(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<Option<Conversation>> {
        let conversation = sqlx::query_as!(
            Conversation,
            r#"
            SELECT * FROM conversations
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(conversation)
    }

    /// 获取用户的会话列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Conversation>> {
        let conversations = sqlx::query_as!(
            Conversation,
            r#"
            SELECT * FROM conversations
            WHERE user_id = $1
            ORDER BY is_pinned DESC, last_message_at DESC NULLS LAST
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(conversations)
    }

    /// 更新未读计数
    pub async fn update_unread_count(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
        count: i32,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE conversations
            SET unread_count = $4
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type,
            count
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 清空未读计数
    pub async fn clear_unread_count(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        self.update_unread_count(user_id, target_id, target_type, 0).await
    }

    /// 置顶会话
    pub async fn pin_conversation(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE conversations
            SET is_pinned = true
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 取消置顶
    pub async fn unpin_conversation(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE conversations
            SET is_pinned = false
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 静音会话
    pub async fn mute_conversation(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE conversations
            SET is_muted = true
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 取消静音
    pub async fn unmute_conversation(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE conversations
            SET is_muted = false
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 删除会话
    pub async fn delete_conversation(
        &self,
        user_id: i64,
        target_id: &str,
        target_type: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM conversations
            WHERE user_id = $1 AND target_id = $2 AND target_type = $3
            "#,
            user_id,
            target_id,
            target_type
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取总未读数
    pub async fn get_total_unread_count(&self, user_id: i64) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COALESCE(SUM(unread_count), 0) as "total!"
            FROM conversations
            WHERE user_id = $1 AND is_muted = false
            "#,
            user_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.total)
    }
}
