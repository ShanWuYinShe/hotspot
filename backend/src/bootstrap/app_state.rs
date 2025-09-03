/// 应用状态管理
/// 
/// 定义新架构的应用状态，使用强类型和泛型进行依赖注入
/// 避免trait object，优先使用具体类型和泛型约束

use std::sync::Arc;
use crate::domain::DomainResult;
use crate::bootstrap::CapabilityRegistry;

/// 新架构应用状态
/// 使用泛型参数来避免trait object，提供更好的性能和类型安全
#[derive(Clone)]
pub struct HexagonalAppState<
    UserRepo,
    GroupRepo, 
    MessageRepo,
    MomentRepo,
    CacheService,
    JobDispatcher,
    NotificationService,
    MetricsService,
> {
    /// 能力注册表
    pub capability_registry: Arc<CapabilityRegistry>,
    
    /// 用户仓储
    pub user_repository: Arc<UserRepo>,
    
    /// 群组仓储
    pub group_repository: Arc<GroupRepo>,
    
    /// 消息仓储
    pub message_repository: Arc<MessageRepo>,
    
    /// 瞬间仓储  
    pub moment_repository: Arc<MomentRepo>,
    
    /// 缓存服务
    pub cache_service: Arc<CacheService>,
    
    /// 任务调度器
    pub job_dispatcher: Option<Arc<JobDispatcher>>,
    
    /// 通知服务
    pub notification_service: Arc<NotificationService>,
    
    /// 指标服务
    pub metrics_service: Arc<MetricsService>,
    
    /// 应用配置
    pub config: Arc<AppConfig>,
}

/// 应用配置
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AppConfig {
    /// 数据库配置
    pub database: DatabaseConfig,
    
    /// 缓存配置
    pub cache: CacheConfig,
    
    /// 任务队列配置
    pub job_queue: JobQueueConfig,
    
    /// 通知配置
    pub notification: NotificationConfig,
    
    /// WebSocket配置
    pub websocket: WebSocketConfig,
    
    /// 认证配置
    pub auth: AuthConfig,
    
    /// 监控配置
    pub metrics: MetricsConfig,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connect_timeout: u64,
    pub idle_timeout: u64,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct CacheConfig {
    pub cache_type: String, // "memory" or "redis"
    pub redis_url: Option<String>,
    pub default_ttl: u64,
    pub max_memory_size: Option<usize>,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct JobQueueConfig {
    pub queue_type: String, // "lite" or "redis"
    pub redis_url: Option<String>,
    pub max_retries: u32,
    pub retry_delay: u64,
    pub max_concurrent_jobs: u32,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct NotificationConfig {
    pub enable_friend_notifications: bool,
    pub enable_group_notifications: bool,
    pub enable_nearby_notifications: bool,
    pub nearby_radius_km: f64,
    pub enable_global_broadcast: bool, // 建议为false
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct WebSocketConfig {
    pub auth_timeout_seconds: u64,
    pub heartbeat_interval_seconds: u64,
    pub idle_timeout_seconds: u64,
    pub max_connections_per_user: u32,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct AuthConfig {
    pub jwt_secret: String,
    pub jwt_expiry_hours: i64,
    pub temp_user_expiry_hours: i64,
}

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct MetricsConfig {
    pub enable_basic_metrics: bool,
    pub enable_prometheus: bool,
    pub prometheus_endpoint: String,
    pub collection_interval_seconds: u64,
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            database: DatabaseConfig {
                url: "postgresql://hotspot:hotspot@localhost:5432/hotspot".to_string(),
                max_connections: 100,
                min_connections: 5,
                connect_timeout: 30,
                idle_timeout: 300,
            },
            cache: CacheConfig {
                cache_type: "memory".to_string(),
                redis_url: None,
                default_ttl: 3600, // 1小时
                max_memory_size: Some(100 * 1024 * 1024), // 100MB
            },
            job_queue: JobQueueConfig {
                queue_type: "lite".to_string(),
                redis_url: None,
                max_retries: 3,
                retry_delay: 5000, // 5秒
                max_concurrent_jobs: 10,
            },
            notification: NotificationConfig {
                enable_friend_notifications: true,
                enable_group_notifications: true,
                enable_nearby_notifications: false, // 默认关闭附近通知
                nearby_radius_km: 5.0,
                enable_global_broadcast: false, // 禁用全局广播
            },
            websocket: WebSocketConfig {
                auth_timeout_seconds: 5,
                heartbeat_interval_seconds: 30,
                idle_timeout_seconds: 300,
                max_connections_per_user: 5,
            },
            auth: AuthConfig {
                jwt_secret: "your-secret-key-change-in-production".to_string(),
                jwt_expiry_hours: 24, // 24小时
                temp_user_expiry_hours: 72, // 临时用户3天
            },
            metrics: MetricsConfig {
                enable_basic_metrics: true,
                enable_prometheus: false, // 默认关闭Prometheus
                prometheus_endpoint: "/metrics".to_string(),
                collection_interval_seconds: 60,
            },
        }
    }
}

impl AppConfig {
    /// 从环境变量加载配置
    pub fn from_env() -> DomainResult<Self> {
        let mut config = Self::default();
        
        // 数据库配置
        if let Ok(url) = std::env::var("DATABASE_URL") {
            config.database.url = url;
        }
        
        if let Ok(max_conn) = std::env::var("DATABASE_MAX_CONNECTIONS") {
            config.database.max_connections = max_conn.parse().unwrap_or(config.database.max_connections);
        }
        
        // 缓存配置
        if let Ok(cache_type) = std::env::var("CACHE_TYPE") {
            config.cache.cache_type = cache_type;
        }
        
        if let Ok(redis_url) = std::env::var("REDIS_URL") {
            config.cache.redis_url = Some(redis_url);
        }
        
        // 任务队列配置
        if let Ok(queue_type) = std::env::var("JOB_QUEUE_TYPE") {
            config.job_queue.queue_type = queue_type;
        }
        
        if let Ok(redis_url) = std::env::var("JOB_REDIS_URL") {
            config.job_queue.redis_url = Some(redis_url);
        }
        
        // 通知配置
        if let Ok(enable_friend) = std::env::var("ENABLE_FRIEND_NOTIFICATIONS") {
            config.notification.enable_friend_notifications = enable_friend.parse().unwrap_or(true);
        }
        
        if let Ok(enable_group) = std::env::var("ENABLE_GROUP_NOTIFICATIONS") {
            config.notification.enable_group_notifications = enable_group.parse().unwrap_or(true);
        }
        
        if let Ok(enable_nearby) = std::env::var("ENABLE_NEARBY_NOTIFICATIONS") {
            config.notification.enable_nearby_notifications = enable_nearby.parse().unwrap_or(false);
        }
        
        if let Ok(radius) = std::env::var("NEARBY_RADIUS_KM") {
            config.notification.nearby_radius_km = radius.parse().unwrap_or(5.0);
        }
        
        Ok(config)
    }
    
    /// 从配置文件加载
    pub fn from_file(path: &str) -> DomainResult<Self> {
        let content = std::fs::read_to_string(path)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Failed to read config file {}: {}", path, e),
            })?;
            
        let config: Self = toml::from_str(&content)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Failed to parse config file {}: {}", path, e),
            })?;
            
        Ok(config)
    }
    
    /// 验证配置
    pub fn validate(&self) -> DomainResult<Vec<String>> {
        let mut warnings = Vec::new();
        
        // 验证数据库配置
        if self.database.url.is_empty() {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "Database URL cannot be empty".to_string(),
            });
        }
        
        if self.database.max_connections == 0 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "Database max_connections must be greater than 0".to_string(),
            });
        }
        
        // 验证缓存配置
        if self.cache.cache_type == "redis" && self.cache.redis_url.is_none() {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "Redis URL is required when cache_type is 'redis'".to_string(),
            });
        }
        
        // 验证任务队列配置
        if self.job_queue.queue_type == "redis" && self.job_queue.redis_url.is_none() {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "Redis URL is required when queue_type is 'redis'".to_string(),
            });
        }
        
        // 警告检查
        if self.notification.enable_global_broadcast {
            warnings.push("Global broadcast is enabled - this may impact performance".to_string());
        }
        
        if self.notification.nearby_radius_km > 50.0 {
            warnings.push(format!("Nearby radius is very large: {} km", self.notification.nearby_radius_km));
        }
        
        Ok(warnings)
    }
}

/// 应用状态构建器
/// 用于分阶段构建应用状态，支持不同的能力组合
pub struct AppStateBuilder {
    config: AppConfig,
    capability_registry: CapabilityRegistry,
}

impl AppStateBuilder {
    /// 创建新的构建器
    pub fn new(config: AppConfig) -> Self {
        Self {
            config,
            capability_registry: CapabilityRegistry::default(),
        }
    }
    
    /// 设置能力注册表
    pub fn with_capability_registry(mut self, registry: CapabilityRegistry) -> Self {
        self.capability_registry = registry;
        self
    }
    
    /// 构建应用状态
    /// 这里使用泛型约束来确保类型安全，避免trait object
    pub async fn build<UserRepo, GroupRepo, MessageRepo, MomentRepo, CacheService, JobDispatcher, NotificationService, MetricsService>(
        self,
        user_repository: UserRepo,
        group_repository: GroupRepo,
        message_repository: MessageRepo,
        moment_repository: MomentRepo,
        cache_service: CacheService,
        job_dispatcher: Option<JobDispatcher>,
        notification_service: NotificationService,
        metrics_service: MetricsService,
    ) -> DomainResult<HexagonalAppState<UserRepo, GroupRepo, MessageRepo, MomentRepo, CacheService, JobDispatcher, NotificationService, MetricsService>>
    where
        UserRepo: Send + Sync + 'static,
        GroupRepo: Send + Sync + 'static,
        MessageRepo: Send + Sync + 'static,
        MomentRepo: Send + Sync + 'static,
        CacheService: Send + Sync + 'static,
        JobDispatcher: Send + Sync + 'static,
        NotificationService: Send + Sync + 'static,
        MetricsService: Send + Sync + 'static,
    {
        // 验证配置
        let warnings = self.config.validate()?;
        for warning in warnings {
            log::warn!("Configuration warning: {}", warning);
        }
        
        let state = HexagonalAppState {
            capability_registry: Arc::new(self.capability_registry),
            user_repository: Arc::new(user_repository),
            group_repository: Arc::new(group_repository),
            message_repository: Arc::new(message_repository),
            moment_repository: Arc::new(moment_repository),
            cache_service: Arc::new(cache_service),
            job_dispatcher: job_dispatcher.map(Arc::new),
            notification_service: Arc::new(notification_service),
            metrics_service: Arc::new(metrics_service),
            config: Arc::new(self.config),
        };
        
        Ok(state)
    }
}

