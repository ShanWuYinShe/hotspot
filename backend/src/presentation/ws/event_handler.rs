/// WebSocket事件处理器
/// 
/// 处理来自领域层的事件，转换为WebSocket通知
use async_trait::async_trait;
use std::sync::Arc;
use crate::domain::{UserEvent, GroupEvent, MessageEvent, MomentEvent};
use crate::ports::GroupRepository;
use crate::presentation::ws::{WSEventType, MessageData};

/// 事件处理器trait
#[async_trait]
pub trait EventHandler: Send + Sync {
    /// 处理用户事件
    async fn handle_user_event(&self, event: UserEvent) -> Result<Option<WSEventType>, EventHandlerError>;
    
    /// 处理群组事件
    async fn handle_group_event(&self, event: GroupEvent) -> Result<Option<WSEventType>, EventHandlerError>;
    
    /// 处理消息事件
    async fn handle_message_event(&self, event: MessageEvent) -> Result<Option<WSEventType>, EventHandlerError>;
    
    /// 处理瞬间事件
    async fn handle_moment_event(&self, event: MomentEvent) -> Result<Option<WSEventType>, EventHandlerError>;
}

/// WebSocket事件处理器实现
pub struct WebSocketEventHandler {
    group_repo: Arc<dyn GroupRepository>,
    user_repo: Arc<dyn crate::ports::UserRepository>,
}

impl WebSocketEventHandler {
    pub fn new(group_repo: Arc<dyn GroupRepository>, user_repo: Arc<dyn crate::ports::UserRepository>) -> Self {
        Self { group_repo, user_repo }
    }
}

// 不提供 Default，要求通过依赖注入传入仓储端口

#[async_trait]
impl EventHandler for WebSocketEventHandler {
    async fn handle_user_event(&self, event: UserEvent) -> Result<Option<WSEventType>, EventHandlerError> {
        let ws_event = match event {
            UserEvent::UserCreated { user_id, .. } => {
                // 用户创建事件，需要获取用户昵称
                let nickname = match self.user_repo.find_user_profile(user_id.value()).await {
                    Ok(Some(user)) => user.nickname,
                    _ => "新用户".to_string(),
                };
                Some(WSEventType::UserOnline { user_id: user_id.value(), nickname })
            }
            UserEvent::UserUpgraded { .. } => {
                // 用户升级事件可能不需要实时通知
                None
            }
            UserEvent::UserLocationUpdated { .. } => {
                // 位置更新事件可根据需要决定是否通知
                None
            }
            UserEvent::UserProfileUpdated { .. } => {
                // 资料更新事件可根据需要决定是否通知
                None
            }
            UserEvent::UserPasswordSet { .. } => {
                // 密码设置事件不需要实时通知
                None
            }
            UserEvent::UserDeleted { .. } => {
                // 用户删除事件不需要实时通知
                None
            }
            _ => None, // 其他用户事件暂不处理
        };
        
        Ok(ws_event)
    }
    
    async fn handle_group_event(&self, event: GroupEvent) -> Result<Option<WSEventType>, EventHandlerError> {
        let ws_event = match event {
            GroupEvent::GroupCreated { group_id, .. } => {
                // 查询群组名称
                let name = match self.group_repo.find_group_profile(&group_id.to_string()).await {
                    Ok(Some(group)) => group.name,
                    _ => "新群组".to_string(),
                };
                Some(WSEventType::GroupInfoUpdated { group_id: group_id.value().to_string(), name })
            }
            GroupEvent::GroupMemberJoined { group_id, user_id, .. } => {
                // 获取用户昵称需要额外查询，这里使用默认值
                Some(WSEventType::UserJoinedGroup { user_id: user_id.value(), group_id: group_id.value().to_string(), nickname: "用户".to_string() })
            }
            GroupEvent::GroupMemberLeft { group_id, user_id, .. } => {
                Some(WSEventType::UserLeftGroup { user_id: user_id.value(), group_id: group_id.value().to_string() })
            }
            GroupEvent::GroupUpdated { group_id, .. } => {
                // 查询群组最新名称，遵循最佳实践，不使用占位值
                let name = match self.group_repo.find_group_profile(&group_id.to_string()).await {
                    Ok(Some(group)) => group.name,
                    _ => "群组信息已更新".to_string(),
                };
                Some(WSEventType::GroupInfoUpdated { group_id: group_id.value().to_string(), name })
            }
            _ => None, // 其他群组事件暂不处理
        };
        
        Ok(ws_event)
    }
    
    async fn handle_message_event(&self, event: MessageEvent) -> Result<Option<WSEventType>, EventHandlerError> {
        let ws_event = match event {
            MessageEvent::MessageSent { 
                message_id, 
                sender_id,
                recipient_type,
                recipient_id,
                .. 
            } => {
                let message_data = MessageData {
                    id: message_id.value().to_string(),
                    sender_id: sender_id.value(),
                    sender_nickname: "用户".to_string(), // 需要额外查询获取昵称
                    content: "新消息".to_string(), // 需要额外查询获取内容
                    message_type: "text".to_string(), // 需要额外查询获取消息类型
                    created_at: chrono::Utc::now(), // 使用当前时间
                };
                
                // 根据接收者类型决定事件类型
                match recipient_type {
                    crate::domain::events::MessageRecipientType::Group => {
                        // 群组消息，recipient_id 是群组ID
                        Some(WSEventType::NewMessage { 
                            group_id: recipient_id, 
                            message: message_data 
                        })
                    }
                    crate::domain::events::MessageRecipientType::User => {
                        // 私聊消息，可以考虑其他处理方式
                        None
                    }
                }
            }
            MessageEvent::MessageDeleted { .. } => {
                // 消息删除事件可根据需要决定是否通知
                None
            }
            _ => None, // 其他消息事件暂不处理
        };
        
        Ok(ws_event)
    }
    
    async fn handle_moment_event(&self, _event: MomentEvent) -> Result<Option<WSEventType>, EventHandlerError> {
        // 瞬间事件通常不需要实时WebSocket通知
        // 可根据业务需求决定是否处理
        Ok(None)
    }
}

/// 事件处理器错误
#[derive(Debug, thiserror::Error)]
pub enum EventHandlerError {
    #[error("事件处理失败: {reason}")]
    ProcessingFailed { reason: String },
    
    #[error("事件序列化失败: {error}")]
    SerializationError { error: String },
    
    #[error("WebSocket发送失败: {error}")]
    WebSocketError { error: String },
}
