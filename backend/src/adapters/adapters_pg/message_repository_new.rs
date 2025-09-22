/// PostgreSQL统一消息仓储实现 - 符合需求文档的统一消息架构
/// 
/// 实现统一消息存储，支持私聊和群聊的统一管理
/// 基于统一的数据库表结构：messages表统一存储所有消息

use async_trait::async_trait;
use sqlx::{PgPool, Row};
use uuid::Uuid;
use chrono::{DateTime, Utc};
use std::sync::Arc;

use crate::domain::{DomainError, DomainResult};
use crate::domain::entities::message::{Message, MessageType};
use crate::domain::values::{MessageId, GroupId, UserId};
use crate::ports::MessageRepository;
use crate::application::handlers::PaginatedResult;
use crate::application::services::message_application_service::GroupWithLastMessage;

/// PostgreSQL消息仓储实现
#[derive(Clone)]
pub struct PgMessageRepository {
    pool: Arc<PgPool>,
}

impl PgMessageRepository {
    /// 创建新的消息仓储实例
    pub fn new(pool: Arc<PgPool>) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl MessageRepository for PgMessageRepository {
    /// 创建消息（统一接口）
    async fn create(&self, mut message: Message) -> DomainResult<Message> {
        let query = r#"
            INSERT INTO messages (
                message_id, group_id, sender_id, content, message_type,
                media_type, media_url, media_size, reply_to_message_id,
                created_at, updated_at, is_deleted
            ) VALUES (
                $1, $2, $3, $4, $5::message_type,
                $6::media_type, $7, $8, $9,
                $10, $11, $12
            )
            RETURNING message_id, created_at, updated_at
        "#;
        
        let message_id = Uuid::new_v4();
        let now = Utc::now();
        
        let row = sqlx::query(query)
            .bind(&message_id)
            .bind(&message.group_id.to_string())
            .bind(message.sender_id.value())
            .bind(&message.content)
            .bind(message.message_type.to_string().to_lowercase())
            .bind(message.media_type.map(|mt| mt.to_string().to_lowercase()))
            .bind(&message.media_url)
            .bind(message.media_size)
            .bind(message.reply_to_message_id.as_ref().map(|id| id.to_string()))
            .bind(&now)
            .bind(&now)
            .bind(false)
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "create_message".to_string(),
                details: e.to_string() 
            })?;
        
        // 更新消息ID和时间戳
        message.id = MessageId::from(row.get::<Uuid, _>("message_id").to_string());
        message.created_at = row.get::<DateTime<Utc>, _>("created_at");
        message.updated_at = row.get::<DateTime<Utc>, _>("updated_at");
        
        Ok(message)
    }
    
    /// 更新消息
    async fn update(&self, message: Message) -> DomainResult<Message> {
        let query = r#"
            UPDATE messages 
            SET content = $2, updated_at = $3, is_deleted = $4, deleted_at = $5
            WHERE message_id = $1
            RETURNING updated_at
        "#;
        
        let row = sqlx::query(query)
            .bind(&message.id.to_string())
            .bind(&message.content)
            .bind(&message.updated_at)
            .bind(message.is_deleted)
            .bind(message.deleted_at)
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "update_message".to_string(),
                details: e.to_string() 
            })?;
        
        let mut updated_message = message;
        updated_message.updated_at = row.get::<DateTime<Utc>, _>("updated_at");
        
        Ok(updated_message)
    }
    
    /// 删除消息
    async fn delete_message(&self, message_id: MessageId) -> DomainResult<bool> {
        let query = r#"
            UPDATE messages 
            SET is_deleted = true, deleted_at = $2, updated_at = $2
            WHERE message_id = $1 AND is_deleted = false
        "#;
        
        let result = sqlx::query(query)
            .bind(&message_id.to_string())
            .bind(&Utc::now())
            .execute(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "delete_message".to_string(),
                details: e.to_string() 
            })?;
        
        Ok(result.rows_affected() > 0)
    }
    
    /// 根据ID查找消息
    async fn find_by_id(&self, message_id: &MessageId) -> DomainResult<Option<Message>> {
        let query = r#"
            SELECT 
                message_id, group_id, sender_id, content, message_type,
                media_type, media_url, media_size, reply_to_message_id,
                created_at, updated_at, is_deleted, deleted_at
            FROM messages 
            WHERE message_id = $1
        "#;
        
        let row = sqlx::query(query)
            .bind(&message_id.to_string())
            .fetch_optional(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "find_message_by_id".to_string(),
                details: e.to_string() 
            })?;
        
        if let Some(row) = row {
            Ok(Some(self.row_to_message(row)?))
        } else {
            Ok(None)
        }
    }
    
    /// 根据群组ID分页查找消息
    async fn find_by_group_id_paginated(
        &self,
        group_id: &GroupId,
        cursor: Option<&str>,
        limit: i32,
        include_system_messages: bool,
    ) -> DomainResult<PaginatedResult<Message>> {
        let mut query = r#"
            SELECT 
                message_id, group_id, sender_id, content, message_type,
                media_type, media_url, media_size, reply_to_message_id,
                created_at, updated_at, is_deleted, deleted_at
            FROM messages 
            WHERE group_id = $1 AND is_deleted = false
        "#.to_string();
        
        if !include_system_messages {
            query.push_str(" AND message_type != 'system'");
        }
        
        if let Some(cursor) = cursor {
            query.push_str(" AND created_at < (SELECT created_at FROM messages WHERE message_id = $3)");
        }
        
        query.push_str(" ORDER BY created_at DESC LIMIT $2");
        
        let mut sql_query = sqlx::query(&query)
            .bind(&group_id.to_string())
            .bind(limit + 1); // 多查询一条用于判断是否有更多数据
        
        if let Some(cursor) = cursor {
            sql_query = sql_query.bind(cursor);
        }
        
        let rows = sql_query
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "find_messages_by_group_id".to_string(),
                details: e.to_string() 
            })?;
        
        let has_more = rows.len() > limit as usize;
        let mut messages: Vec<Message> = rows
            .into_iter()
            .take(limit as usize)
            .map(|row| self.row_to_message(row))
            .collect::<Result<Vec<_>, _>>()?;
        
        // 按时间正序排列（最新的在后面）
        messages.reverse();
        
        let next_cursor = if has_more && !messages.is_empty() {
            Some(messages.last().unwrap().id.to_string())
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: messages,
            has_more,
            next_cursor,
        })
    }
    
    /// 标记消息已读
    async fn mark_message_read(&self, message_id: MessageId, user_id: UserId) -> DomainResult<()> {
        let query = r#"
            INSERT INTO message_read_status (message_id, user_id, read_at)
            VALUES ($1, $2, $3)
            ON CONFLICT (message_id, user_id) 
            DO UPDATE SET read_at = $3
        "#;
        
        sqlx::query(query)
            .bind(&message_id.to_string())
            .bind(user_id.value())
            .bind(&Utc::now())
            .execute(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "mark_message_read".to_string(),
                details: e.to_string() 
            })?;
        
        Ok(())
    }
    
    /// 标记群组消息已读到指定消息
    async fn mark_read_until_message(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
        message_id: &MessageId,
    ) -> DomainResult<()> {
        let query = r#"
            INSERT INTO message_read_status (message_id, user_id, read_at)
            SELECT m.message_id, $2, $4
            FROM messages m
            WHERE m.group_id = $1 
                AND m.created_at <= (
                    SELECT created_at FROM messages WHERE message_id = $3
                )
                AND NOT EXISTS (
                    SELECT 1 FROM message_read_status mrs 
                    WHERE mrs.message_id = m.message_id AND mrs.user_id = $2
                )
        "#;
        
        sqlx::query(query)
            .bind(&group_id.to_string())
            .bind(user_id.value())
            .bind(&message_id.to_string())
            .bind(&Utc::now())
            .execute(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "mark_read_until_message".to_string(),
                details: e.to_string() 
            })?;
        
        Ok(())
    }
    
    /// 标记群组所有消息已读
    async fn mark_all_read(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
    ) -> DomainResult<()> {
        let query = r#"
            INSERT INTO message_read_status (message_id, user_id, read_at)
            SELECT m.message_id, $2, $3
            FROM messages m
            WHERE m.group_id = $1 
                AND m.is_deleted = false
                AND NOT EXISTS (
                    SELECT 1 FROM message_read_status mrs 
                    WHERE mrs.message_id = m.message_id AND mrs.user_id = $2
                )
        "#;
        
        sqlx::query(query)
            .bind(&group_id.to_string())
            .bind(user_id.value())
            .bind(&Utc::now())
            .execute(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "mark_all_read".to_string(),
                details: e.to_string() 
            })?;
        
        Ok(())
    }
    
    /// 获取群组未读消息数
    async fn get_unread_count(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
    ) -> DomainResult<i32> {
        let query = r#"
            SELECT COUNT(*)::int as unread_count
            FROM messages m
            WHERE m.group_id = $1 
                AND m.is_deleted = false
                AND m.sender_id != $2  -- 排除自己发送的消息
                AND NOT EXISTS (
                    SELECT 1 FROM message_read_status mrs 
                    WHERE mrs.message_id = m.message_id AND mrs.user_id = $2
                )
        "#;
        
        let row = sqlx::query(query)
            .bind(&group_id.to_string())
            .bind(user_id.value())
            .fetch_one(&*self.pool)
            .await
            .map_err(|e| DomainError::DatabaseError { 
                operation: "get_unread_count".to_string(),
                details: e.to_string() 
            })?;
        
        Ok(row.get::<i32, _>("unread_count"))
    }
}

impl PgMessageRepository {
    /// 将数据库行转换为Message实体
    fn row_to_message(&self, row: sqlx::postgres::PgRow) -> DomainResult<Message> {
        let message_type_str: String = row.get("message_type");
        let message_type = message_type_str.parse::<MessageType>()
            .map_err(|e| DomainError::ValidationError { 
                field: "message_type".to_string(),
                reason: e 
            })?;
        
        let media_type = if let Some(media_type_str) = row.get::<Option<String>, _>("media_type") {
            Some(media_type_str.parse().map_err(|e| DomainError::ValidationError { 
                field: "media_type".to_string(),
                reason: format!("无效的媒体类型: {}", e)
            })?)
        } else {
            None
        };
        
        Ok(Message {
            id: MessageId::from(row.get::<Uuid, _>("message_id").to_string()),
            group_id: GroupId::from(row.get::<String, _>("group_id")),
            sender_id: UserId::from(row.get::<i64, _>("sender_id")),
            content: row.get("content"),
            message_type,
            media_type,
            media_url: row.get("media_url"),
            media_size: row.get("media_size"),
            reply_to_message_id: row.get::<Option<String>, _>("reply_to_message_id")
                .map(MessageId::from),
            created_at: row.get("created_at"),
            updated_at: row.get("updated_at"),
            is_deleted: row.get("is_deleted"),
            deleted_at: row.get("deleted_at"),
            events: vec![], // 领域事件在仓储层不需要处理
        })
    }
}

