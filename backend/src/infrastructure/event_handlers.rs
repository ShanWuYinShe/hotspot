/// 事件处理器实现 - 具体的业务事件处理逻辑
/// 
/// 处理器类型：
/// - 用户事件处理器
/// - 群组事件处理器
/// - 通知事件处理器
/// - 统计事件处理器
/// - 审计事件处理器

use async_trait::async_trait;
use std::sync::Arc;
use crate::domain::{DomainResult, DomainError, UserId, GroupId};
use crate::infrastructure::event_system::{
    GenericEventHandler, EventEnvelope, EventMetadata,
    UserRegisteredEvent, GroupCreatedEvent, UserJoinedGroupEvent,
};

// === 用户事件处理器 ===

/// 用户注册事件处理器 - 处理用户注册后的业务逻辑
pub struct UserRegistrationHandler {
    /// 邮件服务（可选）
    email_service: Option<Arc<dyn EmailService>>,
    /// 欢迎消息服务
    welcome_service: Arc<dyn WelcomeService>,
    /// 用户统计服务
    stats_service: Arc<dyn UserStatsService>,
}

impl UserRegistrationHandler {
    /// 创建新的用户注册处理器
    pub fn new(
        email_service: Option<Arc<dyn EmailService>>,
        welcome_service: Arc<dyn WelcomeService>,
        stats_service: Arc<dyn UserStatsService>,
    ) -> Self {
        Self {
            email_service,
            welcome_service,
            stats_service,
        }
    }
    
    /// 处理用户注册事件
    async fn handle_user_registered(&self, event: &UserRegisteredEvent, metadata: &EventMetadata) -> DomainResult<()> {
        tracing::info!(
            user_id = %event.user_id.value(),
            login_id = %event.login_id,
            event_id = %metadata.event_id.value(),
            "Processing user registration event"
        );
        
        // 发送欢迎邮件（如果配置了邮件服务）
        if let Some(email_service) = &self.email_service {
            if let Some(email) = &event.email {
                match email_service.send_welcome_email(email, &event.nickname).await {
                    Ok(()) => {
                        tracing::info!(
                            user_id = %event.user_id.value(),
                            email = %email,
                            "Welcome email sent successfully"
                        );
                    }
                    Err(e) => {
                        tracing::warn!(
                            user_id = %event.user_id.value(),
                            email = %email,
                            error = %e,
                            "Failed to send welcome email, but continuing processing"
                        );
                        // 不阻断处理流程，邮件发送失败不是致命错误
                    }
                }
            }
        }
        
        // 创建欢迎消息
        self.welcome_service.create_welcome_message(event.user_id).await?;
        
        // 更新用户统计
        self.stats_service.increment_user_count(event.user_type).await?;
        
        // 记录用户注册审计日志
        tracing::info!(
            user_id = %event.user_id.value(),
            login_id = %event.login_id,
            user_type = ?event.user_type,
            registered_at = %event.registered_at,
            "User registration completed successfully"
        );
        
        Ok(())
    }
}

#[async_trait]
impl GenericEventHandler for UserRegistrationHandler {
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()> {
        match envelope.metadata.event_type.as_str() {
            "UserRegistered" => {
                let event: UserRegisteredEvent = serde_json::from_slice(&envelope.data)
                    .map_err(|e| DomainError::ValidationError {
                        field: "event_data".to_string(),
                        reason: format!("Failed to deserialize UserRegisteredEvent: {}", e),
                    })?;
                
                self.handle_user_registered(&event, &envelope.metadata).await
            }
            _ => {
                tracing::warn!(
                    event_type = %envelope.metadata.event_type,
                    handler = self.name(),
                    "Unsupported event type"
                );
                Ok(()) // 忽略不支持的事件类型
            }
        }
    }
    
    fn name(&self) -> &str {
        "UserRegistrationHandler"
    }
    
    fn supported_event_types(&self) -> Vec<&'static str> {
        vec!["UserRegistered"]
    }
}

// === 群组事件处理器 ===

/// 群组创建事件处理器
pub struct GroupCreationHandler {
    /// 通知服务
    notification_service: Arc<dyn NotificationService>,
    /// 群组统计服务
    stats_service: Arc<dyn GroupStatsService>,
    /// 推荐服务
    recommendation_service: Arc<dyn RecommendationService>,
}

impl GroupCreationHandler {
    /// 创建新的群组创建处理器
    pub fn new(
        notification_service: Arc<dyn NotificationService>,
        stats_service: Arc<dyn GroupStatsService>,
        recommendation_service: Arc<dyn RecommendationService>,
    ) -> Self {
        Self {
            notification_service,
            stats_service,
            recommendation_service,
        }
    }
    
    /// 处理群组创建事件
    async fn handle_group_created(&self, event: &GroupCreatedEvent, metadata: &EventMetadata) -> DomainResult<()> {
        tracing::info!(
            group_id = %event.group_id.value(),
            creator_id = %event.creator_id.value(),
            group_name = %event.name,
            event_id = %metadata.event_id.value(),
            "Processing group creation event"
        );
        
        // 通知附近的用户有新群组创建
        self.notification_service
            .notify_nearby_users_about_new_group(event.group_id.clone(), event.creator_id.clone())
            .await?;
        
        // 更新群组统计
        self.stats_service
            .increment_group_count(&event.group_type)
            .await?;
        
        // 为创建者推荐相似的群组
        self.recommendation_service
            .generate_group_recommendations(event.creator_id)
            .await?;
        
        tracing::info!(
            group_id = %event.group_id.value(),
            "Group creation event processed successfully"
        );
        
        Ok(())
    }
}

#[async_trait]
impl GenericEventHandler for GroupCreationHandler {
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()> {
        match envelope.metadata.event_type.as_str() {
            "GroupCreated" => {
                let event: GroupCreatedEvent = serde_json::from_slice(&envelope.data)
                    .map_err(|e| DomainError::ValidationError {
                        field: "event_data".to_string(),
                        reason: format!("Failed to deserialize GroupCreatedEvent: {}", e),
                    })?;
                
                self.handle_group_created(&event, &envelope.metadata).await
            }
            _ => Ok(()) // 忽略不支持的事件类型
        }
    }
    
    fn name(&self) -> &str {
        "GroupCreationHandler"
    }
    
    fn supported_event_types(&self) -> Vec<&'static str> {
        vec!["GroupCreated"]
    }
}

// === 群组成员事件处理器 ===

/// 群组成员事件处理器
pub struct GroupMembershipHandler {
    /// WebSocket通知服务
    websocket_service: Arc<dyn WebSocketService>,
    /// 群组统计服务
    stats_service: Arc<dyn GroupStatsService>,
    /// 活动记录服务
    activity_service: Arc<dyn ActivityService>,
}

impl GroupMembershipHandler {
    /// 创建新的群组成员处理器
    pub fn new(
        websocket_service: Arc<dyn WebSocketService>,
        stats_service: Arc<dyn GroupStatsService>,
        activity_service: Arc<dyn ActivityService>,
    ) -> Self {
        Self {
            websocket_service,
            stats_service,
            activity_service,
        }
    }
    
    /// 处理用户加入群组事件
    async fn handle_user_joined_group(&self, event: &UserJoinedGroupEvent, metadata: &EventMetadata) -> DomainResult<()> {
        tracing::info!(
            user_id = %event.user_id.value(),
            group_id = %event.group_id.value(),
            role = %event.role,
            event_id = %metadata.event_id.value(),
            "Processing user joined group event"
        );
        
        // 实时通知群组其他成员
        self.websocket_service
            .notify_group_members(
                event.group_id.clone(),
                &format!("用户加入了群组"),
                Some(event.user_id),
            )
            .await?;
        
        // 更新群组成员统计
        self.stats_service
            .increment_member_count(event.group_id.clone())
            .await?;
        
        // 记录活动
        self.activity_service
            .record_group_activity(
                event.group_id.clone(),
                event.user_id,
                "joined_group",
                event.joined_at,
            )
            .await?;
        
        tracing::info!(
            user_id = %event.user_id.value(),
            group_id = %event.group_id.value(),
            "User joined group event processed successfully"
        );
        
        Ok(())
    }
}

#[async_trait]
impl GenericEventHandler for GroupMembershipHandler {
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()> {
        match envelope.metadata.event_type.as_str() {
            "UserJoinedGroup" => {
                let event: UserJoinedGroupEvent = serde_json::from_slice(&envelope.data)
                    .map_err(|e| DomainError::ValidationError {
                        field: "event_data".to_string(),
                        reason: format!("Failed to deserialize UserJoinedGroupEvent: {}", e),
                    })?;
                
                self.handle_user_joined_group(&event, &envelope.metadata).await
            }
            _ => Ok(()) // 忽略不支持的事件类型
        }
    }
    
    fn name(&self) -> &str {
        "GroupMembershipHandler"
    }
    
    fn supported_event_types(&self) -> Vec<&'static str> {
        vec!["UserJoinedGroup", "UserLeftGroup"]
    }
}

// === 审计事件处理器 ===

/// 审计事件处理器 - 记录所有事件用于审计
pub struct AuditEventHandler {
    /// 审计日志服务
    audit_service: Arc<dyn AuditService>,
}

impl AuditEventHandler {
    /// 创建新的审计事件处理器
    pub fn new(audit_service: Arc<dyn AuditService>) -> Self {
        Self { audit_service }
    }
}

#[async_trait]
impl GenericEventHandler for AuditEventHandler {
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()> {
        // 记录所有事件到审计日志
        self.audit_service
            .log_event(
                &envelope.metadata.event_type,
                &envelope.metadata.aggregate_id,
                envelope.metadata.user_id,
                &envelope.metadata.occurred_at,
                &envelope.metadata.metadata,
            )
            .await?;
        
        tracing::debug!(
            event_type = %envelope.metadata.event_type,
            event_id = %envelope.metadata.event_id.value(),
            aggregate_id = %envelope.metadata.aggregate_id,
            "Event logged to audit trail"
        );
        
        Ok(())
    }
    
    fn name(&self) -> &str {
        "AuditEventHandler"
    }
    
    fn supported_event_types(&self) -> Vec<&'static str> {
        // 审计处理器处理所有事件类型
        vec!["*"]
    }
}

// === 统计事件处理器 ===

/// 统计事件处理器 - 更新各种业务统计
pub struct StatisticsEventHandler {
    /// 统计服务
    statistics_service: Arc<dyn StatisticsService>,
}

impl StatisticsEventHandler {
    /// 创建新的统计事件处理器
    pub fn new(statistics_service: Arc<dyn StatisticsService>) -> Self {
        Self { statistics_service }
    }
}

#[async_trait]
impl GenericEventHandler for StatisticsEventHandler {
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()> {
        // 根据事件类型更新不同的统计指标
        match envelope.metadata.event_type.as_str() {
            "UserRegistered" => {
                self.statistics_service
                    .increment_counter("users.registered.total")
                    .await?;
                
                self.statistics_service
                    .increment_counter(&format!(
                        "users.registered.by_date.{}",
                        envelope.metadata.occurred_at.format("%Y-%m-%d")
                    ))
                    .await?;
            }
            "GroupCreated" => {
                self.statistics_service
                    .increment_counter("groups.created.total")
                    .await?;
                
                self.statistics_service
                    .increment_counter(&format!(
                        "groups.created.by_date.{}",
                        envelope.metadata.occurred_at.format("%Y-%m-%d")
                    ))
                    .await?;
            }
            "UserJoinedGroup" => {
                self.statistics_service
                    .increment_counter("group_memberships.created.total")
                    .await?;
            }
            _ => {
                // 对于其他事件类型，记录通用统计
                self.statistics_service
                    .increment_counter(&format!("events.{}.total", envelope.metadata.event_type))
                    .await?;
            }
        }
        
        // 记录事件处理延迟
        let processing_delay = chrono::Utc::now()
            .signed_duration_since(envelope.metadata.occurred_at)
            .num_milliseconds();
        
        self.statistics_service
            .record_histogram(
                "events.processing_delay_ms",
                processing_delay as f64,
            )
            .await?;
        
        Ok(())
    }
    
    fn name(&self) -> &str {
        "StatisticsEventHandler"
    }
    
    fn supported_event_types(&self) -> Vec<&'static str> {
        vec!["*"] // 处理所有事件类型
    }
}

// === 服务特征定义 ===

/// 邮件服务特征
#[async_trait]
pub trait EmailService: Send + Sync {
    async fn send_welcome_email(&self, email: &str, nickname: &str) -> DomainResult<()>;
}

/// 欢迎消息服务特征
#[async_trait]
pub trait WelcomeService: Send + Sync {
    async fn create_welcome_message(&self, user_id: UserId) -> DomainResult<()>;
}

/// 用户统计服务特征
#[async_trait]
pub trait UserStatsService: Send + Sync {
    async fn increment_user_count(&self, user_type: crate::domain::entities::UserType) -> DomainResult<()>;
}

/// 通知服务特征
#[async_trait]
pub trait NotificationService: Send + Sync {
    async fn notify_nearby_users_about_new_group(&self, group_id: GroupId, creator_id: UserId) -> DomainResult<()>;
}

/// 群组统计服务特征
#[async_trait]
pub trait GroupStatsService: Send + Sync {
    async fn increment_group_count(&self, group_type: &str) -> DomainResult<()>;
    async fn increment_member_count(&self, group_id: GroupId) -> DomainResult<()>;
}

/// 推荐服务特征
#[async_trait]
pub trait RecommendationService: Send + Sync {
    async fn generate_group_recommendations(&self, user_id: UserId) -> DomainResult<()>;
}

/// WebSocket服务特征
#[async_trait]
pub trait WebSocketService: Send + Sync {
    async fn notify_group_members(&self, group_id: GroupId, message: &str, exclude_user: Option<UserId>) -> DomainResult<()>;
}

/// 活动记录服务特征
#[async_trait]
pub trait ActivityService: Send + Sync {
    async fn record_group_activity(
        &self,
        group_id: GroupId,
        user_id: UserId,
        activity_type: &str,
        occurred_at: chrono::DateTime<chrono::Utc>,
    ) -> DomainResult<()>;
}

/// 审计服务特征
#[async_trait]
pub trait AuditService: Send + Sync {
    async fn log_event(
        &self,
        event_type: &str,
        aggregate_id: &str,
        user_id: Option<UserId>,
        occurred_at: &chrono::DateTime<chrono::Utc>,
        metadata: &std::collections::HashMap<String, String>,
    ) -> DomainResult<()>;
}

/// 统计服务特征
#[async_trait]
pub trait StatisticsService: Send + Sync {
    async fn increment_counter(&self, metric_name: &str) -> DomainResult<()>;
    async fn record_histogram(&self, metric_name: &str, value: f64) -> DomainResult<()>;
}

// === 事件处理器工厂 ===

/// 事件处理器工厂 - 创建和配置事件处理器
pub struct EventHandlerFactory;

impl EventHandlerFactory {
    /// 创建用户注册处理器
    pub fn create_user_registration_handler(
        email_service: Option<Arc<dyn EmailService>>,
        welcome_service: Arc<dyn WelcomeService>,
        stats_service: Arc<dyn UserStatsService>,
    ) -> Arc<dyn GenericEventHandler> {
        Arc::new(UserRegistrationHandler::new(
            email_service,
            welcome_service,
            stats_service,
        ))
    }
    
    /// 创建群组创建处理器
    pub fn create_group_creation_handler(
        notification_service: Arc<dyn NotificationService>,
        stats_service: Arc<dyn GroupStatsService>,
        recommendation_service: Arc<dyn RecommendationService>,
    ) -> Arc<dyn GenericEventHandler> {
        Arc::new(GroupCreationHandler::new(
            notification_service,
            stats_service,
            recommendation_service,
        ))
    }
    
    /// 创建群组成员处理器
    pub fn create_group_membership_handler(
        websocket_service: Arc<dyn WebSocketService>,
        stats_service: Arc<dyn GroupStatsService>,
        activity_service: Arc<dyn ActivityService>,
    ) -> Arc<dyn GenericEventHandler> {
        Arc::new(GroupMembershipHandler::new(
            websocket_service,
            stats_service,
            activity_service,
        ))
    }
    
    /// 创建审计处理器
    pub fn create_audit_handler(
        audit_service: Arc<dyn AuditService>,
    ) -> Arc<dyn GenericEventHandler> {
        Arc::new(AuditEventHandler::new(audit_service))
    }
    
    /// 创建统计处理器
    pub fn create_statistics_handler(
        statistics_service: Arc<dyn StatisticsService>,
    ) -> Arc<dyn GenericEventHandler> {
        Arc::new(StatisticsEventHandler::new(statistics_service))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Mock服务实现用于测试
    struct MockEmailService;
    
    #[async_trait]
    impl EmailService for MockEmailService {
        async fn send_welcome_email(&self, _email: &str, _nickname: &str) -> DomainResult<()> {
            Ok(())
        }
    }
    
    struct MockWelcomeService;
    
    #[async_trait]
    impl WelcomeService for MockWelcomeService {
        async fn create_welcome_message(&self, _user_id: UserId) -> DomainResult<()> {
            Ok(())
        }
    }
    
    struct MockUserStatsService;
    
    #[async_trait]
    impl UserStatsService for MockUserStatsService {
        async fn increment_user_count(&self, _user_type: crate::domain::entities::UserType) -> DomainResult<()> {
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_user_registration_handler() {
        let email_service = Arc::new(MockEmailService);
        let welcome_service = Arc::new(MockWelcomeService);
        let stats_service = Arc::new(MockUserStatsService);
        
        let handler = UserRegistrationHandler::new(
            Some(email_service),
            welcome_service,
            stats_service,
        );
        
        assert_eq!(handler.name(), "UserRegistrationHandler");
        assert_eq!(handler.supported_event_types(), vec!["UserRegistered"]);
    }
}
