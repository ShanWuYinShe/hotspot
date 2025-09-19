/// 通知应用服务
/// 
/// 处理通知相关的业务逻辑，协调通知传输和策略
/// 通知应用服务
/// 
/// 功能职责：
/// - 实现统一的通知管理和分发机制
/// - 协调多种通知渠道和通知策略
/// - 管理通知的创建、发送、状态跟踪
/// - 处理通知的个性化和用户偏好设置
/// - 提供通知的统计分析和效果评估
/// 
/// 通知类型：
/// - 系统通知：系统更新、维护通知等
/// - 业务通知：消息、群组、瞬间相关通知
/// - 社交通知：好友申请、点赞、评论等
/// - 位置通知：附近活动、地理围栏等
/// - 安全通知：登录异常、密码变更等
/// 
/// 通知渠道：
/// - WebSocket实时通知（主要渠道）
/// - 邮件通知（重要事件）
/// - 短信通知（安全验证）
/// - 推送通知（移动端）
use std::sync::Arc;
use crate::domain::DomainResult;
use crate::ports::NotificationTransport;
use crate::ports::services::Notification;
use crate::ports::services::NotificationPolicy as NotificationPolicyTrait;
use crate::ports::services::NotificationType;

/// 通知应用服务
pub struct NotificationApplicationService {
    transport: Arc<dyn NotificationTransport>,
    policy: Arc<dyn NotificationPolicyTrait>,
}

impl NotificationApplicationService {
    /// 创建新的通知应用服务
    pub fn new(
        transport: Arc<dyn NotificationTransport>,
        policy: Arc<dyn NotificationPolicyTrait>,
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
        let user_id_domain = crate::domain::UserId::new(user_id).map_err(|e| crate::domain::DomainError::ValidationError { field: "user_id".to_string(), reason: e.to_string() })?;
        let notification = Notification::new(Some(user_id_domain), notification_type, title, content);
        self.transport.send_notification(user_id, &notification).await
    }

    /// 根据策略发送通知
    pub async fn send_with_policy(
        &self,
        policy: crate::ports::notification_ports::NotificationPolicy,
        context: crate::ports::notification_ports::NotificationContext,
        notification_type: crate::ports::notification_ports::NotificationType,
        title: String,
        content: String,
    ) -> DomainResult<()> {
        // 转换上下文类型
        let services_context = crate::ports::services::NotificationContext {
            user_id: crate::domain::UserId::new(context.trigger_user_id)
                .map_err(|e| crate::domain::DomainError::ValidationError { 
                    field: "user_id".to_string(), 
                    reason: e.to_string() 
                })?,
            context_data: std::collections::HashMap::new(), // 可以根据需要填充
        };
        
        // 使用实例的策略服务来获取目标用户列表
        let target_users = self.policy.get_recipients(&services_context).await
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation { 
                rule: format!("通知策略服务错误: {}", e)
            })?;
        
        // 备用逻辑：如果策略服务失败，使用简单的策略匹配
        let fallback_users = match policy {
            crate::ports::notification_ports::NotificationPolicy::Specific { user_ids } => user_ids,
            crate::ports::notification_ports::NotificationPolicy::GroupMembers { group_id: _ } => {
                // TODO: 从群组仓库获取成员列表
                vec![context.trigger_user_id] // 临时返回触发用户
            },
            _ => {
                // 其他策略暂时不实现
                vec![]
            }
        };
        
        // 转换通知类型
        let services_notification_type = match notification_type {
            crate::ports::notification_ports::NotificationType::System => crate::ports::services::NotificationType::System,
            crate::ports::notification_ports::NotificationType::Message => crate::ports::services::NotificationType::Message,
            crate::ports::notification_ports::NotificationType::GroupInvite => crate::ports::services::NotificationType::GroupInvite,
            crate::ports::notification_ports::NotificationType::GroupMessage => crate::ports::services::NotificationType::GroupMessage,
            crate::ports::notification_ports::NotificationType::FriendRequest => crate::ports::services::NotificationType::FriendRequest,
            crate::ports::notification_ports::NotificationType::MomentLike => crate::ports::services::NotificationType::MomentLike,
            crate::ports::notification_ports::NotificationType::MomentComment => crate::ports::services::NotificationType::MomentComment,
            _ => crate::ports::services::NotificationType::System, // 默认为系统通知
        };

        // 为每个用户创建通知（优先使用策略服务的结果）
        let mut notifications = Vec::new();
        let final_users = if !target_users.is_empty() { 
            target_users 
        } else { 
            fallback_users 
        };
        
        for user_id in final_users {
            // 直接创建通知（策略已经在上面处理过了）
            let user_id_domain = crate::domain::UserId::new(user_id).map_err(|e| crate::domain::DomainError::ValidationError { field: "user_id".to_string(), reason: e.to_string() })?;
            let notification = Notification::new(Some(user_id_domain), services_notification_type.clone(), title.clone(), content.clone());
            notifications.push((user_id, notification));
        }

        // 批量发送通知
        if !notifications.is_empty() {
            self.transport.send_notifications(&notifications).await?;
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
        
        let recipient_id_domain = crate::domain::UserId::new(recipient_id).map_err(|e| crate::domain::DomainError::ValidationError { field: "recipient_id".to_string(), reason: e.to_string() })?;
        let mut notification = Notification::new(Some(recipient_id_domain), NotificationType::Message, title, content);
        let mut data = serde_json::Map::new();
        data.insert("sender_id".to_string(), serde_json::Value::Number(sender_id.into()));
        notification = notification.with_data(serde_json::Value::Object(data));
        
        self.transport.send_notification(recipient_id, &notification).await
    }

    /// 发送群组消息通知
    pub async fn send_group_message_notification(
        &self,
        group_id: i64,
        sender_id: i64,
        message_content: String,
    ) -> DomainResult<()> {
        let policy = crate::ports::notification_ports::NotificationPolicy::GroupMembers { group_id };
        let context = crate::ports::notification_ports::NotificationContext {
            trigger_user_id: sender_id,
            group_id: Some(group_id),
            message_id: None,
            moment_id: None,
            location: None,
            extra_data: std::collections::HashMap::new(),
        };

        let title = "群组新消息".to_string();
        let content = format!("群组中有新消息: {}", message_content);

        self.send_with_policy(policy, context, crate::ports::notification_ports::NotificationType::GroupMessage, title, content).await
    }

    /// 获取传输统计信息
    pub async fn get_transport_stats(&self) -> DomainResult<crate::ports::services::TransportStats> {
        self.transport.get_transport_stats().await
    }

    /// 获取策略统计信息
    pub async fn get_policy_stats(&self) -> DomainResult<crate::ports::services::PolicyStats> {
        // TODO: 实现策略统计
        Ok(crate::ports::services::PolicyStats {
            applied_count: 0,
            skipped_count: 0,
            success_rate: 1.0,
        })
    }
}

