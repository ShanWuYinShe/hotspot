use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::message::Message,
};

pub struct MessageRepository<'a> {
    db: &'a PgPool,
}

impl<'a> MessageRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建消息
    pub async fn create(&self, message: &Message) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO messages (
                message_id, sender_id, target_id, target_type, content, 
                message_type, reply_to, metadata, status
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING message_id
            "#,
            message.message_id,
            message.sender_id,
            message.target_id,
            message.target_type,
            message.content,
            message.message_type,
            message.reply_to,
            message.metadata,
            message.status,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.message_id)
    }

    /// 根据ID查找消息
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<Message>> {
        let message = sqlx::query_as!(
            Message,
            r#"
            SELECT * FROM messages WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(message)
    }

    /// 更新消息内容
    pub async fn update_content(&self, message_id: Uuid, content: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET content = $2, edited_at = NOW()
            WHERE message_id = $1
            "#,
            message_id,
            content
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 更新消息状态
    pub async fn update_status(&self, message_id: Uuid, status: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET status = $2
            WHERE message_id = $1
            "#,
            message_id,
            status
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid) -> Result<()> {
        self.update_status(message_id, "recalled").await
    }

    /// 删除消息（软删除）
    pub async fn soft_delete(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET deleted_at = NOW()
            WHERE message_id = $1
            "#,
            message_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取单聊消息历史
    pub async fn list_user_messages(
        &self,
        user_id: i64,
        other_user_id: i64,
        limit: i64,
        before_timestamp: Option<i64>,
    ) -> Result<Vec<Message>> {
        let before_dt = before_timestamp.map(|ts| {
            chrono::DateTime::from_timestamp(ts, 0)
                .unwrap_or_default()
                .naive_utc()
        });

        let messages = if let Some(before) = before_dt {
            sqlx::query_as!(
                Message,
                r#"
                SELECT * FROM messages
                WHERE 
                    target_type = 'user'
                    AND (
                        (sender_id = $1 AND target_id = $2::text)
                        OR (sender_id = $2 AND target_id = $1::text)
                    )
                    AND created_at < $3
                    AND deleted_at IS NULL
                ORDER BY created_at DESC
                LIMIT $4
                "#,
                user_id,
                other_user_id,
                before,
                limit
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        } else {
            sqlx::query_as!(
                Message,
                r#"
                SELECT * FROM messages
                WHERE 
                    target_type = 'user'
                    AND (
                        (sender_id = $1 AND target_id = $2::text)
                        OR (sender_id = $2 AND target_id = $1::text)
                    )
                    AND deleted_at IS NULL
                ORDER BY created_at DESC
                LIMIT $3
                "#,
                user_id,
                other_user_id,
                limit
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        };

        Ok(messages)
    }

    /// 获取群聊消息历史
    pub async fn list_group_messages(
        &self,
        group_id: Uuid,
        limit: i64,
        before_timestamp: Option<i64>,
    ) -> Result<Vec<Message>> {
        let before_dt = before_timestamp.map(|ts| {
            chrono::DateTime::from_timestamp(ts, 0)
                .unwrap_or_default()
                .naive_utc()
        });

        let messages = if let Some(before) = before_dt {
            sqlx::query_as!(
                Message,
                r#"
                SELECT * FROM messages
                WHERE 
                    target_type = 'group'
                    AND target_id = $1
                    AND created_at < $2
                    AND deleted_at IS NULL
                ORDER BY created_at DESC
                LIMIT $3
                "#,
                group_id.to_string(),
                before,
                limit
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        } else {
            sqlx::query_as!(
                Message,
                r#"
                SELECT * FROM messages
                WHERE 
                    target_type = 'group'
                    AND target_id = $1
                    AND deleted_at IS NULL
                ORDER BY created_at DESC
                LIMIT $2
                "#,
                group_id.to_string(),
                limit
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        };

        Ok(messages)
    }

    /// 标记消息已读
    pub async fn mark_as_read(
        &self,
        user_id: i64,
        message_id: Uuid,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO message_read_records (message_id, user_id, read_at)
            VALUES ($1, $2, NOW())
            ON CONFLICT (message_id, user_id) DO NOTHING
            "#,
            message_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 批量标记已读
    pub async fn mark_multiple_as_read(
        &self,
        user_id: i64,
        message_ids: &[Uuid],
    ) -> Result<()> {
        for message_id in message_ids {
            self.mark_as_read(user_id, *message_id).await?;
        }
        Ok(())
    }

    /// 添加消息反应
    pub async fn add_reaction(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO message_reactions (message_id, user_id, reaction)
            VALUES ($1, $2, $3)
            ON CONFLICT (message_id, user_id, reaction) DO NOTHING
            "#,
            message_id,
            user_id,
            reaction
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM message_reactions
            WHERE message_id = $1 AND user_id = $2 AND reaction = $3
            "#,
            message_id,
            user_id,
            reaction
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取未读消息数量（单聊）
    pub async fn count_unread_user_messages(
        &self,
        user_id: i64,
        other_user_id: i64,
    ) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM messages m
            WHERE 
                m.sender_id = $2
                AND m.target_type = 'user'
                AND m.target_id = $1::text
                AND m.deleted_at IS NULL
                AND NOT EXISTS (
                    SELECT 1 FROM message_read_records r
                    WHERE r.message_id = m.message_id AND r.user_id = $1
                )
            "#,
            user_id,
            other_user_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }

    /// 获取未读消息数量（群聊）
    pub async fn count_unread_group_messages(
        &self,
        user_id: i64,
        group_id: Uuid,
    ) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM messages m
            WHERE 
                m.target_type = 'group'
                AND m.target_id = $2
                AND m.sender_id != $1
                AND m.deleted_at IS NULL
                AND NOT EXISTS (
                    SELECT 1 FROM message_read_records r
                    WHERE r.message_id = m.message_id AND r.user_id = $1
                )
            "#,
            user_id,
            group_id.to_string()
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }
}
