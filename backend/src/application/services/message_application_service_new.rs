/// 统一消息应用服务 - 符合需求文档的统一消息架构
/// 
/// 核心设计理念：
/// - 私聊和群聊都通过group_id统一管理
/// - 私聊也是群组，group_type='private_chat'
/// - 统一的Message实体，不区分DirectMessage和GroupMessage
/// - 完全符合需求文档的统一消息架构

use crate::application::handlers::PaginatedResult;
use crate::domain::{
    entities::message::Message,
    entities::group::Group,
    values::*,
    DomainResult, DomainError
};
use crate::ports::{MessageRepository, GroupRepository, UserRepository, EventPublisher};
use std::sync::Arc;
use chrono::Utc;

#[derive(Clone)]
pub struct MessageApplicationService {
    message_repository: Arc<dyn MessageRepository>,
    group_repository: Arc<dyn GroupRepository>,
    user_repository: Arc<dyn UserRepository>,
    event_publisher: Arc<dyn EventPublisher>,
}

impl MessageApplicationService {
    pub fn new(
        message_repository: Arc<dyn MessageRepository>,
        group_repository: Arc<dyn GroupRepository>,
        user_repository: Arc<dyn UserRepository>,
        event_publisher: Arc<dyn EventPublisher>,
    ) -> Self {
        Self {
            message_repository,
            group_repository,
            user_repository,
            event_publisher,
        }
    }
    
    // ===== 统一消息API实现 =====
    
    /// 发送消息到群组（统一接口）
    /// 支持私聊和群聊，都通过group_id管理
    pub async fn send_message_to_group(
        &self,
        group_id: String,
        sender_id: i64,
        content: String,
        message_type: String,
        reply_to_message_id: Option<String>,
        mentioned_users: Vec<String>,
        latitude: Option<f64>,
        longitude: Option<f64>,
    ) -> DomainResult<Message> {
        // 1. 验证群组存在且用户有权限
        let group = self.group_repository
            .find_by_id(&GroupId::from(group_id.clone()))
            .await?
            .ok_or_else(|| DomainError::NotFound { 
                entity: "Group".to_string(), 
                id: group_id.clone() 
            })?;
        
        // 2. 验证用户是群组成员
        let is_member = self.group_repository
            .is_user_member(&GroupId::from(group_id.clone()), &UserId::from(sender_id))
            .await?;
        
        if !is_member {
            return Err(DomainError::PermissionDenied { 
                action: "发送消息".to_string(),
                reason: "用户不是群组成员".to_string(),
            });
        }
        
        // 3. 创建消息
        let message_id = MessageId::generate();
        let now = Utc::now();
        
        let message = Message {
            id: message_id,
            group_id: GroupId::from(group_id),
            sender_id: UserId::from(sender_id),
            content,
            message_type: message_type.parse().unwrap_or(crate::domain::entities::message::MessageType::Text),
            media_type: None, // TODO: 根据message_type设置
            media_url: None,
            media_size: None,
            reply_to_message_id: reply_to_message_id.map(MessageId::from),
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
            events: vec![],
        };
        
        // 4. 保存消息
        let saved_message = self.message_repository
            .create(message)
            .await?;
        
        // 5. 发布领域事件
        // TODO: 发布消息发送事件，用于实时推送
        
        Ok(saved_message)
    }
    
    /// 获取群组消息历史（统一接口）
    pub async fn get_group_messages(
        &self,
        group_id: String,
        user_id: i64,
        cursor: Option<String>,
        limit: i32,
        include_system_messages: bool,
    ) -> DomainResult<PaginatedResult<Message>> {
        // 1. 验证用户是群组成员
        let is_member = self.group_repository
            .is_user_member(&GroupId::from(group_id.clone()), &UserId::from(user_id))
            .await?;
        
        if !is_member {
            return Err(DomainError::PermissionDenied { 
                action: "查看消息历史".to_string(),
                reason: "用户不是群组成员".to_string(),
            });
        }
        
        // 2. 查询消息历史
        let messages = self.message_repository
            .find_by_group_id_paginated(
                &GroupId::from(group_id),
                cursor.as_deref(),
                limit,
                include_system_messages,
            )
            .await?;
        
        Ok(messages)
    }
    
    /// 删除消息（统一接口）
    pub async fn delete_message(
        &self,
        message_id: String,
        user_id: i64,
        delete_for_everyone: bool,
    ) -> DomainResult<()> {
        // 1. 获取消息
        let mut message = self.message_repository
            .find_by_id(&MessageId::from(message_id.clone()))
            .await?
            .ok_or_else(|| DomainError::NotFound { 
                entity: "Message".to_string(), 
                id: message_id 
            })?;
        
        // 2. 验证权限
        if message.sender_id != UserId::from(user_id) && !delete_for_everyone {
            return Err(DomainError::PermissionDenied { 
                action: "删除消息".to_string(),
                reason: "只能删除自己的消息".to_string(),
            });
        }
        
        // 3. 软删除消息
        message.is_deleted = true;
        message.deleted_at = Some(Utc::now());
        message.updated_at = Utc::now();
        
        self.message_repository
            .update(message)
            .await?;
        
        Ok(())
    }
    
    /// 标记群组消息已读（统一接口）
    pub async fn mark_group_messages_read(
        &self,
        group_id: String,
        user_id: i64,
        message_id: Option<String>,
    ) -> DomainResult<()> {
        // 1. 验证用户是群组成员
        let is_member = self.group_repository
            .is_user_member(&GroupId::from(group_id.clone()), &UserId::from(user_id))
            .await?;
        
        if !is_member {
            return Err(DomainError::PermissionDenied { 
                action: "标记消息已读".to_string(),
                reason: "用户不是群组成员".to_string(),
            });
        }
        
        // 2. 标记消息已读
        if let Some(msg_id) = message_id {
            // 标记到指定消息为已读
            self.message_repository
                .mark_read_until_message(
                    &GroupId::from(group_id),
                    &UserId::from(user_id),
                    &MessageId::from(msg_id),
                )
                .await?;
        } else {
            // 标记所有消息为已读
            self.message_repository
                .mark_all_read(
                    &GroupId::from(group_id),
                    &UserId::from(user_id),
                )
                .await?;
        }
        
        Ok(())
    }
    
    /// 获取群组未读消息数（统一接口）
    pub async fn get_group_unread_count(
        &self,
        group_id: String,
        user_id: i64,
    ) -> DomainResult<i32> {
        // 1. 验证用户是群组成员
        let is_member = self.group_repository
            .is_user_member(&GroupId::from(group_id.clone()), &UserId::from(user_id))
            .await?;
        
        if !is_member {
            return Ok(0); // 非成员返回0
        }
        
        // 2. 获取未读数
        let count = self.message_repository
            .get_unread_count(
                &GroupId::from(group_id),
                &UserId::from(user_id),
            )
            .await?;
        
        Ok(count)
    }
    
    /// 获取所有群组未读消息数（统一接口）
    pub async fn get_all_groups_unread_count(
        &self,
        user_id: i64,
    ) -> DomainResult<crate::presentation::http::controllers::message_controller::UnreadCountResponse> {
        // 1. 获取用户所有群组
        let user_groups = self.group_repository
            .find_by_user_id(&UserId::from(user_id))
            .await?;
        
        // 2. 获取每个群组的未读数
        let mut total_unread = 0;
        let mut groups = Vec::new();
        
        for group in user_groups {
            let unread_count = self.message_repository
                .get_unread_count(&group.id, &UserId::from(user_id))
                .await?;
            
            total_unread += unread_count;
            
            if unread_count > 0 {
                groups.push(crate::presentation::http::controllers::message_controller::GroupUnreadCountResponse {
                    group_id: group.id.to_string(),
                    unread_count,
                });
            }
        }
        
        Ok(crate::presentation::http::controllers::message_controller::UnreadCountResponse {
            total_unread,
            groups: if groups.is_empty() { None } else { Some(groups) },
        })
    }
    
    /// 获取用户群组列表（统一接口）
    pub async fn get_user_groups(
        &self,
        user_id: i64,
        cursor: Option<String>,
        limit: i32,
        include_archived: bool,
    ) -> DomainResult<PaginatedResult<GroupWithLastMessage>> {
        // 获取用户群组列表，包含最后一条消息信息
        let groups = self.group_repository
            .find_by_user_id_with_last_message(
                &UserId::from(user_id),
                cursor.as_deref(),
                limit,
                include_archived,
            )
            .await?;
        
        Ok(groups)
    }
}

/// 群组信息包含最后一条消息
#[derive(Debug, Clone)]
pub struct GroupWithLastMessage {
    pub group_id: String,
    pub name: String,
    pub group_type: String,
    pub avatar_url: Option<String>,
    pub last_message: Option<LastMessageInfo>,
    pub unread_count: i32,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_archived: bool,
    pub member_count: i32,
}

/// 最后一条消息信息
#[derive(Debug, Clone)]
pub struct LastMessageInfo {
    pub message_id: String,
    pub sender_id: String,
    pub sender_name: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: String,
}


