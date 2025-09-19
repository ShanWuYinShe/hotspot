/// PostgreSQL消息仓库完整实现
/// 
/// 实现新架构的消息存储，支持私聊消息、群组消息、会话管理、消息反应、已读状态等完整功能
/// 基于新的数据库表结构：conversations, messages, conversation_participants, message_media, message_reactions, message_read_status

use crate::domain::{
    DirectMessage, GroupMessage, DomainError, DomainResult
};
use crate::domain::entities::{DirectMessageType, GroupMessageType};
use crate::application::queries::message_queries::{ConversationSummary, MessageView, UnreadMessageCount};
use sqlx::{PgPool, Row};
use uuid::Uuid;
use chrono::{DateTime, Utc};
use std::collections::HashMap;
use super::query_optimizer::PgQueryOptimizer;

/// 消息反应行结构体（保留，用于未来的消息反应功能）
#[derive(Debug)]
pub struct MessageReactionRow {
    pub id: i64,
    pub message_id: i64,
    pub user_id: i64,
    pub reaction_type: String,
    pub emoji: Option<String>,
    pub created_at: DateTime<Utc>,
}

// UserConversationRow 已删除，现在直接使用动态查询结果


#[derive(Clone)]
pub struct PgMessageRepository {
    pool: PgPool,
    query_optimizer: Option<PgQueryOptimizer>,
}

impl PgMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { 
            pool,
            query_optimizer: None,
        }
    }

    pub fn with_query_optimizer(pool: PgPool, query_optimizer: PgQueryOptimizer) -> Self {
        Self {
            pool,
            query_optimizer: Some(query_optimizer),
        }
    }

    /// 执行查询并可选地进行性能监控
    async fn execute_query_with_optional_monitoring<F, T>(
        &self,
        operation_name: &str,
        query_fn: F,
    ) -> DomainResult<T>
    where
        F: std::future::Future<Output = DomainResult<T>>,
    {
        if let Some(_optimizer) = &self.query_optimizer {
            let start_time = std::time::Instant::now();
            let result = query_fn.await;
            let duration = start_time.elapsed();
            
            // 记录查询性能
            if duration.as_millis() > 100 {
                tracing::warn!(
                    operation = operation_name,
                    duration_ms = duration.as_millis(),
                    "Slow message query detected"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = duration.as_millis(),
                    "Message query completed"
                );
            }
            
            result
        } else {
            query_fn.await
        }
    }

    // ===== 会话管理操作 =====
    
    /// 获取或创建私聊会话ID
    pub async fn get_or_create_conversation_id(&self, user1_id: i64, user2_id: i64) -> DomainResult<i64> {
        self.execute_query_with_optional_monitoring(
            "get_or_create_conversation_id",
            async {
                // 首先尝试查找现有会话
                let existing = sqlx::query!(
                    r#"
                    SELECT c.id
                    FROM conversations c
                    JOIN conversation_participants cp1 ON c.id = cp1.conversation_id AND cp1.user_id = $1
                    JOIN conversation_participants cp2 ON c.id = cp2.conversation_id AND cp2.user_id = $2
                    WHERE c.conversation_type = 'private' AND c.is_active = true
                    LIMIT 1
                    "#,
                    user1_id,
                    user2_id
                )
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                if let Some(row) = existing {
                    return Ok(row.id);
                }

                // 创建新会话
                let conversation_uuid = Uuid::new_v4().to_string();
                let conversation_row = sqlx::query!(
                    r#"
                    INSERT INTO conversations (conversation_id, conversation_type, participant_count, created_by)
                    VALUES ($1, 'private', 2, $2)
                    RETURNING id
                    "#,
                    conversation_uuid,
                    user1_id
                )
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                // 添加参与者
                sqlx::query!(
                    r#"
                    INSERT INTO conversation_participants (conversation_id, user_id, role, status)
                    VALUES ($1, $2, 'member', 'active'), ($1, $3, 'member', 'active')
                    "#,
                    conversation_row.id,
                    user1_id,
                    user2_id
                )
                .execute(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                Ok(conversation_row.id)
            }
        ).await
    }

    /// 获取群组对应的会话ID
    pub async fn get_group_conversation_id(&self, group_id: &str) -> DomainResult<i64> {
        let group_id_i64 = group_id.parse::<i64>()
            .map_err(|_| DomainError::BusinessRuleViolation { rule: "Invalid group ID format".to_string() })?;
        
        let row = sqlx::query!(
            r#"
            SELECT id FROM conversations 
            WHERE conversation_type = 'group' AND group_id = $1 AND is_active = true
            "#,
            group_id_i64
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        row.map(|r| r.id)
            .ok_or_else(|| DomainError::BusinessRuleViolation { 
                rule: "Group conversation not found".to_string() 
            })
    }

    /// 获取会话中的对方用户ID
    pub async fn get_conversation_recipient_id(&self, conversation_id: i64, sender_id: i64) -> DomainResult<i64> {
        let row = sqlx::query!(
            r#"
            SELECT user_id
            FROM conversation_participants 
            WHERE conversation_id = $1 AND user_id != $2 AND status = 'active'
            LIMIT 1
            "#,
            conversation_id,
            sender_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        row.map(|r| r.user_id)
            .ok_or_else(|| DomainError::BusinessRuleViolation { 
                rule: "Recipient not found in conversation".to_string() 
            })
    }

    /// 获取用户的会话列表
    pub async fn list_conversations(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<ConversationSummary>> {
        self.execute_query_with_optional_monitoring(
            "list_conversations",
            async {
                let rows = sqlx::query!(
                    r#"
                    SELECT c.conversation_id, c.conversation_type, c.name, c.last_message_at,
                           COALESCE(cp.unread_count, 0) as "unread_count!", c.participant_count,
                           CASE 
                               WHEN c.conversation_type = 'private' THEN 
                                   (SELECT ARRAY_AGG(cp2.user_id) FROM conversation_participants cp2 
                                    WHERE cp2.conversation_id = c.id AND cp2.status = 'active')
                               ELSE ARRAY[c.created_by]
                           END as "participant_ids!",
                           (SELECT content FROM messages m WHERE m.conversation_id = c.id 
                            AND m.is_deleted = false ORDER BY m.id DESC LIMIT 1) as last_message_content
                    FROM conversations c
                    JOIN conversation_participants cp ON c.id = cp.conversation_id
                    WHERE cp.user_id = $1 AND cp.status = 'active' AND c.is_active = true
                          AND ($2::bigint = 0 OR c.id < $2::bigint)
                    ORDER BY c.last_activity_at DESC, c.id DESC
                    LIMIT $3
                    "#,
                    user_id,
                    cursor,
                    limit as i64
                )
                .fetch_all(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let mut conversations = Vec::new();
                for row in rows {
                    conversations.push(ConversationSummary {
                        conversation_id: row.conversation_id,
                        participant_ids: row.participant_ids,
                        last_message_content: row.last_message_content,
                        last_message_at: row.last_message_at,
                        unread_count: row.unread_count as u64,
                    });
                }
                
                Ok(conversations)
            }
        ).await
    }

    // ===== 私聊消息操作 =====
    
    /// 根据ID查找私聊消息
    pub async fn find_direct_message_by_id(&self, message_id: i64) -> DomainResult<Option<DirectMessage>> {
        self.execute_query_with_optional_monitoring(
            "find_direct_message_by_id",
            async {
                let row = sqlx::query!(
                    r#"
                    SELECT m.id, m.message_id, m.conversation_id, 
                           m.sender_id, m.content, 
                           m.message_type::text as "message_type!", m.created_at as "created_at!",
                           m.is_deleted, m.is_edited, m.edited_at,
                           m.reply_to_message_id, m.thread_id,
                           m.reaction_count, m.reply_count, m.mentions,
                           mrs.read_at,
                           COALESCE(mm.latitude, 0.0) as "latitude!",
                           COALESCE(mm.longitude, 0.0) as "longitude!"
                    FROM messages m
                    LEFT JOIN message_read_status mrs ON m.id = mrs.message_id
                    LEFT JOIN message_media mm ON m.id = mm.message_id AND mm.media_type = 'location'
                    WHERE m.id = $1 AND m.is_deleted = false
                    "#,
                    message_id
                )
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                if let Some(r) = row {
                    // 获取对话参与者信息以确定recipient_id
                    let recipient_id = self.get_conversation_recipient_id(r.conversation_id, r.sender_id).await?;
                    
                    Ok(Some(DirectMessage {
                        id: r.id,
                        message_id: r.message_id,
                        conversation_id: r.conversation_id.to_string(),
                        sender_id: r.sender_id,
                        recipient_id,
                        content: r.content.unwrap_or_default(),
                        message_type: DirectMessageType::from_str(&r.message_type),
                        created_at: r.created_at,
                        read_at: r.read_at,
                        latitude: r.latitude,
                        longitude: r.longitude,
                    }))
                } else {
                    Ok(None)
                }
            }
        ).await
    }

    /// 创建私聊消息
    pub async fn create_direct_message(&self, message: &DirectMessage) -> DomainResult<DirectMessage> {
        self.execute_query_with_optional_monitoring(
            "create_direct_message",
            async {
                let message_uuid = Uuid::parse_str(&message.message_id)
                    .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid message UUID: {}", e) })?;
                
                // 获取或创建会话
                let conversation_id = self.get_or_create_conversation_id(message.sender_id, message.recipient_id).await?;
                    
                let row = sqlx::query!(
                    r#"
                    INSERT INTO messages (message_id, conversation_id, sender_id, 
                                         content, message_type, created_at)
                    VALUES ($1, $2, $3, $4, $5, $6)
                    RETURNING id, message_id, conversation_id, 
                              sender_id, content, 
                              message_type::text as "message_type!", created_at as "created_at!"
                    "#,
                    message_uuid.to_string(),
                    conversation_id,
                    message.sender_id,
                    message.content,
                    message.message_type.as_str() as &str,
                    message.created_at
                )
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                // 如果有位置信息，插入到message_media表
                if message.latitude != 0.0 || message.longitude != 0.0 {
                    let _ = sqlx::query!(
                        r#"
                        INSERT INTO message_media (message_id, media_type, file_url, latitude, longitude)
                        VALUES ($1, 'location', '', $2, $3)
                        "#,
                        row.id,
                        message.latitude,
                        message.longitude
                    )
                    .execute(&self.pool)
                    .await;
                }

                Ok(DirectMessage {
                    id: row.id,
                    message_id: row.message_id,
                    conversation_id: row.conversation_id.to_string(),
                    sender_id: row.sender_id,
                    recipient_id: message.recipient_id,
                    content: row.content.unwrap_or_default(),
                    message_type: DirectMessageType::from_str(&row.message_type),
                    created_at: row.created_at,
                    read_at: None,
                    latitude: message.latitude,
                    longitude: message.longitude,
                })
            }
        ).await
    }

    /// 删除私聊消息（软删除）
    pub async fn delete_direct_message(&self, message_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!(
            r#"
            UPDATE messages 
            SET is_deleted = true, deleted_at = CURRENT_TIMESTAMP
            WHERE id = $1 AND is_deleted = false
            "#,
            message_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .rows_affected();
        
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message not found or already deleted".to_string(),
            });
        }
        Ok(())
    }

    /// 获取私聊消息历史
    pub async fn find_direct_messages_by_conversation(
        &self,
        conversation_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<DirectMessage>> {
        self.execute_query_with_optional_monitoring(
            "find_direct_messages_by_conversation",
            async {
                // 首先通过conversation_id获取内部ID
                let conv_row = sqlx::query!(
                    r#"
                    SELECT id FROM conversations WHERE conversation_id = $1 AND is_active = true
                    "#,
                    conversation_id
                )
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let conv_internal_id = conv_row
                    .ok_or_else(|| DomainError::BusinessRuleViolation { rule: "Conversation not found".to_string() })?
                    .id;
                
                let rows = sqlx::query!(
                    r#"
                    SELECT m.id, m.message_id, m.conversation_id, 
                           m.sender_id, m.content, 
                           m.message_type::text as "message_type!", m.created_at as "created_at!",
                           m.is_deleted, m.is_edited, m.edited_at,
                           m.reply_to_message_id, m.thread_id,
                           m.reaction_count, m.reply_count, m.mentions,
                           mrs.read_at,
                           COALESCE(mm.latitude, 0.0) as "latitude!",
                           COALESCE(mm.longitude, 0.0) as "longitude!"
                    FROM messages m
                    LEFT JOIN message_read_status mrs ON m.id = mrs.message_id
                    LEFT JOIN message_media mm ON m.id = mm.message_id AND mm.media_type = 'location'
                    WHERE m.conversation_id = $1 AND m.is_deleted = false 
                          AND ($2::bigint = 0 OR m.id < $2::bigint)
                    ORDER BY m.id DESC
                    LIMIT $3
                    "#,
                    conv_internal_id,
                    cursor,
                    limit as i64
                )
                .fetch_all(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let mut messages = Vec::new();
                for row in rows {
                    // 获取对话参与者信息以确定recipient_id
                    let recipient_id = self.get_conversation_recipient_id(row.conversation_id, row.sender_id).await?;
                    
                    messages.push(DirectMessage {
                        id: row.id,
                        message_id: row.message_id,
                        conversation_id: conversation_id.to_string(),
                        sender_id: row.sender_id,
                        recipient_id,
                        content: row.content.unwrap_or_default(),
                        message_type: DirectMessageType::from_str(&row.message_type),
                        created_at: row.created_at,
                        read_at: row.read_at,
                        latitude: row.latitude,
                        longitude: row.longitude,
                    });
                }
                
                Ok(messages)
            }
        ).await
    }

    // ===== 群组消息操作 =====
    
    /// 根据ID查找群组消息
    pub async fn find_group_message_by_id(&self, message_id: i64) -> DomainResult<Option<GroupMessage>> {
        self.execute_query_with_optional_monitoring(
            "find_group_message_by_id",
            async {
                let row = sqlx::query!(
                    r#"
                    SELECT m.id, m.message_id, m.conversation_id, 
                           m.sender_id, m.content, 
                           m.message_type::text as "message_type!", m.created_at as "created_at!",
                           c.group_id,
                           COALESCE(mm.latitude, 0.0) as "latitude!",
                           COALESCE(mm.longitude, 0.0) as "longitude!"
                    FROM messages m
                    JOIN conversations c ON m.conversation_id = c.id
                    LEFT JOIN message_media mm ON m.id = mm.message_id AND mm.media_type = 'location'
                    WHERE m.id = $1 AND m.is_deleted = false AND c.conversation_type = 'group'
                    "#,
                    message_id
                )
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                if let Some(r) = row {
                    let group_id = r.group_id
                        .ok_or_else(|| DomainError::BusinessRuleViolation { rule: "Group ID not found".to_string() })?;
                    
                    Ok(Some(GroupMessage {
                        id: r.id,
                        message_id: r.message_id,
                        group_id: group_id.to_string(),
                        sender_id: r.sender_id,
                        content: r.content.unwrap_or_default(),
                        message_type: GroupMessageType::from_str(&r.message_type),
                        created_at: r.created_at,
                        latitude: r.latitude,
                        longitude: r.longitude,
                    }))
                } else {
                    Ok(None)
                }
            }
        ).await
    }

    /// 创建群组消息
    pub async fn create_group_message(&self, message: &GroupMessage) -> DomainResult<GroupMessage> {
        self.execute_query_with_optional_monitoring(
            "create_group_message",
            async {
                let message_uuid = Uuid::parse_str(&message.message_id)
                    .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid message UUID: {}", e) })?;
                
                // 获取群组对应的会话ID
                let conversation_id = self.get_group_conversation_id(&message.group_id).await?;
                    
                let row = sqlx::query!(
                    r#"
                    INSERT INTO messages (message_id, conversation_id, sender_id, 
                                         content, message_type, created_at)
                    VALUES ($1, $2, $3, $4, $5, $6)
                    RETURNING id, message_id, conversation_id, 
                              sender_id, content, 
                              message_type::text as "message_type!", created_at as "created_at!"
                    "#,
                    message_uuid.to_string(),
                    conversation_id,
                    message.sender_id,
                    message.content,
                    message.message_type.as_str() as &str,
                    message.created_at
                )
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                // 如果有位置信息，插入到message_media表
                if message.latitude != 0.0 || message.longitude != 0.0 {
                    let _ = sqlx::query!(
                        r#"
                        INSERT INTO message_media (message_id, media_type, file_url, latitude, longitude)
                        VALUES ($1, 'location', '', $2, $3)
                        "#,
                        row.id,
                        message.latitude,
                        message.longitude
                    )
                    .execute(&self.pool)
                    .await;
                }

                Ok(GroupMessage {
                    id: row.id,
                    message_id: row.message_id,
                    group_id: message.group_id.clone(),
                    sender_id: row.sender_id,
                    content: row.content.unwrap_or_default(),
                    message_type: GroupMessageType::from_str(&row.message_type),
                    created_at: row.created_at,
                    latitude: message.latitude,
                    longitude: message.longitude,
                })
            }
        ).await
    }

    /// 删除群组消息（软删除）
    pub async fn delete_group_message(&self, message_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!(
            r#"
            UPDATE messages 
            SET is_deleted = true, deleted_at = CURRENT_TIMESTAMP
            WHERE id = $1 AND is_deleted = false
            "#,
            message_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .rows_affected();
        
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message not found or already deleted".to_string(),
            });
        }
        Ok(())
    }

    /// 获取群组消息历史
    pub async fn find_group_messages(
        &self,
        group_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<GroupMessage>> {
        self.execute_query_with_optional_monitoring(
            "find_group_messages",
            async {
                // 获取群组对应的会话ID
                let conversation_id = self.get_group_conversation_id(group_id).await?;
                
                let rows = sqlx::query!(
                    r#"
                    SELECT m.id, m.message_id, m.conversation_id, 
                           m.sender_id, m.content, 
                           m.message_type::text as "message_type!", m.created_at as "created_at!",
                           COALESCE(mm.latitude, 0.0) as "latitude!",
                           COALESCE(mm.longitude, 0.0) as "longitude!"
                    FROM messages m
                    LEFT JOIN message_media mm ON m.id = mm.message_id AND mm.media_type = 'location'
                    WHERE m.conversation_id = $1 AND m.is_deleted = false 
                          AND ($2::bigint = 0 OR m.id < $2::bigint)
                    ORDER BY m.id DESC
                    LIMIT $3
                    "#,
                    conversation_id,
                    cursor,
                    limit as i64
                )
                .fetch_all(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let mut messages = Vec::new();
                for row in rows {
                    messages.push(GroupMessage {
                        id: row.id,
                        message_id: row.message_id,
                        group_id: group_id.to_string(),
                        sender_id: row.sender_id,
                        content: row.content.unwrap_or_default(),
                        message_type: GroupMessageType::from_str(&row.message_type),
                        created_at: row.created_at,
                        latitude: row.latitude,
                        longitude: row.longitude,
                    });
                }
                
                Ok(messages)
            }
        ).await
    }

    // ===== 消息已读状态操作 =====
    
    /// 标记消息为已读
    pub async fn mark_message_as_read(&self, message_id: i64, user_id: i64) -> DomainResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO message_read_status (message_id, user_id, read_at)
            VALUES ($1, $2, CURRENT_TIMESTAMP)
            ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = CURRENT_TIMESTAMP
            "#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 标记会话中所有消息为已读
    pub async fn mark_conversation_as_read(&self, conversation_id: &str, user_id: i64) -> DomainResult<()> {
        // 首先通过conversation_id获取内部ID
        let conv_row = sqlx::query!(
            r#"
            SELECT id FROM conversations WHERE conversation_id = $1 AND is_active = true
            "#,
            conversation_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let conv_internal_id = conv_row
            .ok_or_else(|| DomainError::BusinessRuleViolation { rule: "Conversation not found".to_string() })?
            .id;

        // 标记所有未读消息为已读
        sqlx::query!(
            r#"
            INSERT INTO message_read_status (message_id, user_id, read_at)
            SELECT m.id, $2, CURRENT_TIMESTAMP
            FROM messages m
            LEFT JOIN message_read_status mrs ON m.id = mrs.message_id AND mrs.user_id = $2
            WHERE m.conversation_id = $1 AND m.is_deleted = false AND mrs.id IS NULL
            "#,
            conv_internal_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        // 更新会话参与者的未读计数
        sqlx::query!(
            r#"
            UPDATE conversation_participants 
            SET unread_count = 0, last_read_at = CURRENT_TIMESTAMP
            WHERE conversation_id = $1 AND user_id = $2
            "#,
            conv_internal_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 获取用户的未读消息数量
    pub async fn get_unread_message_count(&self, user_id: i64) -> DomainResult<UnreadMessageCount> {
        self.execute_query_with_optional_monitoring(
            "get_unread_message_count",
            async {
                let rows = sqlx::query!(
                    r#"
                    SELECT c.conversation_id, COALESCE(cp.unread_count, 0) as "unread_count!"
                    FROM conversation_participants cp
                    JOIN conversations c ON cp.conversation_id = c.id
                    WHERE cp.user_id = $1 AND cp.status = 'active' AND c.is_active = true
                    "#,
                    user_id
                )
                .fetch_all(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let mut by_conversation = HashMap::new();
                let mut total_count = 0u64;

                for row in rows {
                    let count = row.unread_count as u64;
                    by_conversation.insert(row.conversation_id, count);
                    total_count += count;
                }

                Ok(UnreadMessageCount {
                    total_count,
                    by_conversation,
                })
            }
        ).await
    }

    // ===== 消息反应操作 =====
    
    /// 添加消息反应
    pub async fn add_message_reaction(&self, message_id: i64, user_id: i64, reaction_type: &str, emoji: Option<&str>) -> DomainResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO message_reactions (message_id, user_id, reaction_type, emoji)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (message_id, user_id) DO UPDATE SET 
                reaction_type = EXCLUDED.reaction_type,
                emoji = EXCLUDED.emoji,
                created_at = CURRENT_TIMESTAMP
            "#,
            message_id,
            user_id,
            reaction_type,
            emoji
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_message_reaction(&self, message_id: i64, user_id: i64) -> DomainResult<()> {
        sqlx::query!(
            r#"
            DELETE FROM message_reactions 
            WHERE message_id = $1 AND user_id = $2
            "#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 获取消息的所有反应
    pub async fn get_message_reactions(&self, message_id: i64) -> DomainResult<Vec<MessageReactionRow>> {
        let rows = sqlx::query_as!(
            MessageReactionRow,
            r#"
            SELECT id, message_id, user_id, reaction_type, emoji, created_at as "created_at!"
            FROM message_reactions 
            WHERE message_id = $1
            ORDER BY created_at ASC
            "#,
            message_id
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(rows)
    }

    // ===== 消息搜索操作 =====
    
    /// 搜索消息
    pub async fn search_messages(
        &self,
        user_id: i64,
        keyword: &str,
        conversation_id: Option<&str>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<MessageView>> {
        self.execute_query_with_optional_monitoring(
            "search_messages",
            async {
                let conversation_filter = if let Some(conv_id) = conversation_id {
                    // 获取会话内部ID
                    let conv_row = sqlx::query!(
                        r#"SELECT id FROM conversations WHERE conversation_id = $1"#,
                        conv_id
                    )
                    .fetch_optional(&self.pool)
                    .await
                    .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
                    
                    conv_row.map(|r| r.id)
                } else {
                    None
                };

                // 构建基础查询
                let base_query = r#"
                    SELECT m.message_id, m.sender_id, u.nickname as sender_nickname,
                           m.content, m.message_type::text as message_type, m.created_at as "created_at!",
                           CASE WHEN mrs.read_at IS NOT NULL THEN true ELSE false END as is_read
                    FROM messages m
                    JOIN users u ON m.sender_id = u.id
                    JOIN conversation_participants cp ON m.conversation_id = cp.conversation_id
                    LEFT JOIN message_read_status mrs ON m.id = mrs.message_id AND mrs.user_id = $1
                    WHERE cp.user_id = $1 AND cp.status = 'active'
                          AND m.is_deleted = false
                          AND m.content ILIKE $2
                "#;

                let rows = if let Some(conv_id) = conversation_filter {
                    let query_with_conv = format!("{} AND m.conversation_id = $3 AND ($4::bigint = 0 OR m.id < $4::bigint) ORDER BY m.id DESC LIMIT $5", base_query);
                    sqlx::query(&query_with_conv)
                        .bind(user_id)
                        .bind(format!("%{}%", keyword))
                        .bind(conv_id)
                        .bind(cursor)
                        .bind(limit as i64)
                        .fetch_all(&self.pool)
                        .await
                } else {
                    let query_without_conv = format!("{} AND ($3::bigint = 0 OR m.id < $3::bigint) ORDER BY m.id DESC LIMIT $4", base_query);
                    sqlx::query(&query_without_conv)
                        .bind(user_id)
                        .bind(format!("%{}%", keyword))
                        .bind(cursor)
                        .bind(limit as i64)
                        .fetch_all(&self.pool)
                        .await
                };

                let rows = rows.map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                let mut messages = Vec::new();
                for row in rows {
                    let message_id: Uuid = row.get("message_id");
                    let sender_id: i64 = row.get("sender_id");
                    let sender_nickname: String = row.get("sender_nickname");
                    let content: String = row.get("content");
                    let message_type_str: String = row.get("message_type");
                    let created_at: DateTime<Utc> = row.get("created_at");
                    let is_read: bool = row.get("is_read");

                    messages.push(MessageView {
                        message_id: message_id.to_string(),
                        sender_id,
                        sender_nickname,
                        content,
                        message_type: message_type_str,
                        created_at,
                        is_read,
                    });
                }
                
                Ok(messages)
            }
        ).await
    }
    
    /// 标记私聊消息已读
    pub async fn mark_direct_messages_read(
        &self,
        user_id: i64,
        conversation_id: &str,
        message_id: Option<&str>,
    ) -> DomainResult<()> {
        // 获取会话ID
        let conv_id: i64 = sqlx::query_scalar(
            "SELECT id FROM conversations WHERE conversation_id = $1"
        )
        .bind(conversation_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Conversation not found".to_string(),
        })?;

        if let Some(msg_id) = message_id {
            // 标记特定消息已读
            // 1. 插入或更新 message_read_status
            sqlx::query(
                r#"
                INSERT INTO message_read_status (message_id, user_id, read_at)
                SELECT m.id, $2, CURRENT_TIMESTAMP
                FROM messages m
                WHERE m.conversation_id = $1 AND m.message_id = $3
                ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = CURRENT_TIMESTAMP
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .bind(msg_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            // 2. 更新 conversation_participants 中的 last_read_message_id
            sqlx::query(
                r#"
                UPDATE conversation_participants 
                SET last_read_message_id = (
                    SELECT id FROM messages WHERE conversation_id = $1 AND message_id = $3
                ), last_read_at = CURRENT_TIMESTAMP
                WHERE conversation_id = $1 AND user_id = $2
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .bind(msg_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
        } else {
            // 标记会话中所有消息已读
            // 1. 插入所有未读消息的已读状态
            sqlx::query(
                r#"
                INSERT INTO message_read_status (message_id, user_id, read_at)
                SELECT m.id, $2, CURRENT_TIMESTAMP
                FROM messages m
                LEFT JOIN message_read_status mrs ON m.id = mrs.message_id AND mrs.user_id = $2
                WHERE m.conversation_id = $1 AND m.sender_id != $2 AND mrs.id IS NULL
                ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = CURRENT_TIMESTAMP
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            // 2. 更新 conversation_participants 中的未读数量和最后已读消息
            sqlx::query(
                r#"
                UPDATE conversation_participants 
                SET unread_count = 0,
                    last_read_message_id = (
                        SELECT id FROM messages 
                        WHERE conversation_id = $1 
                        ORDER BY created_at DESC 
                        LIMIT 1
                    ),
                    last_read_at = CURRENT_TIMESTAMP
                WHERE conversation_id = $1 AND user_id = $2
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
        }
        
        Ok(())
    }
    
    /// 获取私聊未读消息数
    pub async fn get_direct_unread_count(
        &self,
        user_id: i64,
        conversation_id: Option<&str>,
    ) -> DomainResult<i32> {
        let count: i64 = if let Some(conv_id) = conversation_id {
            // 获取特定会话的未读消息数
            sqlx::query_scalar(
                r#"
                SELECT COALESCE(cp.unread_count, 0)
                FROM conversation_participants cp
                JOIN conversations c ON c.id = cp.conversation_id
                WHERE c.conversation_id = $1 AND cp.user_id = $2 AND c.conversation_type = 'private'
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .unwrap_or(0)
        } else {
            // 获取用户所有私聊未读消息数
            sqlx::query_scalar(
                r#"
                SELECT COALESCE(SUM(cp.unread_count), 0)
                FROM conversation_participants cp
                JOIN conversations c ON c.id = cp.conversation_id
                WHERE cp.user_id = $1 AND c.conversation_type = 'private' AND cp.status = 'active'
                "#
            )
            .bind(user_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
        };
        
        Ok(count as i32)
    }
    
    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        &self,
        user_id: i64,
        group_id: &str,
        message_id: Option<&str>,
    ) -> DomainResult<()> {
        // 获取群组会话ID
        let conv_id: i64 = sqlx::query_scalar(
            r#"
            SELECT c.id 
            FROM conversations c
            JOIN groups g ON g.id = c.group_id
            WHERE g.group_id = $1 AND c.conversation_type = 'group'
            "#
        )
        .bind(group_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group conversation not found".to_string(),
        })?;

        if let Some(msg_id) = message_id {
            // 标记特定消息已读
            sqlx::query(
                r#"
                INSERT INTO message_read_status (message_id, user_id, read_at)
                SELECT m.id, $2, CURRENT_TIMESTAMP
                FROM messages m
                WHERE m.conversation_id = $1 AND m.message_id = $3
                ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = CURRENT_TIMESTAMP
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .bind(msg_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            // 更新 conversation_participants 中的 last_read_message_id
            sqlx::query(
                r#"
                UPDATE conversation_participants 
                SET last_read_message_id = (
                    SELECT id FROM messages WHERE conversation_id = $1 AND message_id = $3
                ), last_read_at = CURRENT_TIMESTAMP
                WHERE conversation_id = $1 AND user_id = $2
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .bind(msg_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
        } else {
            // 标记群组中所有消息已读
            sqlx::query(
                r#"
                INSERT INTO message_read_status (message_id, user_id, read_at)
                SELECT m.id, $2, CURRENT_TIMESTAMP
                FROM messages m
                LEFT JOIN message_read_status mrs ON m.id = mrs.message_id AND mrs.user_id = $2
                WHERE m.conversation_id = $1 AND m.sender_id != $2 AND mrs.id IS NULL
                ON CONFLICT (message_id, user_id) DO UPDATE SET read_at = CURRENT_TIMESTAMP
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            // 更新 conversation_participants 中的未读数量和最后已读消息
            sqlx::query(
                r#"
                UPDATE conversation_participants 
                SET unread_count = 0,
                    last_read_message_id = (
                        SELECT id FROM messages 
                        WHERE conversation_id = $1 
                        ORDER BY created_at DESC 
                        LIMIT 1
                    ),
                    last_read_at = CURRENT_TIMESTAMP
                WHERE conversation_id = $1 AND user_id = $2
                "#
            )
            .bind(conv_id)
            .bind(user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
        }
        
        Ok(())
    }
    
    /// 获取群组未读消息数
    pub async fn get_group_unread_count(
        &self,
        user_id: i64,
        group_id: Option<&str>,
    ) -> DomainResult<i32> {
        let count: i64 = if let Some(grp_id) = group_id {
            // 获取特定群组的未读消息数
            sqlx::query_scalar(
                r#"
                SELECT COALESCE(cp.unread_count, 0)
                FROM conversation_participants cp
                JOIN conversations c ON c.id = cp.conversation_id
                JOIN groups g ON g.id = c.group_id
                WHERE g.group_id = $1 AND cp.user_id = $2 AND c.conversation_type = 'group'
                "#
            )
            .bind(grp_id)
            .bind(user_id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .unwrap_or(0)
        } else {
            // 获取用户所有群组未读消息数
            sqlx::query_scalar(
                r#"
                SELECT COALESCE(SUM(cp.unread_count), 0)
                FROM conversation_participants cp
                JOIN conversations c ON c.id = cp.conversation_id
                WHERE cp.user_id = $1 AND c.conversation_type = 'group' AND cp.status = 'active'
                "#
            )
            .bind(user_id)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
        };
        
        Ok(count as i32)
    }
    
    /// 获取用户会话列表
    pub async fn find_conversations_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<crate::domain::Conversation>> {
        let rows = sqlx::query(
            r#"
            SELECT 
                c.id,
                c.conversation_id,
                c.conversation_type,
                c.name,
                c.description,
                c.avatar_url,
                c.is_active,
                c.message_count,
                c.participant_count,
                c.last_message_at,
                c.created_at,
                c.updated_at,
                cp.unread_count,
                cp.last_read_at,
                -- 获取最后一条消息的预览
                (SELECT content FROM messages m 
                 WHERE m.conversation_id = c.id 
                 ORDER BY m.created_at DESC 
                 LIMIT 1) as last_message_preview,
                -- 对于私聊，获取对方用户信息
                CASE 
                    WHEN c.conversation_type = 'private' THEN
                        (SELECT u.nickname FROM users u 
                         JOIN conversation_participants cp2 ON cp2.user_id = u.id
                         WHERE cp2.conversation_id = c.id AND cp2.user_id != $1
                         LIMIT 1)
                    ELSE c.name
                END as display_name
            FROM conversations c
            JOIN conversation_participants cp ON cp.conversation_id = c.id
            WHERE cp.user_id = $1 
                AND cp.status = 'active'
                AND c.is_active = true
                AND ($2 = 0 OR c.id < $2)
            ORDER BY c.last_activity_at DESC, c.id DESC
            LIMIT $3
            "#
        )
        .bind(user_id)
        .bind(cursor)
        .bind(limit)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        
        let mut conversations = Vec::new();
        for row in rows {
            // 为了兼容现有的 Conversation 结构，我们需要适配数据
            // 注意：这里需要根据实际的 Conversation 结构进行调整
            let id: i64 = row.get("id");
            let conversation_id: String = row.get("conversation_id");
            let created_at: DateTime<Utc> = row.get("created_at");
            let updated_at: DateTime<Utc> = row.get("updated_at");
            let last_message_at: Option<DateTime<Utc>> = row.get("last_message_at");
            let last_message_preview: Option<String> = row.get("last_message_preview");
            
            conversations.push(crate::domain::Conversation {
                id,
                conversation_id,
                // 对于兼容性，我们暂时使用占位符值
                // 实际应用中应该根据 conversation_type 来设置这些字段
                user1_id: user_id, // 当前用户
                user2_id: user_id, // 占位符，实际应该是对方用户ID（仅对私聊有效）
                created_at,
                updated_at,
                last_message_at: last_message_at.unwrap_or(created_at), // 如果没有消息，使用创建时间
                last_message_preview,
            });
        }
        
        Ok(conversations)
    }
}