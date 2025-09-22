/// WebSocket通知集成服务
/// 
/// 功能职责：
/// - 集成通知系统与WebSocket实时推送
/// - 管理实时通知的发送和路由
/// - 处理WebSocket连接状态和通知传递
/// - 提供通知的实时性和可靠性保证

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};
use serde_json;

use crate::domain::{DomainResult, DomainError, UserId};
use crate::ws::unified_connection_manager::UnifiedConnectionManager;
use crate::ws::unified_message_types::{ServerMessage, NotificationPayload};
use crate::ws::message_types::{SystemNotificationData, SystemNotificationType};
use crate::adapters::adapters_pg::notification_repository::{
    Notification, NotificationType, NotificationPriority
};
use crate::application::services::notification_application_service::NotificationApplicationService;

/// WebSocket通知集成服务接口
#[async_trait]
pub trait WebSocketNotificationService: Send + Sync {
    /// 发送实时通知
    async fn send_realtime_notification(&self, notification: &Notification) -> DomainResult<()>;
    
    /// 批量发送实时通知
    async fn send_bulk_realtime_notifications(&self, notifications: &[Notification]) -> DomainResult<()>;
    
    /// 发送系统广播通知
    async fn broadcast_system_notification(&self, title: String, content: String, data: Option<serde_json::Value>) -> DomainResult<()>;
    
    /// 发送群组通知
    async fn send_group_notification(&self, group_id: String, notification: &Notification) -> DomainResult<()>;
    
    /// 检查用户在线状态
    async fn is_user_online(&self, user_id: i64) -> DomainResult<bool>;
    
    /// 获取在线用户列表
    async fn get_online_users(&self) -> DomainResult<Vec<i64>>;
    
    /// 获取WebSocket连接统计
    async fn get_connection_stats(&self) -> DomainResult<WebSocketStats>;
}

/// WebSocket连接统计
#[derive(Debug, Clone)]
pub struct WebSocketStats {
    pub total_connections: usize,
    pub authenticated_connections: usize,
    pub active_rooms: usize,
    pub messages_sent: u64,
    pub messages_received: u64,
    pub uptime_seconds: u64,
}

/// WebSocket通知集成服务实现
pub struct WebSocketNotificationServiceImpl {
    ws_manager: Arc<UnifiedConnectionManager>,
    notification_service: Arc<dyn NotificationApplicationService>,
}

impl WebSocketNotificationServiceImpl {
    pub fn new(
        ws_manager: Arc<UnifiedConnectionManager>,
        notification_service: Arc<dyn NotificationApplicationService>,
    ) -> Self {
        Self {
            ws_manager,
            notification_service,
        }
    }
    
    /// 将通知类型转换为WebSocket通知类型
    fn convert_notification_type(&self, notification_type: &NotificationType) -> SystemNotificationType {
        match notification_type {
            NotificationType::FriendRequest => SystemNotificationType::FriendRequest,
            NotificationType::FriendAccepted => SystemNotificationType::System,
            NotificationType::NewMessage => SystemNotificationType::NewMessage,
            NotificationType::GroupInvite => SystemNotificationType::GroupInvitation,
            NotificationType::MomentLike => SystemNotificationType::NewMoment,
            NotificationType::MomentComment => SystemNotificationType::NewMoment,
            NotificationType::System => SystemNotificationType::System,
            NotificationType::Security => SystemNotificationType::System,
        }
    }
    
    /// 创建WebSocket通知消息
    fn create_ws_notification(&self, notification: &Notification) -> SystemNotificationData {
        SystemNotificationData {
            notification_id: uuid::Uuid::parse_str(&notification.notification_id)
                .unwrap_or_else(|_| uuid::Uuid::new_v4()),
            notification_type: self.convert_notification_type(&notification.notification_type),
            title: notification.title.clone(),
            content: notification.content.clone(),
            data: notification.data.clone(),
            created_at: notification.created_at,
        }
    }
    
    /// 发送WebSocket消息给指定用户
    async fn send_to_user(&self, user_id: i64, message: ServerMessage) -> DomainResult<()> {
        let user_id_str = user_id.to_string();
        
        match self.ws_manager.send_to_user(&user_id_str, message).await {
            Ok(_) => {
                debug!("WebSocket通知发送成功: 用户={}", user_id);
                Ok(())
            }
            Err(e) => {
                warn!("WebSocket通知发送失败: 用户={}, 错误={:?}", user_id, e);
                // 不将WebSocket发送失败视为致命错误，因为用户可能离线
                Ok(())
            }
        }
    }
    
    /// 处理通知发送失败的回退策略
    async fn handle_notification_fallback(&self, notification: &Notification) -> DomainResult<()> {
        // 如果WebSocket发送失败，确保通知仍然保存在数据库中
        // 用户下次连接时可以获取未读通知
        debug!("WebSocket发送失败，通知已保存到数据库: ID={}", notification.notification_id);
        
        // TODO: 可以在这里添加其他推送渠道
        // 例如：推送通知服务、邮件通知等
        
        Ok(())
    }
}

#[async_trait]
impl WebSocketNotificationService for WebSocketNotificationServiceImpl {
    /// 发送实时通知
    async fn send_realtime_notification(&self, notification: &Notification) -> DomainResult<()> {
        debug!("发送实时通知: 用户={}, 类型={:?}, 标题={}", 
               notification.user_id, notification.notification_type, notification.title);
        
        // 检查用户是否在线
        let is_online = self.is_user_online(notification.user_id).await?;
        if !is_online {
            debug!("用户 {} 不在线，跳过实时通知发送", notification.user_id);
            return Ok(());
        }
        
        // 创建WebSocket通知消息
        let ws_notification = self.create_ws_notification(notification);
        let server_message = ServerMessage::SystemNotification(ws_notification);
        
        // 发送WebSocket消息
        match self.send_to_user(notification.user_id, server_message).await {
            Ok(_) => {
                info!("实时通知发送成功: ID={}, 用户={}", notification.notification_id, notification.user_id);
                Ok(())
            }
            Err(_) => {
                // 处理发送失败的回退策略
                self.handle_notification_fallback(notification).await
            }
        }
    }
    
    /// 批量发送实时通知
    async fn send_bulk_realtime_notifications(&self, notifications: &[Notification]) -> DomainResult<()> {
        debug!("批量发送实时通知: 数量={}", notifications.len());
        
        let mut success_count = 0;
        let mut failure_count = 0;
        
        for notification in notifications {
            match self.send_realtime_notification(notification).await {
                Ok(_) => success_count += 1,
                Err(_) => {
                    failure_count += 1;
                    // 继续处理其他通知，不因单个失败而中断
                }
            }
        }
        
        info!("批量实时通知发送完成: 成功={}, 失败={}", success_count, failure_count);
        Ok(())
    }
    
    /// 发送系统广播通知
    async fn broadcast_system_notification(&self, title: String, content: String, data: Option<serde_json::Value>) -> DomainResult<()> {
        debug!("发送系统广播通知: 标题={}", title);
        
        let notification_data = SystemNotificationData {
            notification_id: uuid::Uuid::new_v4(),
            notification_type: WsNotificationType::System,
            title,
            content,
            data,
            sender_id: None,
            priority: 3, // 系统通知默认高优先级
            created_at: chrono::Utc::now(),
            expires_at: None,
        };
        
        let server_message = ServerMessage::SystemNotification(notification_data);
        
        match self.ws_manager.broadcast_message(server_message).await {
            Ok(_) => {
                info!("系统广播通知发送成功");
                Ok(())
            }
            Err(e) => {
                error!("系统广播通知发送失败: {:?}", e);
                Err(DomainError::InternalError {
                    message: "系统广播通知发送失败".to_string(),
                })
            }
        }
    }
    
    /// 发送群组通知
    async fn send_group_notification(&self, group_id: String, notification: &Notification) -> DomainResult<()> {
        debug!("发送群组通知: 群组={}, 类型={:?}", group_id, notification.notification_type);
        
        let ws_notification = self.create_ws_notification(notification);
        let server_message = ServerMessage::SystemNotification(ws_notification);
        
        match self.ws_manager.send_to_room(&group_id, server_message).await {
            Ok(_) => {
                info!("群组通知发送成功: 群组={}, 通知ID={}", group_id, notification.notification_id);
                Ok(())
            }
            Err(e) => {
                warn!("群组通知发送失败: 群组={}, 错误={:?}", group_id, e);
                // 群组通知发送失败不视为致命错误
                Ok(())
            }
        }
    }
    
    /// 检查用户在线状态
    async fn is_user_online(&self, user_id: i64) -> DomainResult<bool> {
        let user_id_str = user_id.to_string();
        
        match self.ws_manager.is_user_connected(&user_id_str).await {
            Ok(is_connected) => Ok(is_connected),
            Err(_) => {
                // 如果检查失败，默认认为用户离线
                Ok(false)
            }
        }
    }
    
    /// 获取在线用户列表
    async fn get_online_users(&self) -> DomainResult<Vec<i64>> {
        match self.ws_manager.get_connected_users().await {
            Ok(user_ids) => {
                let mut online_users = Vec::new();
                for user_id_str in user_ids {
                    if let Ok(user_id) = user_id_str.parse::<i64>() {
                        online_users.push(user_id);
                    }
                }
                Ok(online_users)
            }
            Err(e) => {
                error!("获取在线用户列表失败: {:?}", e);
                Err(DomainError::InternalError {
                    message: "获取在线用户列表失败".to_string(),
                })
            }
        }
    }
    
    /// 获取WebSocket连接统计
    async fn get_connection_stats(&self) -> DomainResult<WebSocketStats> {
        match self.ws_manager.get_stats().await {
            Ok(stats) => {
                Ok(WebSocketStats {
                    total_connections: stats.total_connections,
                    authenticated_connections: stats.authenticated_connections,
                    active_rooms: stats.active_rooms,
                    messages_sent: stats.messages_sent,
                    messages_received: stats.messages_received,
                    uptime_seconds: stats.uptime.as_secs(),
                })
            }
            Err(e) => {
                error!("获取WebSocket统计失败: {:?}", e);
                Err(DomainError::InternalError {
                    message: "获取WebSocket统计失败".to_string(),
                })
            }
        }
    }
}

impl Clone for WebSocketNotificationServiceImpl {
    fn clone(&self) -> Self {
        Self {
            ws_manager: Arc::clone(&self.ws_manager),
            notification_service: Arc::clone(&self.notification_service),
        }
    }
}

/// WebSocket通知事件处理器
/// 
/// 用于处理各种业务事件并转换为实时通知
pub struct WebSocketNotificationEventHandler {
    ws_notification_service: Arc<dyn WebSocketNotificationService>,
}

impl WebSocketNotificationEventHandler {
    pub fn new(ws_notification_service: Arc<dyn WebSocketNotificationService>) -> Self {
        Self {
            ws_notification_service,
        }
    }
    
    /// 处理好友申请事件
    pub async fn handle_friend_request_event(&self, requester_id: i64, target_id: i64, request_id: String) -> DomainResult<()> {
        debug!("处理好友申请事件: {} -> {}", requester_id, target_id);
        
        // 创建好友申请通知
        let notification = Notification {
            notification_id: uuid::Uuid::new_v4().to_string(),
            user_id: target_id,
            notification_type: NotificationType::FriendRequest,
            title: "新的好友申请".to_string(),
            content: "有人向您发送了好友申请".to_string(),
            data: Some(serde_json::json!({
                "requester_id": requester_id,
                "request_id": request_id
            })),
            status: crate::adapters::adapters_pg::notification_repository::NotificationStatus::Unread,
            priority: NotificationPriority::Normal,
            sender_id: Some(requester_id),
            related_id: Some(request_id),
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
        };
        
        self.ws_notification_service.send_realtime_notification(&notification).await
    }
    
    /// 处理新消息事件
    pub async fn handle_new_message_event(&self, sender_id: i64, recipient_id: i64, message_content: String) -> DomainResult<()> {
        debug!("处理新消息事件: {} -> {}", sender_id, recipient_id);
        
        // 创建新消息通知
        let notification = Notification {
            notification_id: uuid::Uuid::new_v4().to_string(),
            user_id: recipient_id,
            notification_type: NotificationType::NewMessage,
            title: "新消息".to_string(),
            content: message_content,
            data: Some(serde_json::json!({
                "sender_id": sender_id
            })),
            status: crate::adapters::adapters_pg::notification_repository::NotificationStatus::Unread,
            priority: NotificationPriority::High,
            sender_id: Some(sender_id),
            related_id: None,
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
        };
        
        self.ws_notification_service.send_realtime_notification(&notification).await
    }
    
    /// 处理瞬间互动事件
    pub async fn handle_moment_interaction_event(&self, actor_id: i64, moment_owner_id: i64, interaction_type: &str, moment_id: String) -> DomainResult<()> {
        debug!("处理瞬间互动事件: {} {} 瞬间 {}", actor_id, interaction_type, moment_id);
        
        let (notification_type, title, content) = match interaction_type {
            "like" => (NotificationType::MomentLike, "瞬间获得点赞".to_string(), "有人点赞了您的瞬间".to_string()),
            "comment" => (NotificationType::MomentComment, "瞬间获得评论".to_string(), "有人评论了您的瞬间".to_string()),
            _ => return Ok(()), // 未知的互动类型
        };
        
        // 创建瞬间互动通知
        let notification = Notification {
            notification_id: uuid::Uuid::new_v4().to_string(),
            user_id: moment_owner_id,
            notification_type,
            title,
            content,
            data: Some(serde_json::json!({
                "actor_id": actor_id,
                "moment_id": moment_id,
                "interaction_type": interaction_type
            })),
            status: crate::adapters::adapters_pg::notification_repository::NotificationStatus::Unread,
            priority: NotificationPriority::Normal,
            sender_id: Some(actor_id),
            related_id: Some(moment_id),
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
        };
        
        self.ws_notification_service.send_realtime_notification(&notification).await
    }
}
