/// 消息应用服务
/// 
/// 处理消息相关的业务逻辑，使用新的实体和仓库架构

use crate::application::queries::{PaginatedResult};
use crate::domain::{
    DirectMessage, DirectMessageType, DomainResult, GroupMessage, 
    GroupMessageType
};
use crate::adapters::adapters_pg::message_repository::PgMessageRepository;
use std::sync::Arc;
use uuid::Uuid;
use chrono::Utc;

#[derive(Clone)]
pub struct MessageApplicationService {
    pg_message_repository: Arc<PgMessageRepository>,
}

impl MessageApplicationService {
    pub fn new(
        pg_message_repository: Arc<PgMessageRepository>,
    ) -> Self {
        Self { 
            pg_message_repository,
        }
    }
    
    /// 获取PG消息仓库的引用（用于控制器直接访问）
    pub fn get_pg_repository(&self) -> &Arc<PgMessageRepository> {
        &self.pg_message_repository
    }

    // ===== 私聊消息业务逻辑 =====
    
    /// 创建私聊消息
    pub async fn create_direct_message(
        &self,
        sender_id: i64,
        recipient_id: i64,
        content: String,
        _message_type: DirectMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<DirectMessage> {
        // 获取或创建会话
        let conversation = self.pg_message_repository
            .get_or_create_conversation(sender_id, recipient_id)
            .await?;
        
        // 创建消息
        let message = DirectMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            conversation_id: conversation.conversation_id,
            sender_id,
            recipient_id,
            content,
            message_type: DirectMessageType::Text, // 暂时固定为文本
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        };
        
        self.pg_message_repository.create_direct_message(&message).await
    }
    
    /// 获取私聊消息历史
    pub async fn get_direct_messages_by_conversation(
        &self,
        conversation_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<DirectMessage>> {
        let limit = limit.clamp(1, 100);
        let messages = self.pg_message_repository
            .find_direct_messages_by_conversation(conversation_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let mut items = messages;
        if has_more {
            items.pop();
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }

    // ===== 群组消息业务逻辑 =====
    
    /// 创建群组消息
    pub async fn create_group_message(
        &self,
        group_id: String,
        sender_id: i64,
        content: String,
        _message_type: GroupMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<GroupMessage> {
        let message = GroupMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            group_id,
            sender_id,
            content,
            message_type: GroupMessageType::Text, // 暂时固定为文本
            created_at: Utc::now(),
            latitude,
            longitude,
        };
        
        self.pg_message_repository.create_group_message(&message).await
    }
    
    /// 获取群组消息历史
    pub async fn get_group_messages_by_group(
        &self,
        group_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<GroupMessage>> {
        let limit = limit.clamp(1, 100);
        let messages = self.pg_message_repository
            .find_group_messages(group_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let mut items = messages;
        if has_more {
            items.pop();
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}