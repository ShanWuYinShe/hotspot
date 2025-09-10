/// PostgreSQL消息仓库实现
/// 
/// 实现新架构的消息存储，支持私聊消息、群组消息和会话管理
use crate::domain::{
    Conversation, DirectMessage, DirectMessageType, DomainError, DomainResult, 
    GroupEventRecord, GroupMessage, GroupMessageType
};
use crate::application::queries::message_queries::ConversationSummary;
use sqlx::PgPool;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use super::query_optimizer::PgQueryOptimizer;

/// 私聊消息查询结果行结构体
#[derive(Debug)]
struct DirectMessageRow {
    pub id: i64,
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

    // ===== 私聊消息操作 =====
    
    /// 根据ID查找私聊消息
    pub async fn find_direct_message_by_id(&self, message_id: i64) -> DomainResult<Option<DirectMessage>> {
        self.execute_query_with_optional_monitoring(
            "find_direct_message_by_id",
            async {
                let row = sqlx::query_as!(
                    DirectMessageRow,
                    r#"
                    SELECT id as "id!", message_id as "message_id!", conversation_id as "conversation_id!", 
                           sender_id as "sender_id!", recipient_id as "recipient_id!", content as "content!", 
                           message_type::text as "message_type!", created_at as "created_at!", read_at,
                           latitude as "latitude!", longitude as "longitude!"
                    FROM direct_messages
                    WHERE id = $1
                    "#,
                    message_id
                )
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                Ok(row.map(|r| DirectMessage {
                    id: r.id,
                    message_id: r.message_id.to_string(),
                    conversation_id: r.conversation_id.to_string(),
                    sender_id: r.sender_id,
                    recipient_id: r.recipient_id,
                    content: r.content,
                    message_type: DirectMessageType::from_str(&r.message_type),
                    created_at: r.created_at,
                    read_at: r.read_at,
                    latitude: r.latitude,
                    longitude: r.longitude,
                }))
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
                let conversation_uuid = Uuid::parse_str(&message.conversation_id)
                    .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid conversation UUID: {}", e) })?;
                    
                let row = sqlx::query_as!(
            DirectMessageRow,
            r#"
            INSERT INTO direct_messages (message_id, conversation_id, sender_id, recipient_id, 
                                       content, message_type, created_at, latitude, longitude)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING id as "id!", message_id as "message_id!", conversation_id as "conversation_id!", 
                      sender_id as "sender_id!", recipient_id as "recipient_id!", content as "content!", 
                      message_type::text as "message_type!", created_at as "created_at!", read_at,
                      latitude as "latitude!", longitude as "longitude!"
            "#,
            message_uuid,
            conversation_uuid,
            message.sender_id,
            message.recipient_id,
            message.content,
            message.message_type.as_str() as &str,
            message.created_at,
            message.latitude,
            message.longitude
                )
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

                Ok(DirectMessage {
                    id: row.id,
                    message_id: row.message_id.to_string(),
                    conversation_id: row.conversation_id.to_string(),
                    sender_id: row.sender_id,
                    recipient_id: row.recipient_id,
                    content: row.content,
                    message_type: DirectMessageType::from_str(&row.message_type),
                    created_at: row.created_at,
                    read_at: row.read_at,
                    latitude: row.latitude,
                    longitude: row.longitude,
                })
            }
        ).await
    }

    /// 删除私聊消息
    pub async fn delete_direct_message(&self, message_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!("DELETE FROM direct_messages WHERE id = $1", message_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();
        
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message not found".to_string(),
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
        let conversation_uuid = Uuid::parse_str(conversation_id)
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid conversation UUID: {}", e) })?;
            
        let rows = sqlx::query!(
            r#"
            SELECT id as "id!", message_id as "message_id!", conversation_id as "conversation_id!", 
                   sender_id as "sender_id!", recipient_id as "recipient_id!", content as "content!", 
                   message_type as "message_type: DirectMessageType", created_at as "created_at!", read_at,
                   latitude as "latitude!", longitude as "longitude!"
            FROM direct_messages
            WHERE conversation_id = $1 AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            conversation_uuid,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(rows.into_iter().map(|r| DirectMessage {
            id: r.id,
            message_id: r.message_id.to_string(),
            conversation_id: r.conversation_id.to_string(),
            sender_id: r.sender_id,
            recipient_id: r.recipient_id,
            content: r.content,
            message_type: DirectMessageType::from_str("text"), // TODO: 修复枚举类型映射
            created_at: r.created_at,
            read_at: r.read_at,
            latitude: r.latitude,
            longitude: r.longitude,
        }).collect())
    }

    // ===== 群组消息操作 =====
    
    /// 根据ID查找群组消息
    pub async fn find_group_message_by_id(&self, message_id: i64) -> DomainResult<Option<GroupMessage>> {
        let row = sqlx::query!(
            r#"
            SELECT id as "id!", message_id as "message_id!", group_id as "group_id!", 
                   sender_id as "sender_id!", content as "content!", message_type::text as "message_type!", 
                   created_at as "created_at!", latitude as "latitude!", longitude as "longitude!"
            FROM group_messages
            WHERE id = $1
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(row.map(|r| GroupMessage {
            id: r.id,
            message_id: r.message_id.to_string(),
            group_id: r.group_id.to_string(),
            sender_id: r.sender_id,
            content: r.content,
            message_type: GroupMessageType::from_str("text"), // TODO: 修复枚举类型映射
            created_at: r.created_at,
            latitude: r.latitude,
            longitude: r.longitude,
        }))
    }

    /// 创建群组消息
    pub async fn create_group_message(&self, message: &GroupMessage) -> DomainResult<GroupMessage> {
        let message_uuid = Uuid::parse_str(&message.message_id)
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid message UUID: {}", e) })?;
        let group_uuid = Uuid::parse_str(&message.group_id)
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid group UUID: {}", e) })?;
            
        let row = sqlx::query!(
            r#"
            INSERT INTO group_messages (message_id, group_id, sender_id, content, message_type, 
                                      created_at, latitude, longitude)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING id as "id!", message_id as "message_id!", group_id as "group_id!", 
                      sender_id as "sender_id!", content as "content!", message_type::text as "message_type!", 
                      created_at as "created_at!", latitude as "latitude!", longitude as "longitude!"
            "#,
            message_uuid,
            group_uuid,
            message.sender_id,
            message.content,
            message.message_type.as_str() as &str,
            message.created_at,
            message.latitude,
            message.longitude
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(GroupMessage {
            id: row.id,
            message_id: row.message_id.to_string(),
            group_id: row.group_id.to_string(),
            sender_id: row.sender_id,
            content: row.content,
            message_type: GroupMessageType::from_str(&row.message_type),
            created_at: row.created_at,
            latitude: row.latitude,
            longitude: row.longitude,
        })
    }

    /// 删除群组消息
    pub async fn delete_group_message(&self, message_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!("DELETE FROM group_messages WHERE id = $1", message_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();
        
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message not found".to_string(),
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
        let group_uuid = Uuid::parse_str(group_id)
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid group UUID: {}", e) })?;
            
        let rows = sqlx::query!(
            r#"
            SELECT id as "id!", message_id as "message_id!", group_id as "group_id!", 
                   sender_id as "sender_id!", content as "content!", message_type::text as "message_type!", 
                   created_at as "created_at!", latitude as "latitude!", longitude as "longitude!"
            FROM group_messages
            WHERE group_id = $1 AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            group_uuid,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(rows.into_iter().map(|r| GroupMessage {
            id: r.id,
            message_id: r.message_id.to_string(),
            group_id: r.group_id.to_string(),
            sender_id: r.sender_id,
            content: r.content,
            message_type: GroupMessageType::from_str("text"), // TODO: 修复枚举类型映射
            created_at: r.created_at,
            latitude: r.latitude,
            longitude: r.longitude,
        }).collect())
    }

    // ===== 会话管理 =====
    
    /// 获取或创建会话
    pub async fn get_or_create_conversation(&self, user1_id: i64, user2_id: i64) -> DomainResult<Conversation> {
        // 确保user1_id < user2_id
        let (user1, user2) = if user1_id < user2_id {
            (user1_id, user2_id)
        } else {
            (user2_id, user1_id)
        };

        // 先尝试查找现有会话
        let existing = sqlx::query!(
            r#"
            SELECT id as "id!", conversation_id as "conversation_id!", user1_id as "user1_id!", 
                   user2_id as "user2_id!", created_at as "created_at!", updated_at as "updated_at!",
                   last_message_at as "last_message_at!", last_message_preview
            FROM conversations
            WHERE user1_id = $1 AND user2_id = $2
            "#,
            user1,
            user2
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        if let Some(row) = existing {
            return Ok(Conversation {
                id: row.id,
                conversation_id: row.conversation_id.to_string(),
                user1_id: row.user1_id,
                user2_id: row.user2_id,
                created_at: row.created_at,
                updated_at: row.updated_at,
                last_message_at: row.last_message_at,
                last_message_preview: row.last_message_preview,
            });
        }

        // 创建新会话
        let conversation_uuid = Uuid::new_v4();
        let row = sqlx::query!(
            r#"
            INSERT INTO conversations (conversation_id, user1_id, user2_id, created_at, updated_at, last_message_at)
            VALUES ($1, $2, $3, NOW(), NOW(), NOW())
            RETURNING id as "id!", conversation_id as "conversation_id!", user1_id as "user1_id!", 
                      user2_id as "user2_id!", created_at as "created_at!", updated_at as "updated_at!",
                      last_message_at as "last_message_at!", last_message_preview
            "#,
            conversation_uuid,
            user1,
            user2
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(Conversation {
            id: row.id,
            conversation_id: row.conversation_id.to_string(),
            user1_id: row.user1_id,
            user2_id: row.user2_id,
            created_at: row.created_at,
            updated_at: row.updated_at,
            last_message_at: row.last_message_at,
            last_message_preview: row.last_message_preview,
        })
    }

    /// 获取用户的会话列表
    pub async fn list_conversations(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<(Vec<ConversationSummary>, bool, Option<i64>)> {
        let rows = sqlx::query!(
            r#"
            SELECT c.id as "id!", c.conversation_id as "conversation_id!", 
                   c.user1_id as "user1_id!", c.user2_id as "user2_id!",
                   c.created_at as "created_at!", c.last_message_at as "last_message_at!", 
                   c.last_message_preview,
                   CASE 
                       WHEN c.user1_id = $1 THEN c.user2_id 
                       ELSE c.user1_id 
                   END as "peer_id!"
            FROM conversations c
            WHERE (c.user1_id = $1 OR c.user2_id = $1) AND ($2 = 0 OR c.id < $2)
            ORDER BY c.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            (limit + 1) as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more {
            items.pop();
        }

        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id)
        } else {
            None
        };

        let summaries = items.into_iter().map(|row| ConversationSummary {
            conversation_id: row.conversation_id.to_string(),
            peer_id: row.peer_id,
            last_message_preview: row.last_message_preview,
            last_message_at: row.last_message_at,
        }).collect();

        Ok((summaries, has_more, next_cursor))
    }

    // ===== 群组事件操作 =====
    
    /// 创建群组事件
    /// TODO: 修复 sqlx 枚举映射问题
    #[allow(dead_code)]
    pub async fn create_group_event(&self, _event: &GroupEventRecord) -> DomainResult<GroupEventRecord> {
        // 暂时返回一个占位实现，避免编译错误
        // 这个方法需要在运行时修复 sqlx 枚举映射问题
        Err(DomainError::BusinessRuleViolation {
            rule: "create_group_event 方法暂未实现，需要修复 sqlx 枚举映射".to_string()
        })
    }

    // ===== 通用删除方法（兼容旧控制器） =====
    
    /// 删除消息（尝试私聊和群组消息）
    pub async fn delete_message(&self, message_id: i64) -> DomainResult<()> {
        // 先尝试从私聊消息表删除
        let direct_rows = sqlx::query!("DELETE FROM direct_messages WHERE id = $1", message_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();

        if direct_rows > 0 {
            return Ok(());
        }

        // 再尝试从群组消息表删除
        let group_rows = sqlx::query!("DELETE FROM group_messages WHERE id = $1", message_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();

        if group_rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message not found".to_string(),
            });
        }
        Ok(())
    }
    
    // ===== 消息状态管理方法 =====
    
    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        &self,
        group_id: &str,
        user_id: i64,
        last_read_message_id: Option<&str>,
    ) -> DomainResult<()> {
        let group_uuid = uuid::Uuid::parse_str(group_id)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Invalid group_id format: {}", e),
            })?;

        if let Some(message_id) = last_read_message_id {
            // 获取消息的自增ID
            let message_uuid = uuid::Uuid::parse_str(message_id)
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Invalid message_id format: {}", e),
                })?;
            
            let message_row = sqlx::query!(
                "SELECT id FROM group_messages WHERE message_id = $1",
                message_uuid
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
            
            if let Some(row) = message_row {
                // 更新用户已读状态表
                sqlx::query!(
                    "INSERT INTO user_message_read_status (user_id, group_id, last_read_message_id)
                     VALUES ($1, $2, $3)
                     ON CONFLICT (user_id, group_id)
                     DO UPDATE SET last_read_message_id = $3, updated_at = NOW()",
                    user_id,
                    group_uuid,
                    row.id
                )
                .execute(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Database error: {}", e),
                })?;
            }
        } else {
            // 获取该群组最新消息的ID
            let latest_message = sqlx::query!(
                "SELECT id FROM group_messages WHERE group_id = $1 ORDER BY id DESC LIMIT 1",
                group_uuid
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
            
            if let Some(row) = latest_message {
                // 更新用户已读状态表
                sqlx::query!(
                    "INSERT INTO user_message_read_status (user_id, group_id, last_read_message_id)
                     VALUES ($1, $2, $3)
                     ON CONFLICT (user_id, group_id)
                     DO UPDATE SET last_read_message_id = $3, updated_at = NOW()",
                    user_id,
                    group_uuid,
                    row.id
                )
                .execute(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Database error: {}", e),
                })?;
            }
        }
        
        Ok(())
    }
    
    /// 标记私聊消息已读
    pub async fn mark_direct_messages_read(
        &self,
        conversation_id: &str,
        user_id: i64,
        last_read_message_id: Option<&str>,
    ) -> DomainResult<()> {
        let conversation_uuid = uuid::Uuid::parse_str(conversation_id)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Invalid conversation_id format: {}", e),
            })?;

        if let Some(message_id) = last_read_message_id {
            // 获取消息的自增ID
            let message_uuid = uuid::Uuid::parse_str(message_id)
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Invalid message_id format: {}", e),
                })?;
            
            let message_row = sqlx::query!(
                "SELECT id FROM direct_messages WHERE message_id = $1",
                message_uuid
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
            
            if let Some(row) = message_row {
                // 更新该消息及之前的消息为已读
                sqlx::query!(
                    "UPDATE direct_messages SET read_at = NOW() 
                     WHERE conversation_id = $1 AND recipient_id = $2 AND id <= $3 AND read_at IS NULL",
                    conversation_uuid,
                    user_id,
                    row.id
                )
                .execute(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Database error: {}", e),
                })?;
            }
        } else {
            // 标记该会话中用户接收的所有未读消息为已读
            sqlx::query!(
                "UPDATE direct_messages SET read_at = NOW() 
                 WHERE conversation_id = $1 AND recipient_id = $2 AND read_at IS NULL",
                conversation_uuid,
                user_id
            )
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
        group_id: &str,
        user_id: i64,
    ) -> DomainResult<i64> {
        let group_uuid = uuid::Uuid::parse_str(group_id)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Invalid group_id format: {}", e),
            })?;

        // 获取用户在该群组的已读状态
        let last_read_id = sqlx::query!(
            "SELECT last_read_message_id FROM user_message_read_status 
             WHERE user_id = $1 AND group_id = $2",
            user_id,
            group_uuid
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let unread_count = if let Some(read_status) = last_read_id {
            // 计算比最后已读消息ID更大的消息数量
            sqlx::query!(
                "SELECT COUNT(*) as \"count!\"
                 FROM group_messages gm
                 WHERE gm.group_id = $1 
                 AND gm.sender_id != $2 
                 AND gm.id > $3",
                group_uuid,
                user_id,
                read_status.last_read_message_id
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .count
        } else {
            // 用户从未读过该群组消息，计算所有非自己发送的消息
            sqlx::query!(
                "SELECT COUNT(*) as \"count!\"
                 FROM group_messages gm
                 WHERE gm.group_id = $1 
                 AND gm.sender_id != $2",
                group_uuid,
                user_id
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .count
        };
        
        Ok(unread_count)
    }
    
    /// 获取会话未读消息数
    pub async fn get_conversation_unread_count(
        &self,
        conversation_id: &str,
        user_id: i64,
    ) -> DomainResult<i64> {
        let conversation_uuid = uuid::Uuid::parse_str(conversation_id)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Invalid conversation_id format: {}", e),
            })?;

        let row = sqlx::query!(
            "SELECT COUNT(*) as \"count!\"
             FROM direct_messages dm
             WHERE dm.conversation_id = $1 
             AND dm.recipient_id = $2 
             AND dm.read_at IS NULL",
            conversation_uuid,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        
        Ok(row.count)
    }
    
    /// 获取用户所有会话的总未读消息数
    pub async fn get_user_total_unread_count(
        &self,
        user_id: i64,
    ) -> DomainResult<i64> {
        let row = sqlx::query!(
            "SELECT COUNT(*) as \"count!\"
             FROM direct_messages dm
             WHERE dm.recipient_id = $1 
             AND dm.read_at IS NULL",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        
        Ok(row.count)
    }
}