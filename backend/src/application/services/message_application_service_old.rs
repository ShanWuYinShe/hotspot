/// 消息应用服务
/// 
/// 功能职责：
/// - 实现消息通讯相关的业务用例和应用逻辑
/// - 协调消息领域实体和仓储层的交互
/// - 管理私聊和群聊消息的发送、接收、存储
/// - 处理消息历史查询和分页显示
/// - 管理消息状态和已读回执机制
/// 
/// 主要用例：
/// - 私聊消息的发送和接收
/// - 群组消息的广播和分发
/// - 消息历史的查询和分页加载
/// - 会话列表的管理和维护
/// - 消息已读状态的更新和同步
/// - 未读消息数量的统计和查询
/// 
/// 实时通信：与WebSocket系统集成实现实时消息推送

use crate::application::handlers::PaginatedResult;
use crate::domain::{
    DirectMessage, DomainResult, GroupMessage
};
use crate::domain::entities::{DirectMessageType, GroupMessageType};

/// 统一消息枚举，支持群聊和私聊消息的统一处理
#[derive(Debug, Clone)]
pub enum UnifiedMessage {
    Group(GroupMessage),
    Direct(DirectMessage),
}
use crate::adapters::adapters_pg::message_repository::PgMessageRepository;
use crate::ports::NotificationTransport;
use crate::ports::services::{Notification, NotificationType};
use std::sync::Arc;
use uuid::Uuid;
use chrono::Utc;

#[derive(Clone)]
pub struct MessageApplicationService {
    base: super::BaseApplicationService,
    pg_message_repository: Arc<PgMessageRepository>,
    notification_transport: Option<Arc<dyn NotificationTransport>>,
}

impl MessageApplicationService {
    pub fn new(
        pg_message_repository: Arc<PgMessageRepository>,
        user_repository: Arc<dyn crate::ports::UserRepository>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        Self { 
            base: super::BaseApplicationService::new("MessageApplicationService")
                .with_user_repository(user_repository)
                .with_event_publisher(event_publisher),
            pg_message_repository,
            notification_transport: None,
        }
    }
    
    /// 添加通知传输器
    pub fn with_notification_transport(mut self, transport: Arc<dyn NotificationTransport>) -> Self {
        self.notification_transport = Some(transport);
        self
    }
    
    /// 获取PG消息仓库的引用（用于控制器直接访问）
    pub fn get_pg_repository(&self) -> &Arc<PgMessageRepository> {
        &self.pg_message_repository
    }

    // ===== 统一消息业务逻辑 =====
    
    /// 发送消息到会话（统一接口）
    /// 支持群聊和私聊，通过conversation_id区分
    pub async fn send_message_to_conversation(
        &self,
        conversation_id: String,
        sender_id: i64,
        content: String,
        message_type: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
    ) -> DomainResult<UnifiedMessage> {
        // 验证用户存在
        self.base.validate_users_exist(&[sender_id]).await?;
        
        // 验证消息内容
        super::ValidationRules::validate_text_content(&content, 1000, "消息内容")?;
        
        // 记录操作日志
        self.base.log_operation("send_message_to_conversation", Some(sender_id), Some(&conversation_id));
        
        // 通过conversation_id获取会话信息，确定是群聊还是私聊
        let conversation = self.pg_message_repository
            .get_conversation_by_id(&conversation_id)
            .await?;
        
        // 根据会话类型发送消息
        match conversation.conversation_type.as_str() {
            "group" => {
                // 发送群组消息
                let group_message = self.create_group_message(
                    conversation.group_id.unwrap(),
                    sender_id,
                    content,
                    GroupMessageType::from_str(&message_type),
                    latitude.unwrap_or(0.0),
                    longitude.unwrap_or(0.0),
                ).await?;
                
                Ok(UnifiedMessage::Group(group_message))
            }
            "private" => {
                // 发送私聊消息
                // 需要从会话参与者中获取接收者ID
                let participants = self.pg_message_repository
                    .get_conversation_participants(&conversation_id)
                    .await?;
                
                let recipient_id = participants.iter()
                    .find(|p| p.user_id != sender_id)
                    .map(|p| p.user_id)
                    .ok_or_else(|| crate::domain::DomainError::BusinessRuleViolation {
                        rule: "无法找到私聊接收者".to_string(),
                    })?;
                
                let direct_message = self.create_direct_message(
                    sender_id,
                    recipient_id,
                    content,
                    DirectMessageType::from_str(&message_type),
                    latitude.unwrap_or(0.0),
                    longitude.unwrap_or(0.0),
                ).await?;
                
                Ok(UnifiedMessage::Direct(direct_message))
            }
            _ => Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "不支持的会话类型".to_string(),
            })
        }
    }
    
    /// 获取会话消息历史（统一接口）
    pub async fn get_conversation_messages(
        &self,
        conversation_id: String,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<UnifiedMessage>> {
        // 验证用户存在
        self.base.validate_users_exist(&[user_id]).await?;
        
        // 验证用户是否为会话参与者
        let is_participant = self.pg_message_repository
            .is_conversation_participant(&conversation_id, user_id)
            .await?;
        
        if !is_participant {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "用户不是该会话的参与者".to_string(),
            });
        }
        
        // 获取会话信息
        let conversation = self.pg_message_repository
            .get_conversation_by_id(&conversation_id)
            .await?;
        
        // 根据会话类型获取消息
        match conversation.conversation_type.as_str() {
            "group" => {
                let group_messages = self.get_group_messages(
                    conversation.group_id.unwrap(),
                    cursor,
                    limit,
                ).await?;
                
                let unified_messages = group_messages.items
                    .into_iter()
                    .map(UnifiedMessage::Group)
                    .collect();
                
                Ok(PaginatedResult {
                    items: unified_messages,
                    has_more: group_messages.has_more,
                    next_cursor: group_messages.next_cursor,
                })
            }
            "private" => {
                let direct_messages = self.get_direct_messages(
                    &conversation_id,
                    cursor,
                    limit,
                ).await?;
                
                let unified_messages = direct_messages.items
                    .into_iter()
                    .map(UnifiedMessage::Direct)
                    .collect();
                
                Ok(PaginatedResult {
                    items: unified_messages,
                    has_more: direct_messages.has_more,
                    next_cursor: direct_messages.next_cursor,
                })
            }
            _ => Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "不支持的会话类型".to_string(),
            })
        }
    }
    
    /// 标记会话消息已读（统一接口）
    pub async fn mark_conversation_messages_read(
        &self,
        conversation_id: String,
        user_id: i64,
        message_id: Option<String>,
    ) -> DomainResult<()> {
        // 验证用户存在
        self.base.validate_users_exist(&[user_id]).await?;
        
        // 获取会话信息
        let conversation = self.pg_message_repository
            .get_conversation_by_id(&conversation_id)
            .await?;
        
        // 根据会话类型标记已读
        match conversation.conversation_type.as_str() {
            "group" => {
                self.pg_message_repository
                    .mark_group_messages_read(user_id, &conversation_id, message_id.as_deref())
                    .await
            }
            "private" => {
                self.pg_message_repository
                    .mark_direct_messages_read(user_id, &conversation_id, message_id.as_deref())
                    .await
            }
            _ => Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "不支持的会话类型".to_string(),
            })
        }
    }
    
    /// 获取会话未读消息数（统一接口）
    pub async fn get_conversation_unread_count(
        &self,
        conversation_id: String,
        user_id: i64,
    ) -> DomainResult<i32> {
        // 验证用户存在
        self.base.validate_users_exist(&[user_id]).await?;
        
        // 获取会话信息
        let conversation = self.pg_message_repository
            .get_conversation_by_id(&conversation_id)
            .await?;
        
        // 根据会话类型获取未读数
        match conversation.conversation_type.as_str() {
            "group" => {
                self.pg_message_repository
                    .get_group_unread_count(user_id, &conversation_id)
                    .await
            }
            "private" => {
                self.pg_message_repository
                    .get_direct_unread_count(user_id, &conversation_id)
                    .await
            }
            _ => Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "不支持的会话类型".to_string(),
            })
        }
    }
    
    /// 删除消息（统一接口）
    pub async fn delete_message(
        &self,
        message_id: String,
        user_id: i64,
    ) -> DomainResult<()> {
        // 验证用户存在
        self.base.validate_users_exist(&[user_id]).await?;
        
        // 记录操作日志
        self.base.log_operation("delete_message", Some(user_id), Some(&message_id));
        
        // 通过消息ID获取消息信息，确定是群聊还是私聊消息
        // 这里需要在仓库层实现统一的消息删除方法
        self.pg_message_repository
            .delete_message_by_id(&message_id, user_id)
            .await
    }

    // ===== 私聊消息业务逻辑 =====
    
    /// 创建私聊消息
    pub async fn create_direct_message(
        &self,
        sender_id: i64,
        recipient_id: i64,
        content: String,
        message_type: DirectMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<DirectMessage> {
        // 使用基类验证用户存在
        self.base.validate_users_exist(&[sender_id, recipient_id]).await?;
        
        // 验证消息内容
        super::ValidationRules::validate_text_content(&content, 1000, "消息内容")?;
        
        // 记录操作日志
        self.base.log_operation("create_direct_message", Some(sender_id), Some(&format!("to: {}", recipient_id)));
        
        // 创建消息
        let message = DirectMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            conversation_id: String::new(), // 将由仓库层处理
            sender_id,
            recipient_id,
            content,
            message_type,
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        };
        
        let created_message = self.pg_message_repository.create_direct_message(&message).await?;
        
        // 发送WebSocket通知给接收者
        if let Some(transport) = &self.notification_transport {
            let notification = Notification {
                notification_id: Uuid::new_v4().to_string(),
                user_id: Some(crate::domain::UserId::from_trusted(recipient_id)),
                notification_type: NotificationType::Message,
                title: "新消息".to_string(),
                content: format!("您收到一条新消息: {}", created_message.content),
                data: Some(serde_json::json!({
                    "message_id": created_message.message_id,
                    "sender_id": sender_id,
                    "message_type": "direct_message"
                })),
                priority: crate::ports::services::NotificationPriority::Normal,
            };
            
            if let Err(e) = transport.send_notification(recipient_id, &notification).await {
                tracing::warn!("发送WebSocket通知失败: {:?}", e);
                // 不影响消息创建的成功
            }
        }
        
        Ok(created_message)
    }
    
    /// 获取私聊消息历史
    pub async fn get_direct_messages_by_conversation(
        &self,
        conversation_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<DirectMessage>> {
        // 记录操作日志
        self.base.log_operation("get_direct_messages_by_conversation", None, Some(&format!("conversation: {}", conversation_id)));
        
        let messages = self.pg_message_repository
            .find_direct_messages_by_conversation(conversation_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let items = if has_more {
            messages.into_iter().take(limit as usize).collect()
        } else {
            messages
        };
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id.to_string())
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 标记私聊消息已读
    pub async fn mark_direct_messages_read(
        &self,
        user_id: i64,
        conversation_id: &str,
        message_id: Option<&str>,
    ) -> DomainResult<()> {
        self.base.validate_users_exist(&[user_id]).await?;
        
        self.base.log_operation("mark_direct_messages_read", Some(user_id), Some(&format!("conversation: {}", conversation_id)));
        
        self.pg_message_repository
            .mark_direct_messages_read(user_id, conversation_id, message_id)
            .await
    }
    
    /// 获取私聊未读消息数
    pub async fn get_direct_unread_count(
        &self,
        user_id: i64,
        conversation_id: Option<&str>,
    ) -> DomainResult<i32> {
        self.base.validate_users_exist(&[user_id]).await?;
        
        self.base.log_operation("get_direct_unread_count", Some(user_id), conversation_id);
        
        self.pg_message_repository
            .get_direct_unread_count(user_id, conversation_id)
            .await
    }
    
    /// 获取用户会话列表
    pub async fn get_conversations(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<crate::domain::Conversation>> {
        self.base.validate_users_exist(&[user_id]).await?;
        
        self.base.log_operation("get_conversations", Some(user_id), Some(&format!("cursor: {}, limit: {}", cursor, limit)));
        
        let conversations = self.pg_message_repository
            .find_conversations_by_user(user_id, cursor, limit + 1)
            .await?;
        
        let has_more = conversations.len() > limit as usize;
        let items = if has_more {
            conversations.into_iter().take(limit as usize).collect()
        } else {
            conversations
        };
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id.to_string())
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items,
            has_more,
            next_cursor,
        })
    }

    // ===== 群组消息业务逻辑 =====
    
    /// 创建群组消息
    pub async fn create_group_message(
        &self,
        sender_id: i64,
        group_id: String,
        content: String,
        message_type: GroupMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<GroupMessage> {
        // 验证用户存在
        self.base.validate_users_exist(&[sender_id]).await?;
        
        // 验证消息内容
        super::ValidationRules::validate_text_content(&content, 1000, "消息内容")?;
        
        // 记录操作日志
        self.base.log_operation("create_group_message", Some(sender_id), Some(&format!("group: {}", group_id)));
        
        // 创建消息
        let message = GroupMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            group_id,
            sender_id,
            content,
            message_type,
            created_at: Utc::now(),
            latitude,
            longitude,
        };
        
        let created_message = self.pg_message_repository.create_group_message(&message).await?;
        
        // 发送WebSocket通知给群组成员
        if let Some(_transport) = &self.notification_transport {
            let _notification = Notification {
                notification_id: Uuid::new_v4().to_string(),
                user_id: None, // 群组消息不指定特定用户
                notification_type: NotificationType::Message,
                title: "群组新消息".to_string(),
                content: format!("群组收到新消息: {}", created_message.content),
                data: Some(serde_json::json!({
                    "message_id": created_message.message_id,
                    "sender_id": sender_id,
                    "group_id": created_message.group_id,
                    "message_type": "group_message"
                })),
                priority: crate::ports::services::NotificationPriority::Normal,
            };
            
            // 这里应该通过群组服务获取群组成员列表，然后逐个发送通知
            // 暂时记录日志，实际实现需要群组成员查询
            tracing::info!("群组消息通知: 群组 {} 收到新消息", created_message.group_id);
            
            // TODO: 实现群组成员通知
            // let group_members = self.group_service.get_group_members(&created_message.group_id).await?;
            // for member in group_members {
            //     if member.user_id != sender_id { // 不给发送者发送通知
            //         transport.send_notification(member.user_id, &notification).await?;
            //     }
            // }
        }
        
        Ok(created_message)
    }
    
    /// 获取群组消息历史
    pub async fn get_group_messages(
        &self,
        group_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<GroupMessage>> {
        // 记录操作日志
        self.base.log_operation("get_group_messages", None, Some(&format!("group: {}", group_id)));
        
        let messages = self.pg_message_repository
            .find_group_messages(group_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let items = if has_more {
            messages.into_iter().take(limit as usize).collect()
        } else {
            messages
        };
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id.to_string())
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items,
            has_more,
            next_cursor,
        })
    }

    
    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        &self,
        user_id: i64,
        group_id: &str,
        message_id: Option<&str>,
    ) -> DomainResult<()> {
        self.base.validate_users_exist(&[user_id]).await?;
        
        self.base.log_operation("mark_group_messages_read", Some(user_id), Some(&format!("group: {}", group_id)));
        
        self.pg_message_repository
            .mark_group_messages_read(user_id, group_id, message_id)
            .await
    }
    
    /// 获取群组未读消息数
    pub async fn get_group_unread_count(
        &self,
        user_id: i64,
        group_id: Option<&str>,
    ) -> DomainResult<i32> {
        self.base.validate_users_exist(&[user_id]).await?;
        
        self.base.log_operation("get_group_unread_count", Some(user_id), group_id);
        
        self.pg_message_repository
            .get_group_unread_count(user_id, group_id)
            .await
    }
}