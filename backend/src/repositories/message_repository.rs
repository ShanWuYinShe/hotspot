//! 消息相关数据仓库实现

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use tracing::error;
use utoipa::ToSchema;
use uuid::Uuid;

use crate::repositories::{Repository, RepositoryError, RepositoryResult};

// =============== 查询结果结构体定义 ===============

/// 直接消息查询结果
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct MessageRow {
    pub message_id: Uuid,
    pub conversation_id: Uuid,
    pub sender_id: i64,
    pub recipient_id: i64,
    pub content: String,
    pub message_type: String,
    pub created_at: DateTime<Utc>,
    pub read_at: Option<DateTime<Utc>>,
    pub latitude: f64,
    pub longitude: f64,
}

/// 群组消息查询结果
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct GroupMessageRow {
    pub message_id: Uuid,
    pub group_id: Uuid,
    pub sender_id: i64,
    pub content: String,
    pub message_type: String,
    pub created_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
}

/// 会话查询结果
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct ConversationRow {
    pub conversation_id: Uuid,
    pub user1_id: i64,
    pub user2_id: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_message_at: DateTime<Utc>,
    pub last_message_preview: Option<String>,
}

/// 群组消息查询结果（带ID用于分页）
#[derive(Debug, Clone)]
pub struct GroupMessageWithIdRow {
    pub message_id: Uuid,
    pub group_id: Uuid,
    pub sender_id: i64,
    pub content: String,
    pub message_type: String,
    pub created_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub id: i64,
}

/// PostgreSQL实现的消息仓库
pub struct PgMessageRepository {
    pool: PgPool,
}

impl PgMessageRepository {
    /// 创建新的PostgreSQL消息仓库
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

impl Repository for PgMessageRepository {
    fn name(&self) -> &str {
        "pg_message_repository"
    }
}

impl PgMessageRepository {
    

    pub async fn check_user_in_conversation(
        &self,
        conversation_id: Uuid,
        user_id: i64,
    ) -> RepositoryResult<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM conversations 
                WHERE conversation_id = $1 AND (user1_id = $2 OR user2_id = $2)
            ) as "exists!""#,
            conversation_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查用户是否在私信会话失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(exists)
    }

    pub async fn check_user_in_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> RepositoryResult<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members 
                WHERE group_id = $1 AND user_id = $2
            ) as "exists!""#,
            group_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查用户是否在群组失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(exists)
    }

    

    pub async fn get_other_user_in_conversation(
        &self,
        conversation_id: Uuid,
        current_user_id: i64,
    ) -> RepositoryResult<i64> {
        let user1_id = sqlx::query_scalar!(
            r#"SELECT user1_id FROM conversations WHERE conversation_id = $1"#,
            conversation_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取私信会话信息失败: {:?}", e);
            RepositoryError::NotFound
        })?;

        let user2_id = sqlx::query_scalar!(
            r#"SELECT user2_id FROM conversations WHERE conversation_id = $1"#,
            conversation_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取私信会话信息失败: {:?}", e);
            RepositoryError::NotFound
        })?;

        let other_user_id = if user1_id == current_user_id {
            user2_id
        } else {
            user1_id
        };

        Ok(other_user_id)
    }

    

    pub async fn save_direct_message(
        &self,
        conversation_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        content: &str,
        content_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> RepositoryResult<Uuid> {
        let message_id = Uuid::new_v4();
        let now = Utc::now();

        // 开始事务
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 插入消息
        sqlx::query!(
            r#"
            INSERT INTO direct_messages (message_id, conversation_id, sender_id, recipient_id, content, message_type, created_at, latitude, longitude)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            "#,
            message_id,
            conversation_id,
            sender_id,
            recipient_id,
            content,
            content_type,
            now,
            latitude,
            longitude
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("保存私信消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 更新会话的最后消息时间和预览
        let preview = if content.len() > 100 {
            format!("{}...", &content[..100])
        } else {
            content.to_string()
        };

        sqlx::query!(
            r#"
            UPDATE conversations 
            SET last_message_at = $1, last_message_preview = $2, updated_at = $1
            WHERE conversation_id = $3
            "#,
            now,
            preview,
            conversation_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("更新会话最后消息时间失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 提交事务
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(message_id)
    }

    pub async fn save_group_message(
        &self,
        group_id: Uuid,
        sender_id: i64,
        content: &str,
        message_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> RepositoryResult<Uuid> {
        let message_id = Uuid::new_v4();
        let now = Utc::now();
        sqlx::query!(
            r#"
            INSERT INTO group_messages (message_id, group_id, sender_id, content, message_type, created_at, latitude, longitude)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            "#,
            message_id,
            group_id,
            sender_id,
            content,
            message_type,
            now,
            latitude,
            longitude
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("保存群聊消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(message_id)
    }

    pub async fn get_direct_message_by_id(&self, message_id: Uuid) -> RepositoryResult<MessageRow> {
        let row = sqlx::query_as!(
            MessageRow,
            r#"
            SELECT 
                message_id as "message_id!",
                conversation_id as "conversation_id!",
                sender_id as "sender_id!",
                recipient_id as "recipient_id!",
                content as "content!",
                message_type as "message_type!",
                created_at as "created_at!",
                read_at,
                latitude as "latitude!: f64",
                longitude as "longitude!: f64"
            FROM direct_messages
            WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取消息详情失败: {:?}", e);
            match e {
                sqlx::Error::RowNotFound => RepositoryError::NotFound,
                _ => RepositoryError::Database(e),
            }
        })?;

        Ok(row)
    }

    pub async fn get_group_message_by_id(
        &self,
        message_id: Uuid,
    ) -> RepositoryResult<GroupMessageRow> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT 
                message_id as "message_id!",
                group_id as "group_id!",
                sender_id as "sender_id!",
                content as "content!",
                message_type as "message_type!",
                created_at as "created_at!",
                latitude as "latitude!: f64",
                longitude as "longitude!: f64"
            FROM group_messages
            WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取消息详情失败: {:?}", e);
            match e {
                sqlx::Error::RowNotFound => RepositoryError::NotFound,
                _ => RepositoryError::Database(e),
            }
        })?;

        Ok(row)
    }

    pub async fn get_or_create_conversation(
        &self,
        user1_id: i64,
        user2_id: i64,
    ) -> RepositoryResult<Uuid> {
        let (smaller_id, larger_id) = if user1_id < user2_id {
            (user1_id, user2_id)
        } else {
            (user2_id, user1_id)
        };
        let existing_conv = sqlx::query_scalar!(
            r#"
            SELECT conversation_id FROM conversations 
            WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1)
            "#,
            smaller_id,
            larger_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| {
            error!("查询会话失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        if let Some(conv) = existing_conv {
            return Ok(conv);
        }

        let conversation_id = Uuid::new_v4();
        let now = Utc::now();
        sqlx::query!(
            r#"
            INSERT INTO conversations (conversation_id, user1_id, user2_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $4)
            "#,
            conversation_id,
            smaller_id,
            larger_id,
            now
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("插入会话失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(conversation_id)
    }

    pub async fn get_conversations(&self, user_id: i64) -> RepositoryResult<Vec<ConversationRow>> {
        let conversations = sqlx::query_as!(
            ConversationRow,
            r#"
            SELECT 
                conversation_id as "conversation_id!",
                user1_id as "user1_id!",
                user2_id as "user2_id!",
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_message_at as "last_message_at!",
                last_message_preview
            FROM conversations
            WHERE user1_id = $1 OR user2_id = $1
            ORDER BY last_message_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取会话列表失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(conversations)
    }
    pub async fn get_direct_messages(
        &self,
        conversation_id: Uuid,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> RepositoryResult<(Vec<MessageRow>, bool, Option<i64>)> {
        // 验证用户是否在会话中
        let is_member = self
            .check_user_in_conversation(conversation_id, user_id)
            .await?;
        if !is_member {
            return Err(RepositoryError::Other("User not in conversation".into()));
        }

        let rows = sqlx::query_as!(
            MessageRow,
            r#"
            SELECT 
                message_id as "message_id!",
                conversation_id as "conversation_id!",
                sender_id as "sender_id!",
                recipient_id as "recipient_id!",
                content as "content!",
                message_type as "message_type!",
                created_at as "created_at!",
                read_at,
                latitude as "latitude!: f64",
                longitude as "longitude!: f64"
            FROM direct_messages
            WHERE conversation_id = $1
            AND ($2::bigint = 0 OR id < $2::bigint)
            ORDER BY id DESC
            LIMIT $3
            "#,
            conversation_id,
            cursor as i32,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取私信消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        // 标记消息为已读
        sqlx::query!(
            r#"
            UPDATE direct_messages
            SET read_at = CURRENT_TIMESTAMP
            WHERE conversation_id = $1
            AND recipient_id = $2
            AND read_at IS NULL
            "#,
            conversation_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("标记已读失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        let has_more = rows.len() > limit as usize;
        let messages = if has_more {
            rows[..limit as usize].to_vec()
        } else {
            rows
        };

        let next_cursor = if has_more && !messages.is_empty() {
            // 取最后一条的 id 作为游标
            let last_msg = messages.last().unwrap();
            // 需要查 id 字段
            sqlx::query_scalar!(
                "SELECT id FROM direct_messages WHERE message_id = $1",
                last_msg.message_id
            )
            .fetch_one(&self.pool)
            .await
            .ok()
        } else {
            None
        };

        Ok((messages, has_more, next_cursor))
    }

    pub async fn get_group_messages(
        &self,
        group_id: Uuid,
        _user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> RepositoryResult<(Vec<GroupMessageRow>, bool, Option<i64>)> {
        let messages = sqlx::query_as!(
            GroupMessageWithIdRow,
            r#"
            SELECT 
                message_id as "message_id!",
                group_id as "group_id!",
                sender_id as "sender_id!",
                content as "content!",
                message_type as "message_type!",
                created_at as "created_at!",
                latitude as "latitude!: f64",
                longitude as "longitude!: f64",
                id as "id!: i64"
            FROM group_messages
            WHERE group_id = $1 
            AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            group_id,
            cursor as i32,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取群组消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        let has_more = messages.len() > limit as usize;
        let messages_limited = if has_more {
            &messages[..limit as usize]
        } else {
            &messages
        };

        let result_messages = messages_limited
            .iter()
            .map(|row| GroupMessageRow {
                message_id: row.message_id,
                group_id: row.group_id,
                sender_id: row.sender_id,
                content: row.content.clone(),
                message_type: row.message_type.clone(),
                created_at: row.created_at,
                latitude: row.latitude,
                longitude: row.longitude,
            })
            .collect::<Vec<_>>();

        // 使用最后一条消息的 id 作为游标
        let next_cursor = if has_more && !messages_limited.is_empty() {
            Some(messages_limited.last().unwrap().id)
        } else {
            None
        };

        Ok((result_messages, has_more, next_cursor))
    }

    pub async fn mark_direct_messages_read(
        &self,
        conversation_id: Uuid,
        user_id: i64,
        message_id: Option<Uuid>,
    ) -> RepositoryResult<()> {
        // 验证用户是否在会话中
        let is_member = self
            .check_user_in_conversation(conversation_id, user_id)
            .await?;
        if !is_member {
            return Err(RepositoryError::Other("User not in conversation".into()));
        }

        let target_message_sequence_id = if let Some(msg_id) = message_id {
            // 如果指定了消息ID，获取该消息的序列ID
            let sequence_id = sqlx::query_scalar!(
                "SELECT id FROM direct_messages WHERE message_id = $1 AND conversation_id = $2",
                msg_id,
                conversation_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("获取消息序列ID失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            match sequence_id {
                Some(id) => id,
                None => return Err(RepositoryError::NotFound),
            }
        } else {
            // 如果没有指定消息ID，获取会话中最新消息的序列ID
            let latest_id = sqlx::query_scalar!(
                "SELECT id FROM direct_messages WHERE conversation_id = $1 ORDER BY id DESC LIMIT 1",
                conversation_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("获取最新消息序列ID失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            match latest_id {
                Some(id) => id,
                None => return Ok(()), // 会话中没有消息
            }
        };

        // 更新或插入已读状态
        sqlx::query!(
            r#"
            INSERT INTO user_message_read_status (user_id, conversation_id, last_read_message_id, updated_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id, conversation_id)
            DO UPDATE SET 
                last_read_message_id = GREATEST(user_message_read_status.last_read_message_id, $3),
                updated_at = NOW()
            "#,
            user_id,
            conversation_id,
            target_message_sequence_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("更新私聊消息已读状态失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(())
    }

    pub async fn delete_group_message(
        &self,
        message_id: Uuid,
        user_id: i64,
    ) -> RepositoryResult<()> {
        let result = sqlx::query!(
            r#"DELETE FROM group_messages WHERE message_id = $1 AND sender_id = $2"#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("删除群组消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        if result.rows_affected() == 0 {
            return Err(RepositoryError::NotFound);
        }

        Ok(())
    }

    pub async fn delete_direct_message(
        &self,
        message_id: Uuid,
        user_id: i64,
    ) -> RepositoryResult<()> {
        let result = sqlx::query!(
            r#"DELETE FROM direct_messages WHERE message_id = $1 AND sender_id = $2"#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("删除私聊消息失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        if result.rows_affected() == 0 {
            return Err(RepositoryError::NotFound);
        }

        Ok(())
    }

    

    

    pub async fn get_direct_message_unread_count(&self, user_id: i64) -> RepositoryResult<i64> {
        let total_unread = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(SUM(
                CASE 
                    WHEN urs.last_read_message_id IS NULL THEN (
                        SELECT COUNT(*) FROM direct_messages dm2 
                        WHERE dm2.conversation_id = c.conversation_id 
                        AND dm2.recipient_id = $1
                    )
                    ELSE (
                        SELECT COUNT(*) FROM direct_messages dm2 
                        WHERE dm2.conversation_id = c.conversation_id 
                        AND dm2.id > urs.last_read_message_id
                        AND dm2.recipient_id = $1
                    )
                END
            ), 0) as unread_count
            FROM conversations c
            LEFT JOIN user_message_read_status urs ON (
                urs.user_id = $1 
                AND urs.conversation_id = c.conversation_id
            )
            WHERE c.user1_id = $1 OR c.user2_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取私聊未读消息数量失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(total_unread.unwrap_or(sqlx::types::BigDecimal::from(0)).to_string().parse::<i64>().unwrap_or(0))
    }

    /// 标记群组消息已读 - 基于最后已读消息ID的高效实现
    pub async fn mark_group_messages_read(
        &self,
        group_id: Uuid,
        user_id: i64,
        message_id: Option<Uuid>,
    ) -> RepositoryResult<()> {
        // 验证用户是否在群组中
        let is_member = self.check_user_in_group(group_id, user_id).await?;
        if !is_member {
            return Err(RepositoryError::Other("User not in group".into()));
        }

        let target_message_sequence_id = if let Some(msg_id) = message_id {
            // 如果指定了消息ID，获取该消息的序列ID
            let sequence_id = sqlx::query_scalar!(
                "SELECT id FROM group_messages WHERE message_id = $1 AND group_id = $2",
                msg_id,
                group_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("获取消息序列ID失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            match sequence_id {
                Some(id) => id,
                None => return Err(RepositoryError::NotFound),
            }
        } else {
            // 如果没有指定消息ID，获取群组中最新消息的序列ID
            let latest_id = sqlx::query_scalar!(
                "SELECT id FROM group_messages WHERE group_id = $1 ORDER BY id DESC LIMIT 1",
                group_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                error!("获取最新消息序列ID失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            match latest_id {
                Some(id) => id,
                None => return Ok(()), // 群组中没有消息
            }
        };

        // 更新或插入已读状态
        sqlx::query!(
            r#"
            INSERT INTO user_message_read_status (user_id, group_id, last_read_message_id, updated_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id, group_id)
            DO UPDATE SET 
                last_read_message_id = GREATEST(user_message_read_status.last_read_message_id, $3),
                updated_at = NOW()
            "#,
            user_id,
            group_id,
            target_message_sequence_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("更新群组消息已读状态失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(())
    }

    /// 获取群组未读消息数量 - 基于最后已读消息ID的高效实现
    pub async fn get_group_messages_unread_count(&self, user_id: i64) -> RepositoryResult<i64> {
        let total_unread = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(SUM(
                CASE 
                    WHEN urs.last_read_message_id IS NULL THEN (
                        SELECT COUNT(*) FROM group_messages gm2 
                        WHERE gm2.group_id = gme.group_id 
                        AND gm2.sender_id != $1
                    )
                    ELSE (
                        SELECT COUNT(*) FROM group_messages gm2 
                        WHERE gm2.group_id = gme.group_id 
                        AND gm2.id > urs.last_read_message_id
                        AND gm2.sender_id != $1
                    )
                END
            ), 0) as unread_count
            FROM group_members gme
            LEFT JOIN user_message_read_status urs ON (
                urs.user_id = gme.user_id 
                AND urs.group_id = gme.group_id
            )
            WHERE gme.user_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取群组未读消息数量失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(total_unread.unwrap_or(sqlx::types::BigDecimal::from(0)).to_string().parse::<i64>().unwrap_or(0))
    }
}
