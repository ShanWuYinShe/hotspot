use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 私聊Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatRow {
    pub id: i64,
    pub private_chat_id: Uuid,
    pub user_a_id: i64,
    pub user_b_id: i64,
    // 陌生人消息限制字段
    pub user_a_can_message_freely: bool,
    pub user_b_can_message_freely: bool,
    pub user_a_sent_count: i32,
    pub user_b_sent_count: i32,
    pub created_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 私聊设置Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatSettingsRow {
    pub private_chat_id: Uuid,
    pub user_id: i64,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_archived: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct PrivateChatRepository {
    pool: PgPool,
}

impl PrivateChatRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM private_chats WHERE private_chat_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 创建私聊
    /// 注意：user_a_id 和 user_b_id 会自动排序以满足数据库约束 (user_a_id < user_b_id)
    pub async fn create(&self, user1_id: i64, user2_id: i64) -> Result<PrivateChatRow> {
        // 确保 user_a_id < user_b_id 以满足数据库约束
        let (user_a_id, user_b_id) = if user1_id < user2_id {
            (user1_id, user2_id)
        } else {
            (user2_id, user1_id)
        };

        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            INSERT INTO private_chats (user_a_id, user_b_id)
            VALUES ($1, $2)
            RETURNING id, private_chat_id, user_a_id, user_b_id,
                user_a_can_message_freely, user_b_can_message_freely,
                user_a_sent_count, user_b_sent_count,
                created_at, is_deleted, deleted_at
            "#,
            user_a_id,
            user_b_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 查找私聊（按两个用户ID）
    pub async fn find_by_users(&self, user_a_id: i64, user_b_id: i64) -> Result<Option<PrivateChatRow>> {
        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, private_chat_id, user_a_id, user_b_id,
                user_a_can_message_freely, user_b_can_message_freely,
                user_a_sent_count, user_b_sent_count,
                created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE ((user_a_id = $1 AND user_b_id = $2) OR (user_a_id = $2 AND user_b_id = $1))
              AND is_deleted = FALSE
            "#,
            user_a_id,
            user_b_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找
    pub async fn find_by_id(&self, private_chat_id: Uuid) -> Result<Option<PrivateChatRow>> {
        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, private_chat_id, user_a_id, user_b_id,
                user_a_can_message_freely, user_b_can_message_freely,
                user_a_sent_count, user_b_sent_count,
                created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE private_chat_id = $1 AND is_deleted = FALSE
            "#,
            private_chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 软删除私聊
    pub async fn soft_delete(&self, private_chat_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_chats
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE private_chat_id = $1
            "#,
            private_chat_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 检查用户是否在私聊中
    pub async fn is_user_in_chat(&self, private_chat_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
            SELECT 1 FROM private_chats
                WHERE private_chat_id = $1 
                  AND (user_a_id = $2 OR user_b_id = $2)
                  AND is_deleted = FALSE
            ) as "exists!"
            "#,
            private_chat_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 获取用户的所有私聊列表
    pub async fn list_by_user(&self, user_id: i64, limit: i64, cursor: Option<Uuid>) -> Result<Vec<PrivateChatRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, private_chat_id, user_a_id, user_b_id,
                user_a_can_message_freely, user_b_can_message_freely,
                user_a_sent_count, user_b_sent_count,
                created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE (user_a_id = $1 OR user_b_id = $1) AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }
    
    /// 获取私聊设置
    pub async fn get_settings(&self, private_chat_id: Uuid, user_id: i64) -> Result<Option<PrivateChatSettingsRow>> {
        let row = sqlx::query_as!(
            PrivateChatSettingsRow,
            r#"
            SELECT private_chat_id, user_id, is_muted, is_pinned, is_archived, created_at, updated_at
            FROM private_chat_settings
            WHERE private_chat_id = $1 AND user_id = $2
            "#,
            private_chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;
        
        Ok(row)
    }
    
    /// 创建或更新私聊设置
    pub async fn upsert_settings(
        &self, 
        private_chat_id: Uuid, 
        user_id: i64, 
        is_muted: Option<bool>, 
        is_pinned: Option<bool>, 
        is_archived: Option<bool>
    ) -> Result<()> {
        // 使用 ON CONFLICT 实现 upsert，避免 if-else 分支
        sqlx::query!(
            r#"
            INSERT INTO private_chat_settings (private_chat_id, user_id, is_muted, is_pinned, is_archived)
            VALUES ($1, $2, COALESCE($3, FALSE), COALESCE($4, FALSE), COALESCE($5, FALSE))
            ON CONFLICT (private_chat_id, user_id) DO UPDATE
            SET is_muted = COALESCE($3, private_chat_settings.is_muted),
                is_pinned = COALESCE($4, private_chat_settings.is_pinned),
                is_archived = COALESCE($5, private_chat_settings.is_archived),
                updated_at = NOW()
            "#,
            private_chat_id,
            user_id,
            is_muted,
            is_pinned,
            is_archived
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }

    /// 获取用户的消息限制信息
    /// 返回 (can_message_freely, sent_count)
    pub async fn get_message_limit(&self, private_chat_id: Uuid, user_id: i64) -> Result<(bool, i32)> {
        let chat = self.find_by_id(private_chat_id).await?.ok_or_else(|| {
            crate::error::AppError::NotFound("私聊不存在".to_string())
        })?;
        
        if chat.user_a_id == user_id {
            Ok((chat.user_a_can_message_freely, chat.user_a_sent_count))
        } else if chat.user_b_id == user_id {
            Ok((chat.user_b_can_message_freely, chat.user_b_sent_count))
        } else {
            Err(crate::error::AppError::Forbidden("用户不在此私聊中".to_string()))
        }
    }

    /// 增加用户发送消息计数
    pub async fn increment_sent_count(&self, private_chat_id: Uuid, user_id: i64) -> Result<i32> {
        let chat = self.find_by_id(private_chat_id).await?.ok_or_else(|| {
            crate::error::AppError::NotFound("私聊不存在".to_string())
        })?;
        
        if chat.user_a_id == user_id {
            let new_count = chat.user_a_sent_count + 1;
            sqlx::query!(
                r#"UPDATE private_chats SET user_a_sent_count = $1 WHERE private_chat_id = $2"#,
                new_count,
                private_chat_id
            )
            .execute(&self.pool)
            .await?;
            Ok(new_count)
        } else if chat.user_b_id == user_id {
            let new_count = chat.user_b_sent_count + 1;
            sqlx::query!(
                r#"UPDATE private_chats SET user_b_sent_count = $1 WHERE private_chat_id = $2"#,
                new_count,
                private_chat_id
            )
            .execute(&self.pool)
            .await?;
            Ok(new_count)
        } else {
            Err(crate::error::AppError::Forbidden("用户不在此私聊中".to_string()))
        }
    }

    /// 解锁用户消息限制（对方回复后调用）
    pub async fn unlock_messaging(&self, private_chat_id: Uuid, user_id: i64) -> Result<()> {
        let chat = self.find_by_id(private_chat_id).await?.ok_or_else(|| {
            crate::error::AppError::NotFound("私聊不存在".to_string())
        })?;
        
        if chat.user_a_id == user_id {
            sqlx::query!(
                r#"UPDATE private_chats SET user_a_can_message_freely = TRUE WHERE private_chat_id = $1"#,
                private_chat_id
            )
            .execute(&self.pool)
            .await?;
        } else if chat.user_b_id == user_id {
            sqlx::query!(
                r#"UPDATE private_chats SET user_b_can_message_freely = TRUE WHERE private_chat_id = $1"#,
                private_chat_id
            )
            .execute(&self.pool)
            .await?;
        } else {
            return Err(crate::error::AppError::Forbidden("用户不在此私聊中".to_string()));
        }
        
        Ok(())
    }
}
