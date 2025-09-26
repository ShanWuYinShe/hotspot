/// 通知推送应用服务
/// 
/// 功能职责：
/// - 协调通知推送的业务逻辑
/// - 管理通知的创建、发送和状态更新
/// - 处理通知设置和偏好管理
/// - 提供通知统计和分析
/// - 集成多种推送渠道（WebSocket、推送通知等）

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, info, warn};

use crate::domain::{DomainResult, DomainError};
use crate::ports::{UserRepository, EventPublisher};
use crate::adapters::adapters_pg::notification_repository::{
    NotificationRepository, Notification, NotificationSettings, NotificationType, 
    NotificationStatus, NotificationPriority
};

/// 通知推送应用服务接口
#[async_trait]
pub trait NotificationApplicationService: Send + Sync {
    /// 发送通知
    async fn send_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Notification>;
    
    /// 批量发送通知
    async fn send_bulk_notifications(
        &self,
        user_ids: Vec<i64>,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Vec<Notification>>;
    
    /// 获取用户通知列表
    async fn get_user_notifications(
        &self,
        user_id: i64,
        status: Option<NotificationStatus>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Notification>>;
    
    /// 标记通知为已读
    async fn mark_notification_read(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 批量标记通知为已读
    async fn mark_notifications_read(&self, notification_ids: Vec<String>, user_id: i64) -> DomainResult<()>;
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()>;
    
    /// 获取未读通知数量
    async fn get_unread_notification_count(&self, user_id: i64) -> DomainResult<i64>;
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<NotificationSettings>;
    
    /// 更新通知设置
    async fn update_notification_settings(&self, user_id: i64, settings: NotificationSettings) -> DomainResult<NotificationSettings>;
    
    /// 删除通知
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 清理过期通知
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64>;
}

/// 通知推送应用服务实现
pub struct NotificationApplicationServiceImpl {
    notification_repository: Arc<dyn NotificationRepository>,
    user_repository: Arc<dyn UserRepository>,
    event_publisher: Arc<dyn EventPublisher>,
}

impl NotificationApplicationServiceImpl {
    pub fn new(
        notification_repository: Arc<dyn NotificationRepository>,
        user_repository: Arc<dyn UserRepository>,
        event_publisher: Arc<dyn EventPublisher>,
    ) -> Self {
        Self {
            notification_repository,
            user_repository,
            event_publisher,
        }
    }
    
    /// 检查用户是否允许接收此类型的通知
    async fn should_send_notification(
        &self,
        user_id: i64,
        notification_type: &NotificationType,
    ) -> DomainResult<bool> {
        // 获取用户通知设置
        let settings = match self.notification_repository.get_notification_settings(user_id).await? {
            Some(settings) => settings,
            None => NotificationSettings::default(), // 使用默认设置
        };
        
        // 检查是否在静默时间
        if let (Some(start), Some(end)) = (settings.quiet_hours_start, settings.quiet_hours_end) {
            let now = chrono::Utc::now().time();
            if now >= start && now <= end {
                debug!("用户 {} 处于静默时间，跳过通知发送", user_id);
                return Ok(false);
            }
        }
        
        // 根据通知类型检查设置
        let should_send = match notification_type {
            NotificationType::FriendRequest | NotificationType::FriendAccepted => settings.friend_requests,
            NotificationType::NewMessage => settings.messages,
            NotificationType::GroupInvite => settings.group_invites,
            NotificationType::MomentLike | NotificationType::MomentComment => settings.moment_interactions,
            NotificationType::System | NotificationType::Security => settings.system_notifications,
        };
        
        Ok(should_send)
    }
    
    /// 创建通知对象
    fn create_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> Notification {
        let notification_id = uuid::Uuid::new_v4().to_string();
        let created_at = chrono::Utc::now();
        
        // 根据优先级设置过期时间
        let expires_at = match priority {
            NotificationPriority::Low => Some(created_at + chrono::Duration::days(7)),
            NotificationPriority::Normal => Some(created_at + chrono::Duration::days(30)),
            NotificationPriority::High => Some(created_at + chrono::Duration::days(90)),
            NotificationPriority::Urgent => None, // 紧急通知不过期
        };
        
        Notification {
            notification_id,
            user_id,
            notification_type,
            title,
            content,
            data,
            status: NotificationStatus::Unread,
            priority,
            sender_id,
            related_id: None,
            expires_at,
            read_at: None,
            created_at,
            is_deleted: false,
        }
    }
    
    /// 发送推送通知（WebSocket、推送服务等）
    async fn send_push_notification(&self, notification: &Notification) -> DomainResult<()> {
        debug!("发送推送通知: 用户={}, 类型={:?}, 标题={}", 
               notification.user_id, notification.notification_type, notification.title);
        
        // 集成WebSocket推送
        // 这里应该调用WebSocket通知服务发送实时通知
        // 由于架构依赖关系，WebSocket集成将在更高层处理

        // 集成移动推送服务
        // TODO: 集成移动推送服务

        // 集成邮件通知
        // TODO: 集成邮件通知

        // 发布通知事件
        let _event_data = serde_json::json!({
            "notification_id": notification.notification_id,
            "user_id": notification.user_id,
            "type": format!("{:?}", notification.notification_type),
            "title": notification.title,
            "content": notification.content,
            "priority": format!("{:?}", notification.priority)
        });

        // TODO: 发布到事件总线
        // self.event_publisher.publish("notification_sent", event_data).await?;
        
        Ok(())
    }
}

#[async_trait]
impl NotificationApplicationService for NotificationApplicationServiceImpl {
    /// 发送通知
    async fn send_notification(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Notification> {
        debug!("发送通知: 用户={}, 类型={:?}, 标题={}", user_id, notification_type, title);
        
        // 验证接收用户存在
        let _user = self.user_repository.find_user_profile(user_id).await?
            .ok_or_else(|| DomainError::NotFound { resource: "User".to_string() })?;
        
        // 检查是否应该发送通知
        if !self.should_send_notification(user_id, &notification_type).await? {
            debug!("用户 {} 的通知设置不允许发送此类型通知", user_id);
            return Err(DomainError::ValidationError {
                field: "notification_settings".to_string(),
                reason: "用户通知设置不允许发送此类型通知".to_string(),
            });
        }
        
        // 创建通知对象
        let notification = self.create_notification(
            user_id,
            notification_type,
            title,
            content,
            data,
            priority,
            sender_id,
        );
        
        // 保存通知到数据库
        let saved_notification = self.notification_repository.create_notification(notification).await?;
        
        // 发送推送通知
        self.send_push_notification(&saved_notification).await?;
        
        info!("通知发送成功: ID={}, 用户={}", saved_notification.notification_id, user_id);
        Ok(saved_notification)
    }
    
    /// 批量发送通知
    async fn send_bulk_notifications(
        &self,
        user_ids: Vec<i64>,
        notification_type: NotificationType,
        title: String,
        content: String,
        data: Option<serde_json::Value>,
        priority: NotificationPriority,
        sender_id: Option<i64>,
    ) -> DomainResult<Vec<Notification>> {
        debug!("批量发送通知: 用户数={}, 类型={:?}", user_ids.len(), notification_type);
        
        let mut notifications = Vec::new();
        
        for user_id in user_ids {
            // 检查是否应该发送通知
            if self.should_send_notification(user_id, &notification_type).await.unwrap_or(false) {
                let notification = self.create_notification(
                    user_id,
                    notification_type.clone(),
                    title.clone(),
                    content.clone(),
                    data.clone(),
                    priority.clone(),
                    sender_id,
                );
                notifications.push(notification);
            }
        }
        
        if notifications.is_empty() {
            return Ok(vec![]);
        }
        
        // 批量保存通知
        let saved_notifications = self.notification_repository.create_notifications(notifications).await?;
        
        // 批量发送推送通知
        for notification in &saved_notifications {
            if let Err(e) = self.send_push_notification(notification).await {
                warn!("发送推送通知失败: {}", e);
            }
        }
        
        info!("批量通知发送成功: 数量={}", saved_notifications.len());
        Ok(saved_notifications)
    }
    
    /// 获取用户通知列表
    async fn get_user_notifications(
        &self,
        user_id: i64,
        status: Option<NotificationStatus>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Notification>> {
        debug!("获取用户通知列表: 用户={}, 状态={:?}", user_id, status);

        let notifications = self.notification_repository.get_user_notifications(
            user_id,
            status,
            cursor,
            limit,
        ).await?;

        Ok(notifications)
    }
    
    /// 标记通知为已读
    async fn mark_notification_read(&self, notification_id: String, user_id: i64) -> DomainResult<()> {
        debug!("标记通知已读: ID={}, 用户={}", notification_id, user_id);
        
        // 验证通知存在且属于当前用户
        let notification = self.notification_repository.find_by_id(notification_id.clone()).await?;
        match notification {
            Some(notif) if notif.user_id == user_id => {
                self.notification_repository.mark_notification_read(notification_id, user_id).await?;
                info!("通知已标记为已读: ID={}", notif.notification_id);
                Ok(())
            }
            Some(_) => Err(DomainError::ValidationError {
                field: "notification_id".to_string(),
                reason: "无权限操作此通知".to_string(),
            }),
            None => Err(DomainError::NotFound {
                resource: "通知".to_string(),
            }),
        }
    }
    
    /// 批量标记通知为已读
    async fn mark_notifications_read(&self, notification_ids: Vec<String>, user_id: i64) -> DomainResult<()> {
        debug!("批量标记通知已读: 数量={}, 用户={}", notification_ids.len(), user_id);
        
        // TODO: 验证所有通知都属于当前用户
        
        self.notification_repository.mark_notifications_read(notification_ids, user_id).await?;
        
        info!("批量标记通知已读完成: 用户={}", user_id);
        Ok(())
    }
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()> {
        debug!("标记所有通知已读: 用户={}", user_id);
        
        self.notification_repository.mark_all_notifications_read(user_id).await?;
        
        info!("所有通知已标记为已读: 用户={}", user_id);
        Ok(())
    }
    
    /// 获取未读通知数量
    async fn get_unread_notification_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取未读通知数量: 用户={}", user_id);
        
        let count = self.notification_repository.get_unread_count(user_id).await?;
        
        Ok(count)
    }
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<NotificationSettings> {
        debug!("获取通知设置: 用户={}", user_id);
        
        // 验证用户存在
        let _user = self.user_repository.find_user_profile(user_id).await?
            .ok_or_else(|| DomainError::NotFound { resource: "User".to_string() })?;
        
        let settings = match self.notification_repository.get_notification_settings(user_id).await? {
            Some(settings) => settings,
            None => {
                // 如果没有设置，创建默认设置
                let mut default_settings = NotificationSettings::default();
                default_settings.user_id = user_id;
                self.notification_repository.update_notification_settings(default_settings.clone()).await?;
                default_settings
            }
        };
        
        Ok(settings)
    }
    
    /// 更新通知设置
    async fn update_notification_settings(&self, user_id: i64, mut settings: NotificationSettings) -> DomainResult<NotificationSettings> {
        debug!("更新通知设置: 用户={}", user_id);
        
        // 验证用户存在
        let _user = self.user_repository.find_user_profile(user_id).await?
            .ok_or_else(|| DomainError::NotFound { resource: "User".to_string() })?;
        
        // 确保设置属于当前用户
        settings.user_id = user_id;
        settings.updated_at = chrono::Utc::now();
        
        let updated_settings = self.notification_repository.update_notification_settings(settings).await?;
        
        info!("通知设置更新成功: 用户={}", user_id);
        Ok(updated_settings)
    }
    
    /// 删除通知
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()> {
        debug!("删除通知: ID={}, 用户={}", notification_id, user_id);
        
        // 验证通知存在且属于当前用户
        let notification = self.notification_repository.find_by_id(notification_id.clone()).await?;
        match notification {
            Some(notif) if notif.user_id == user_id => {
                self.notification_repository.delete_notification(notification_id, user_id).await?;
                info!("通知已删除: ID={}", notif.notification_id);
                Ok(())
            }
            Some(_) => Err(DomainError::ValidationError {
                field: "notification_id".to_string(),
                reason: "无权限删除此通知".to_string(),
            }),
            None => Err(DomainError::NotFound {
                resource: "通知".to_string(),
            }),
        }
    }
    
    /// 清理过期通知
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64> {
        debug!("清理过期通知");
        
        let cleaned_count = self.notification_repository.cleanup_expired_notifications().await?;
        
        info!("过期通知清理完成: 清理数量={}", cleaned_count);
        Ok(cleaned_count)
    }
}

impl Clone for NotificationApplicationServiceImpl {
    fn clone(&self) -> Self {
        Self {
            notification_repository: Arc::clone(&self.notification_repository),
            user_repository: Arc::clone(&self.user_repository),
            event_publisher: Arc::clone(&self.event_publisher),
        }
    }
}