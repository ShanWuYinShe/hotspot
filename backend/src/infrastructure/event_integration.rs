/// 事件驱动架构集成 - 完整的事件系统集成示例
/// 
/// 集成特性：
/// - 事件发布器配置
/// - 事件处理器注册
/// - 事件存储初始化
/// - 领域服务集成
/// - 应用服务集成

use std::sync::Arc;
use sqlx::PgPool;
use crate::domain::{DomainResult, UserId, GroupId};
use crate::infrastructure::{
    EventPublisher, EventStore, PostgresEventStore, InMemoryEventStore,
    EventHandlerFactory, GenericEventHandler,
    event_handlers::WebSocketService,
    UserRegisteredEvent, GroupCreatedEvent, UserJoinedGroupEvent,
    EmailService, WelcomeService, UserStatsService, NotificationService,
    GroupStatsService, RecommendationService, ActivityService,
    AuditService, StatisticsService,
};

/// 事件系统配置
#[derive(Debug, Clone)]
pub struct EventSystemConfig {
    /// 是否启用事件系统
    pub enabled: bool,
    /// 使用PostgreSQL存储（false则使用内存存储）
    pub use_postgres_store: bool,
    /// 是否启用审计处理器
    pub enable_audit_handler: bool,
    /// 是否启用统计处理器
    pub enable_statistics_handler: bool,
    /// 是否启用邮件服务
    pub enable_email_service: bool,
    /// 批量处理大小
    pub batch_size: usize,
    /// 处理器重试次数
    pub max_retries: u32,
}

impl Default for EventSystemConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            use_postgres_store: true,
            enable_audit_handler: true,
            enable_statistics_handler: true,
            enable_email_service: false, // 默认关闭邮件服务
            batch_size: 100,
            max_retries: 3,
        }
    }
}

/// 事件系统集成器 - 负责整个事件系统的初始化和配置
pub struct EventSystemIntegrator {
    config: EventSystemConfig,
    event_publisher: Option<Arc<EventPublisher>>,
    event_store: Option<Arc<dyn EventStore>>,
}

impl EventSystemIntegrator {
    /// 创建新的事件系统集成器
    pub fn new(config: EventSystemConfig) -> Self {
        Self {
            config,
            event_publisher: None,
            event_store: None,
        }
    }
    
    /// 初始化事件系统
    pub async fn initialize(&mut self, pool: Option<PgPool>) -> DomainResult<()> {
        if !self.config.enabled {
            tracing::info!("Event system is disabled");
            return Ok(());
        }
        
        // 初始化事件存储
        let event_store: Arc<dyn EventStore> = if self.config.use_postgres_store {
            if let Some(pool) = pool {
                let postgres_store = PostgresEventStore::new(pool);
                postgres_store.initialize().await?;
                Arc::new(postgres_store)
            } else {
                tracing::warn!("PostgreSQL pool not provided, falling back to in-memory store");
                Arc::new(InMemoryEventStore::new())
            }
        } else {
            Arc::new(InMemoryEventStore::new())
        };
        
        // 创建事件发布器
        let event_publisher = Arc::new(EventPublisher::new(event_store.clone()));
        
        self.event_store = Some(event_store);
        self.event_publisher = Some(event_publisher);
        
        tracing::info!(
            postgres_store = self.config.use_postgres_store,
            audit_enabled = self.config.enable_audit_handler,
            statistics_enabled = self.config.enable_statistics_handler,
            "Event system initialized successfully"
        );
        
        Ok(())
    }
    
    /// 注册所有事件处理器
    pub async fn register_handlers(&self, services: EventServiceRegistry) -> DomainResult<()> {
        let publisher = self.event_publisher.as_ref()
            .ok_or_else(|| crate::domain::DomainError::BusinessRuleViolation {
                rule: "Event system not initialized".to_string(),
            })?;
        
        // 注册用户注册处理器
        if let (Some(welcome_service), Some(stats_service)) = 
            (services.welcome_service, services.user_stats_service) {
            
            let email_service = if self.config.enable_email_service {
                services.email_service
            } else {
                None
            };
            
            let handler = EventHandlerFactory::create_user_registration_handler(
                email_service,
                welcome_service,
                stats_service,
            );
            
            publisher.register_handler(handler).await;
        }
        
        // 注册群组创建处理器
        if let (Some(notification_service), Some(group_stats_service), Some(recommendation_service)) = 
            (services.notification_service, services.group_stats_service, services.recommendation_service) {
            
            let handler = EventHandlerFactory::create_group_creation_handler(
                notification_service,
                group_stats_service,
                recommendation_service,
            );
            
            publisher.register_handler(handler).await;
        }
        
        // 注册群组成员处理器
        if let (Some(websocket_service), Some(group_stats_service), Some(activity_service)) = 
            (services.websocket_service, services.group_stats_service.clone(), services.activity_service) {
            
            let handler = EventHandlerFactory::create_group_membership_handler(
                websocket_service,
                group_stats_service,
                activity_service,
            );
            
            publisher.register_handler(handler).await;
        }
        
        // 注册审计处理器
        if self.config.enable_audit_handler {
            if let Some(audit_service) = services.audit_service {
                let handler = EventHandlerFactory::create_audit_handler(audit_service);
                publisher.register_handler(handler).await;
            }
        }
        
        // 注册统计处理器
        if self.config.enable_statistics_handler {
            if let Some(statistics_service) = services.statistics_service {
                let handler = EventHandlerFactory::create_statistics_handler(statistics_service);
                publisher.register_handler(handler).await;
            }
        }
        
        tracing::info!("Event handlers registered successfully");
        Ok(())
    }
    
    /// 获取事件发布器
    pub fn get_publisher(&self) -> Option<Arc<EventPublisher>> {
        self.event_publisher.clone()
    }
    
    /// 获取事件存储
    pub fn get_store(&self) -> Option<Arc<dyn EventStore>> {
        self.event_store.clone()
    }
    
    /// 发布用户注册事件
    pub async fn publish_user_registered(
        &self,
        user_id: UserId,
        login_id: String,
        nickname: String,
        email: Option<String>,
        user_type: crate::domain::UserType,
    ) -> DomainResult<()> {
        if let Some(publisher) = &self.event_publisher {
            let event = UserRegisteredEvent {
                user_id,
                login_id,
                nickname,
                email,
                user_type,
                registered_at: chrono::Utc::now(),
            };
            
            publisher.publish(Box::new(event)).await?;
            
            tracing::info!(
                user_id = %user_id.value(),
                "User registered event published"
            );
        }
        
        Ok(())
    }
    
    /// 发布群组创建事件
    pub async fn publish_group_created(
        &self,
        group_id: GroupId,
        name: String,
        creator_id: UserId,
        group_type: String,
    ) -> DomainResult<()> {
        if let Some(publisher) = &self.event_publisher {
            let event = GroupCreatedEvent {
                group_id,
                name,
                creator_id,
                group_type,
                created_at: chrono::Utc::now(),
            };
            
            publisher.publish(Box::new(event)).await?;
            
            tracing::info!(
                group_id = %group_id.value(),
                creator_id = %creator_id.value(),
                "Group created event published"
            );
        }
        
        Ok(())
    }
    
    /// 发布用户加入群组事件
    pub async fn publish_user_joined_group(
        &self,
        user_id: UserId,
        group_id: GroupId,
        role: String,
    ) -> DomainResult<()> {
        if let Some(publisher) = &self.event_publisher {
            let event = UserJoinedGroupEvent {
                user_id,
                group_id,
                role,
                joined_at: chrono::Utc::now(),
            };
            
            publisher.publish(Box::new(event)).await?;
            
            tracing::info!(
                user_id = %user_id.value(),
                group_id = %group_id.value(),
                "User joined group event published"
            );
        }
        
        Ok(())
    }
    
    /// 获取事件处理统计
    pub async fn get_processing_stats(&self) -> Option<crate::infrastructure::event_system::EventStats> {
        if let Some(publisher) = &self.event_publisher {
            Some(publisher.get_stats().await)
        } else {
            None
        }
    }
}

/// 事件服务注册表 - 包含所有需要的服务实现
#[derive(Default)]
pub struct EventServiceRegistry {
    pub email_service: Option<Arc<dyn EmailService>>,
    pub welcome_service: Option<Arc<dyn WelcomeService>>,
    pub user_stats_service: Option<Arc<dyn UserStatsService>>,
    pub notification_service: Option<Arc<dyn NotificationService>>,
    pub group_stats_service: Option<Arc<dyn GroupStatsService>>,
    pub recommendation_service: Option<Arc<dyn RecommendationService>>,
    pub websocket_service: Option<Arc<dyn WebSocketService>>,
    pub activity_service: Option<Arc<dyn ActivityService>>,
    pub audit_service: Option<Arc<dyn AuditService>>,
    pub statistics_service: Option<Arc<dyn StatisticsService>>,
}

impl EventServiceRegistry {
    /// 创建新的服务注册表
    pub fn new() -> Self {
        Self::default()
    }
    
    /// 设置邮件服务
    pub fn with_email_service(mut self, service: Arc<dyn EmailService>) -> Self {
        self.email_service = Some(service);
        self
    }
    
    /// 设置欢迎服务
    pub fn with_welcome_service(mut self, service: Arc<dyn WelcomeService>) -> Self {
        self.welcome_service = Some(service);
        self
    }
    
    /// 设置用户统计服务
    pub fn with_user_stats_service(mut self, service: Arc<dyn UserStatsService>) -> Self {
        self.user_stats_service = Some(service);
        self
    }
    
    /// 设置通知服务
    pub fn with_notification_service(mut self, service: Arc<dyn NotificationService>) -> Self {
        self.notification_service = Some(service);
        self
    }
    
    /// 设置群组统计服务
    pub fn with_group_stats_service(mut self, service: Arc<dyn GroupStatsService>) -> Self {
        self.group_stats_service = Some(service);
        self
    }
    
    /// 设置推荐服务
    pub fn with_recommendation_service(mut self, service: Arc<dyn RecommendationService>) -> Self {
        self.recommendation_service = Some(service);
        self
    }
    
    /// 设置WebSocket服务
    pub fn with_websocket_service(mut self, service: Arc<dyn WebSocketService>) -> Self {
        self.websocket_service = Some(service);
        self
    }
    
    /// 设置活动服务
    pub fn with_activity_service(mut self, service: Arc<dyn ActivityService>) -> Self {
        self.activity_service = Some(service);
        self
    }
    
    /// 设置审计服务
    pub fn with_audit_service(mut self, service: Arc<dyn AuditService>) -> Self {
        self.audit_service = Some(service);
        self
    }
    
    /// 设置统计服务
    pub fn with_statistics_service(mut self, service: Arc<dyn StatisticsService>) -> Self {
        self.statistics_service = Some(service);
        self
    }
}

/// 事件驱动的应用服务基类 - 集成事件发布功能
pub struct EventDrivenApplicationService {
    event_integrator: Arc<EventSystemIntegrator>,
}

impl EventDrivenApplicationService {
    /// 创建新的事件驱动应用服务
    pub fn new(event_integrator: Arc<EventSystemIntegrator>) -> Self {
        Self { event_integrator }
    }
    
    /// 处理用户注册业务逻辑并发布事件
    pub async fn handle_user_registration(
        &self,
        user_id: UserId,
        login_id: String,
        nickname: String,
        email: Option<String>,
        user_type: crate::domain::UserType,
    ) -> DomainResult<()> {
        // 这里可以添加业务逻辑验证
        
        // 发布用户注册事件
        self.event_integrator
            .publish_user_registered(user_id, login_id, nickname, email, user_type)
            .await?;
        
        Ok(())
    }
    
    /// 处理群组创建业务逻辑并发布事件
    pub async fn handle_group_creation(
        &self,
        group_id: GroupId,
        name: String,
        creator_id: UserId,
        group_type: String,
    ) -> DomainResult<()> {
        // 这里可以添加业务逻辑验证
        
        // 发布群组创建事件
        self.event_integrator
            .publish_group_created(group_id, name, creator_id, group_type)
            .await?;
        
        Ok(())
    }
    
    /// 处理用户加入群组业务逻辑并发布事件
    pub async fn handle_user_join_group(
        &self,
        user_id: UserId,
        group_id: GroupId,
        role: String,
    ) -> DomainResult<()> {
        // 这里可以添加业务逻辑验证
        
        // 发布用户加入群组事件
        self.event_integrator
            .publish_user_joined_group(user_id, group_id, role)
            .await?;
        
        Ok(())
    }
}

/// 事件系统健康检查
pub struct EventSystemHealthChecker {
    event_integrator: Arc<EventSystemIntegrator>,
}

impl EventSystemHealthChecker {
    /// 创建新的事件系统健康检查器
    pub fn new(event_integrator: Arc<EventSystemIntegrator>) -> Self {
        Self { event_integrator }
    }
    
    /// 检查事件系统健康状态
    pub async fn check_health(&self) -> EventSystemHealth {
        let mut health = EventSystemHealth {
            is_healthy: true,
            publisher_available: false,
            store_available: false,
            processing_stats: None,
            error_message: None,
        };
        
        // 检查发布器是否可用
        if let Some(_publisher) = self.event_integrator.get_publisher() {
            health.publisher_available = true;
            
            // 获取处理统计
            health.processing_stats = self.event_integrator.get_processing_stats().await;
        } else {
            health.is_healthy = false;
            health.error_message = Some("Event publisher not available".to_string());
        }
        
        // 检查存储是否可用
        if let Some(_store) = self.event_integrator.get_store() {
            health.store_available = true;
        } else {
            health.is_healthy = false;
            health.error_message = Some("Event store not available".to_string());
        }
        
        health
    }
}

/// 事件系统健康状态
#[derive(Debug, Clone)]
pub struct EventSystemHealth {
    pub is_healthy: bool,
    pub publisher_available: bool,
    pub store_available: bool,
    pub processing_stats: Option<crate::infrastructure::event_system::EventStats>,
    pub error_message: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_system_config() {
        let config = EventSystemConfig::default();
        assert!(config.enabled);
        assert!(config.use_postgres_store);
        assert!(config.enable_audit_handler);
        assert!(config.enable_statistics_handler);
        assert!(!config.enable_email_service);
        assert_eq!(config.batch_size, 100);
        assert_eq!(config.max_retries, 3);
    }
    
    #[test]
    fn test_event_service_registry() {
        let registry = EventServiceRegistry::new();
        assert!(registry.email_service.is_none());
        assert!(registry.welcome_service.is_none());
        assert!(registry.user_stats_service.is_none());
    }
    
    #[test]
    fn test_event_system_health() {
        let health = EventSystemHealth {
            is_healthy: true,
            publisher_available: true,
            store_available: true,
            processing_stats: None,
            error_message: None,
        };
        
        assert!(health.is_healthy);
        assert!(health.publisher_available);
        assert!(health.store_available);
        assert!(health.error_message.is_none());
    }
}
