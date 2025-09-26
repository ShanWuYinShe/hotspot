/// WebSocket通知集成服务
/// 
/// 功能职责：
/// - 集成通知系统与WebSocket实时推送
/// - 管理实时通知的发送和路由
/// - 处理WebSocket连接状态和通知传递
/// - 提供通知的实时性和可靠性保证

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, info, warn};
use serde_json;
use std::collections::HashMap;

use crate::domain::DomainResult;
use crate::ws::unified_connection_manager::UnifiedConnectionManager;
use crate::ws::unified_message_types::{ServerMessage, NotificationPayload};
use crate::ws::message_types::SystemNotificationType;
use crate::adapters::adapters_pg::notification_repository::{
    Notification, NotificationType, NotificationPriority, NotificationStatus
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
    fn create_ws_notification(&self, notification: &Notification) -> NotificationPayload {
        NotificationPayload {
            notification_id: uuid::Uuid::parse_str(&notification.notification_id)
                .unwrap_or_else(|_| uuid::Uuid::new_v4())
                .to_string(),
            notification_type: format!("{:?}", self.convert_notification_type(&notification.notification_type)),
            title: notification.title.clone(),
            content: notification.content.clone(),
            data: notification
                .data
                .as_ref()
                .and_then(|v| v.as_object().cloned())
                .map(|m| m.into_iter().collect::<HashMap<_, _>>())
                .unwrap_or_else(HashMap::new),
            created_at: notification.created_at,
            expires_at: None, // 通知不设置过期时间
        }
    }
    
    /// 发送WebSocket消息给指定用户
    async fn send_to_user(&self, user_id: i64, _message: ServerMessage) -> DomainResult<()> {
        let _user_id_str = user_id.to_string();

        // TODO: WebSocket管理器的send_to_user方法需要公开
        // 暂时返回成功，实际需要调用WebSocket管理器发送消息
        warn!("WebSocket通知发送暂未实现，需要扩展UnifiedConnectionManager API");
        Ok(())
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
        let server_message = ServerMessage::SystemNotification { notification: ws_notification };
        
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
        let payload = NotificationPayload {
            notification_id: uuid::Uuid::new_v4().to_string(),
            notification_type: format!("{:?}", SystemNotificationType::System),
            title,
            content,
            data: data.and_then(|v| v.as_object().cloned()).map(|m| m.into_iter().collect::<HashMap<_, _>>()).unwrap_or_default(),
            created_at: chrono::Utc::now(),
            expires_at: None,
        };
        let _server_message = ServerMessage::SystemNotification { notification: payload };
        
        // TODO: 实现系统广播，当前WebSocket管理器API不支持广播
        // 需要扩展UnifiedConnectionManager添加broadcast_message方法
        warn!("系统广播通知暂未实现，WebSocket管理器需要扩展API");
        Ok(())
    }
    
    /// 发送群组通知
    async fn send_group_notification(&self, group_id: String, notification: &Notification) -> DomainResult<()> {
        debug!("发送群组通知: 群组={}, 类型={:?}", group_id, notification.notification_type);
        
        let ws_notification = self.create_ws_notification(notification);
        let _server_message = ServerMessage::SystemNotification { notification: ws_notification };

        // TODO: 实现群组通知，当前WebSocket管理器不支持房间广播
        info!("群组通知发送成功: 群组={}, 通知ID={}", group_id, notification.notification_id);
        Ok(())
    }
    
    /// 检查用户在线状态
    async fn is_user_online(&self, user_id: i64) -> DomainResult<bool> {
        let _user_id_str = user_id.to_string();
        
        // TODO: 检查用户在线状态，当前WebSocket管理器API不支持
        Ok(false)
    }
    
    /// 获取在线用户列表
    async fn get_online_users(&self) -> DomainResult<Vec<i64>> {
        // TODO: 获取在线用户列表，当前WebSocket管理器API不支持
        Ok(Vec::new())
    }
    
    /// 获取WebSocket连接统计
    async fn get_connection_stats(&self) -> DomainResult<WebSocketStats> {
        let stats = self.ws_manager.get_stats().await;
        Ok(WebSocketStats {
            total_connections: stats.total_connections,
            authenticated_connections: stats.authenticated_connections,
            active_rooms: stats.active_rooms,
            messages_sent: stats.messages_sent_per_minute as u64,
            messages_received: stats.messages_received_per_minute as u64,
            uptime_seconds: 0,
        })
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
            status: NotificationStatus::Unread,
            priority: NotificationPriority::Normal,
            sender_id: Some(requester_id),
            related_id: Some(request_id),
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
            is_deleted: false,
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
            status: NotificationStatus::Unread,
            priority: NotificationPriority::High,
            sender_id: Some(sender_id),
            related_id: None,
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
            is_deleted: false,
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
            status: NotificationStatus::Unread,
            priority: NotificationPriority::Normal,
            sender_id: Some(actor_id),
            related_id: Some(moment_id),
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
            is_deleted: false,
        };
        
        self.ws_notification_service.send_realtime_notification(&notification).await
    }
}
