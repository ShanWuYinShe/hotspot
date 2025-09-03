/// 通知应用服务
/// 
/// 处理通知相关的业务逻辑，协调通知传输和策略

use std::sync::Arc;
use crate::domain::DomainResult;
use crate::ports::{
    NotificationTransportPort, NotificationPolicyPort,
    Notification, NotificationPolicy, NotificationContext, NotificationType
};

/// 通知应用服务
pub struct NotificationApplicationService {
    transport: Arc<dyn NotificationTransportPort>,
    policy: Arc<dyn NotificationPolicyPort>,
}

impl NotificationApplicationService {
    /// 创建新的通知应用服务
    pub fn new(
        transport: Arc<dyn NotificationTransportPort>,
        policy: Arc<dyn NotificationPolicyPort>,
    ) -> Self {
        Self { transport, policy }
    }

    /// 发送通知给特定用户
    pub async fn send_to_user(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
    ) -> DomainResult<()> {
        let notification = Notification::new(user_id, notification_type, title, content);
        self.transport.send_notification(notification).await
    }

    /// 根据策略发送通知
    pub async fn send_with_policy(
        &self,
        policy: NotificationPolicy,
        context: NotificationContext,
        notification_type: NotificationType,
        title: String,
        content: String,
    ) -> DomainResult<()> {
        // 获取目标用户列表
        let target_users = self.policy.get_target_users(policy, context.clone()).await?;
        
        // 为每个用户创建通知
        let mut notifications = Vec::new();
        for user_id in target_users {
            // 检查是否应该通知该用户
            if self.policy.should_notify_user(user_id, notification_type, &context).await? {
                let notification = Notification::new(user_id, notification_type, title.clone(), content.clone());
                notifications.push(notification);
            }
        }

        // 批量发送通知
        if !notifications.is_empty() {
            self.transport.send_notifications(notifications).await?;
        }

        Ok(())
    }

    /// 发送系统通知
    pub async fn send_system_notification(
        &self,
        user_id: i64,
        title: String,
        content: String,
    ) -> DomainResult<()> {
        self.send_to_user(user_id, NotificationType::System, title, content).await
    }

    /// 发送消息通知
    pub async fn send_message_notification(
        &self,
        recipient_id: i64,
        sender_id: i64,
        message_content: String,
    ) -> DomainResult<()> {
        let title = "新消息".to_string();
        let content = format!("您收到一条新消息: {}", message_content);
        
        let mut notification = Notification::new(recipient_id, NotificationType::Message, title, content);
        notification = notification.with_data("sender_id".to_string(), serde_json::Value::Number(sender_id.into()));
        
        self.transport.send_notification(notification).await
    }

    /// 发送群组消息通知
    pub async fn send_group_message_notification(
        &self,
        group_id: i64,
        sender_id: i64,
        message_content: String,
    ) -> DomainResult<()> {
        let policy = NotificationPolicy::GroupMembers { group_id };
        let context = NotificationContext {
            trigger_user_id: sender_id,
            group_id: Some(group_id),
            message_id: None,
            moment_id: None,
            location: None,
            extra_data: std::collections::HashMap::new(),
        };

        let title = "群组新消息".to_string();
        let content = format!("群组中有新消息: {}", message_content);

        self.send_with_policy(policy, context, NotificationType::GroupMessage, title, content).await
    }

    /// 获取传输统计信息
    pub async fn get_transport_stats(&self) -> DomainResult<crate::ports::TransportStats> {
        self.transport.get_transport_stats().await
    }

    /// 获取策略统计信息
    pub async fn get_policy_stats(&self) -> DomainResult<crate::ports::PolicyStats> {
        self.policy.get_policy_stats().await
    }
}

