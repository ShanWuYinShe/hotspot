/// 依赖注入系统
/// 
/// 负责根据配置和feature flags组装具体的依赖实现
/// 使用编译期和运行期相结合的方式进行依赖选择
use crate::domain::DomainResult;
use crate::bootstrap::{CapabilityRegistry, CapabilityType, CapabilityMetadata, CapabilityStatus, AppConfig, AppStateBuilder};

/// 依赖注入容器
pub struct DependencyContainer {
    config: AppConfig,
    capability_registry: CapabilityRegistry,
}

impl DependencyContainer {
    /// 创建新的依赖注入容器
    pub fn new(config: AppConfig) -> DomainResult<Self> {
        let mut capability_registry = CapabilityRegistry::default();
        
        // 注册默认能力
        Self::register_default_capabilities(&mut capability_registry)?;
        
        // 根据配置启用能力
        Self::enable_capabilities_by_config(&mut capability_registry, &config)?;
        
        Ok(Self {
            config,
            capability_registry,
        })
    }
    
    /// 注册默认能力
    fn register_default_capabilities(registry: &mut CapabilityRegistry) -> DomainResult<()> {
        // 数据库能力（必需）
        // PostgreSQL 是核心功能，始终启用
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::Database,
                name: "PostgreSQL".to_string(),
                version: "1.0.0".to_string(),
                description: "PostgreSQL数据库连接".to_string(),
                dependencies: vec![],
                feature_flag: Some("postgres".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // 缓存能力（默认内存缓存）
        #[cfg(any(feature = "cache_memory", not(feature = "cache_redis")))]
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::Cache,
                name: "MemoryCache".to_string(),
                version: "1.0.0".to_string(),
                description: "内存缓存服务".to_string(),
                dependencies: vec![],
                feature_flag: Some("cache_in_memory".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // Redis缓存能力（可选）
        #[cfg(feature = "cache_redis")]
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::Cache,
                name: "RedisCache".to_string(),
                version: "1.0.0".to_string(),
                description: "Redis缓存服务".to_string(),
                dependencies: vec![],
                feature_flag: Some("cache_redis".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // 任务队列能力（默认轻量级）
        #[cfg(any(feature = "jobs_memory", not(feature = "jobs_redis")))]
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::JobQueue,
                name: "LiteJobQueue".to_string(),
                version: "1.0.0".to_string(),
                description: "轻量级任务队列".to_string(),
                dependencies: vec![],
                feature_flag: Some("jobs_lite".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // Redis任务队列能力（可选）
        #[cfg(feature = "jobs_redis")]
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::JobQueue,
                name: "RedisJobQueue".to_string(),
                version: "1.0.0".to_string(),
                description: "Redis任务队列".to_string(),
                dependencies: vec![],
                feature_flag: Some("jobs_redis".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // WebSocket通知能力（默认且必需）
        // WebSocket 是核心功能，始终启用
        {
            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::NotificationTransport,
                name: "WebSocketNotification".to_string(),
                version: "1.0.0".to_string(),
                description: "WebSocket实时通知".to_string(),
                dependencies: vec![],
                feature_flag: Some("ws".to_string()),
                status: CapabilityStatus::Registered,
            })?;

            registry.register_capability(CapabilityMetadata {
                capability_type: CapabilityType::WebSocket,
                name: "WebSocketManager".to_string(),
                version: "1.0.0".to_string(),
                description: "WebSocket连接管理".to_string(),
                dependencies: vec![],
                feature_flag: Some("ws".to_string()),
                status: CapabilityStatus::Registered,
            })?;
        }

        // 通知策略能力（默认启用好友和群组策略）
        registry.register_capability(CapabilityMetadata {
            capability_type: CapabilityType::NotificationPolicy,
            name: "FriendGroupPolicy".to_string(),
            version: "1.0.0".to_string(),
            description: "好友和群组通知策略".to_string(),
            dependencies: vec![CapabilityType::NotificationTransport],
            feature_flag: None,
            status: CapabilityStatus::Registered,
        })?;

        // 地理位置服务能力
        registry.register_capability(CapabilityMetadata {
            capability_type: CapabilityType::GeoService,
            name: "PostGISGeoService".to_string(),
            version: "1.0.0".to_string(),
            description: "PostGIS地理位置服务".to_string(),
            dependencies: vec![CapabilityType::Database],
            feature_flag: Some("postgres".to_string()),
            status: CapabilityStatus::Registered,
        })?;

        // 基础监控指标能力
        registry.register_capability(CapabilityMetadata {
            capability_type: CapabilityType::Metrics,
            name: "BasicMetrics".to_string(),
            version: "1.0.0".to_string(),
            description: "基础监控指标".to_string(),
            dependencies: vec![],
            feature_flag: None,
            status: CapabilityStatus::Registered,
        })?;

        Ok(())
    }
    
    /// 根据配置启用能力
    fn enable_capabilities_by_config(
        registry: &mut CapabilityRegistry,
        config: &AppConfig,
    ) -> DomainResult<()> {
        // 启用数据库（必需）
        registry.enable_capability(CapabilityType::Database)?;
        
        // 根据配置启用缓存
        registry.enable_capability(CapabilityType::Cache)?;
        
        // 根据配置启用任务队列
        if config.job_queue.queue_type != "disabled" {
            registry.enable_capability(CapabilityType::JobQueue)?;
        }
        
        // 启用WebSocket（如果可用）
        if config.websocket.auth_timeout_seconds > 0 {
            registry.enable_capability(CapabilityType::WebSocket)?;
            registry.enable_capability(CapabilityType::NotificationTransport)?;
        }
        
        // 根据通知配置启用策略
        if config.notification.enable_friend_notifications || 
           config.notification.enable_group_notifications {
            registry.enable_capability(CapabilityType::NotificationPolicy)?;
        }
        
        // 启用监控（如果可用）
        if config.metrics.enable_basic_metrics {
            registry.enable_capability(CapabilityType::Metrics)?;
        }
        
        // 启用地理位置服务
        registry.enable_capability(CapabilityType::GeoService)?;
        
        Ok(())
    }
    
    /// 构建应用状态
    pub fn build_app_state(&self) -> DomainResult<AppStateBuilder> {
        let builder = AppStateBuilder::new(self.config.clone());
        Ok(builder)
    }
    
    /// 获取能力注册表
    pub fn get_capability_registry(&self) -> &CapabilityRegistry {
        &self.capability_registry
    }
    
    /// 获取配置
    pub fn get_config(&self) -> &AppConfig {
        &self.config
    }
}
