/// 统一通知协调器
/// 
/// 功能职责：
/// - 协调通知系统和WebSocket实时推送
/// - 提供统一的通知发送接口
/// - 管理通知的持久化和实时推送
/// - 处理通知发送的容错和重试机制

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::domain::DomainResult;
use crate::adapters::adapters_pg::notification_repository::{
    Notification, NotificationType, NotificationPriority, NotificationStatus
};
use crate::application::services::notification_application_service::NotificationApplicationService;
use crate::application::services::websocket_notification_service::WebSocketNotificationService;

/// 统一通知协调器接口
#[async_trait]
pub trait UnifiedNotificationCoordinator: Send + Sync {
    /// 发送完整通知（持久化 + 实时推送）
    async fn send_complete_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Notification>;
    
    /// 批量发送完整通知
    async fn send_bulk_complete_notifications(
        &self,
        user_ids: Vec<i64>,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Vec<Notification>>;
    
    /// 发送系统广播通知
    async fn send_system_broadcast(
        &self,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
    ) -> DomainResult<()>;
    
    /// 发送群组通知
    async fn send_group_notification(
        &self,
        group_id: String,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        sender_id: Option<i64>,
    ) -> DomainResult<()>;
    
    /// 重新发送失败的通知
    async fn retry_failed_notifications(&self, user_id: i64) -> DomainResult<i32>;
}

/// 统一通知协调器实现
pub struct UnifiedNotificationCoordinatorImpl {
    notification_service: Arc<dyn NotificationApplicationService>,
    websocket_service: Arc<dyn WebSocketNotificationService>,
}

impl UnifiedNotificationCoordinatorImpl {
    pub fn new(
        notification_service: Arc<dyn NotificationApplicationService>,
        websocket_service: Arc<dyn WebSocketNotificationService>,
    ) -> Self {
        Self {
            notification_service,
            websocket_service,
        }
    }
    
    /// 尝试发送实时通知，失败时记录但不中断流程
    async fn try_send_realtime(&self, notification: &Notification) -> bool {
        match self.websocket_service.send_realtime_notification(notification).await {
            Ok(_) => {
                debug!("实时通知发送成功: ID={}", notification.notification_id);
                true
            }
            Err(e) => {
                warn!("实时通知发送失败: ID={}, 错误={:?}", notification.notification_id, e);
                false
            }
        }
    }
    
    /// 处理通知发送的完整流程
    async fn process_complete_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Notification> {
        // 1. 首先持久化通知到数据库
        let notification = self.notification_service.send_notification(
            user_id,
            notification_type,
            title,
            content,
            data,
            priority,
            sender_id,
        ).await?;
        
        // 2. 尝试发送实时通知
        let realtime_success = self.try_send_realtime(&notification).await;
        
        // 3. 记录发送结果
        if realtime_success {
            info!("完整通知发送成功: ID={}, 用户={}", notification.notification_id, user_id);
        } else {
            info!("通知已持久化，实时推送失败: ID={}, 用户={}", notification.notification_id, user_id);
        }
        
        Ok(notification)
    }
}

#[async_trait]
impl UnifiedNotificationCoordinator for UnifiedNotificationCoordinatorImpl {
    /// 发送完整通知（持久化 + 实时推送）
    async fn send_complete_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Notification> {
        debug!("发送完整通知: 用户={}, 类型={:?}, 标题={}", user_id, notification_type, title);
        
        self.process_complete_notification(
            user_id,
            notification_type,
            title,
            content,
            data,
            priority,
            sender_id,
        ).await
    }
    
    /// 批量发送完整通知
    async fn send_bulk_complete_notifications(
        &self,
        user_ids: Vec<i64>,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Vec<Notification>> {
        debug!("批量发送完整通知: 用户数={}, 类型={:?}", user_ids.len(), notification_type);
        
        let mut notifications = Vec::new();
        let mut success_count = 0;
        let mut failure_count = 0;
        
        for user_id in user_ids {
            match self.process_complete_notification(
                user_id,
                notification_type.clone(),
                title.clone(),
                content.clone(),
                data.clone(),
                priority.clone(),
                sender_id,
            ).await {
                Ok(notification) => {
                    notifications.push(notification);
                    success_count += 1;
                }
                Err(e) => {
                    error!("批量通知发送失败: 用户={}, 错误={:?}", user_id, e);
                    failure_count += 1;
                }
            }
        }
        
        info!("批量通知发送完成: 成功={}, 失败={}", success_count, failure_count);
        Ok(notifications)
    }
    
    /// 发送系统广播通知
    async fn send_system_broadcast(
        &self,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
    ) -> DomainResult<()> {
        debug!("发送系统广播通知: 标题={}", title);
        
        // 发送WebSocket广播
        match self.websocket_service.broadcast_system_notification(title.clone(), content.clone(), data.clone()).await {
            Ok(_) => {
                info!("系统广播通知发送成功: 标题={}", title);
            }
            Err(e) => {
                error!("系统广播通知发送失败: 标题={}, 错误={:?}", title, e);
                return Err(e);
            }
        }
        
        // TODO: 可以选择是否也要为所有在线用户创建持久化通知
        // 这取决于业务需求，系统广播通常不需要持久化
        
        Ok(())
    }
    
    /// 发送群组通知
    async fn send_group_notification(
        &self,
        group_id: String,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        sender_id: Option<i64>,
    ) -> DomainResult<()> {
        debug!("发送群组通知: 群组={}, 类型={:?}", group_id, notification_type);
        
        // 创建临时通知对象用于WebSocket发送
        let temp_notification = Notification {
            notification_id: uuid::Uuid::new_v4().to_string(),
            user_id: 0, // 群组通知不针对特定用户
            notification_type,
            title,
            content,
            data,
            status: NotificationStatus::Unread,
            priority: NotificationPriority::Normal,
            sender_id,
            related_id: Some(group_id.clone()),
            expires_at: None,
            read_at: None,
            created_at: chrono::Utc::now(),
            is_deleted: false,
        };
        
        // 发送群组WebSocket通知
        match self.websocket_service.send_group_notification(group_id.clone(), &temp_notification).await {
            Ok(_) => {
                info!("群组通知发送成功: 群组={}", group_id);
            }
            Err(e) => {
                error!("群组通知发送失败: 群组={}, 错误={:?}", group_id, e);
                return Err(e);
            }
        }
        
        // TODO: 可以选择是否为群组成员创建个人通知
        // 这取决于业务需求和通知类型
        
        Ok(())
    }
    
    /// 重新发送失败的通知
    async fn retry_failed_notifications(&self, user_id: i64) -> DomainResult<i32> {
        debug!("重新发送失败的通知: 用户={}", user_id);
        
        // 获取用户的未读通知
        let notifications = self.notification_service.get_user_notifications(
            user_id,
            Some(NotificationStatus::Unread),
            0,
            50, // 限制重试数量
        ).await?;
        
        let mut retry_count = 0;
        
        for notification in notifications {
            // 检查用户是否在线
            if let Ok(true) = self.websocket_service.is_user_online(user_id).await {
                // 尝试重新发送实时通知
                if self.try_send_realtime(&notification).await {
                    retry_count += 1;
                }
            }
        }
        
        info!("通知重试完成: 用户={}, 重试数量={}", user_id, retry_count);
        Ok(retry_count)
    }
}

impl Clone for UnifiedNotificationCoordinatorImpl {
    fn clone(&self) -> Self {
        Self {
            notification_service: Arc::clone(&self.notification_service),
            websocket_service: Arc::clone(&self.websocket_service),
        }
    }
}

/// 通知协调器工厂
pub struct NotificationCoordinatorFactory;

impl NotificationCoordinatorFactory {
    /// 创建统一通知协调器
    pub fn create_coordinator(
        notification_service: Arc<dyn NotificationApplicationService>,
        websocket_service: Arc<dyn WebSocketNotificationService>,
    ) -> Arc<dyn UnifiedNotificationCoordinator> {
        Arc::new(UnifiedNotificationCoordinatorImpl::new(
            notification_service,
            websocket_service,
        ))
    }
    
    /// 创建带有默认配置的协调器
    pub fn create_default_coordinator(
        notification_service: Arc<dyn NotificationApplicationService>,
        websocket_service: Arc<dyn WebSocketNotificationService>,
    ) -> Arc<dyn UnifiedNotificationCoordinator> {
        let coordinator = UnifiedNotificationCoordinatorImpl::new(
            notification_service,
            websocket_service,
        );
        
        Arc::new(coordinator)
    }
}
