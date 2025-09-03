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
        #[cfg(feature = "postgres")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::Database,
                "PostgreSQL Database",
                "1.0.0",
                "PostgreSQL database adapter with sqlx",
                feature: "postgres"
            );
            registry.register_capability(metadata)?;
        }
        
        // 缓存能力
        #[cfg(any(feature = "cache_in_memory", not(feature = "cache_redis")))]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::Cache,
                "In-Memory Cache",
                "1.0.0",
                "Fast in-memory cache implementation",
                feature: "cache_in_memory"
            );
            registry.register_capability(metadata)?;
        }
        
        #[cfg(feature = "cache_redis")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::Cache,
                "Redis Cache",
                "1.0.0",
                "Distributed Redis cache implementation",
                feature: "cache_redis"
            );
            registry.register_capability(metadata)?;
        }
        
        // 任务队列能力
        #[cfg(any(feature = "jobs_lite", not(feature = "jobs_redis")))]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::JobQueue,
                "Lite Job Queue",
                "1.0.0",
                "Simple in-memory job queue",
                feature: "jobs_lite"
            );
            registry.register_capability(metadata)?;
        }
        
        #[cfg(feature = "jobs_redis")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::JobQueue,
                "Redis Job Queue",
                "1.0.0",
                "Distributed Redis job queue",
                feature: "jobs_redis"
            );
            registry.register_capability(metadata)?;
        }
        
        // WebSocket能力
        #[cfg(feature = "ws")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::WebSocket,
                "WebSocket Manager",
                "1.0.0",
                "Real-time WebSocket communication",
                feature: "ws"
            );
            registry.register_capability(metadata)?;
        }
        
        // 通知传输能力
        #[cfg(feature = "ws")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::NotificationTransport,
                "WebSocket Notification Transport",
                "1.0.0",
                "WebSocket-based notification delivery",
                deps: [CapabilityType::WebSocket],
                feature: "ws"
            );
            registry.register_capability(metadata)?;
        }
        
        // 通知策略能力
        #[cfg(feature = "policy_friend")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::NotificationPolicy,
                "Friend Notification Policy",
                "1.0.0",
                "Friend-based notification targeting",
                feature: "policy_friend"
            );
            registry.register_capability(metadata)?;
        }
        
        // 监控指标能力
        #[cfg(feature = "metrics_basic")]
        {
            let metadata = crate::capability_metadata!(
                CapabilityType::Metrics,
                "Basic Metrics",
                "1.0.0",
                "Basic application metrics collection",
                feature: "metrics_basic"
            );
            registry.register_capability(metadata)?;
        }
        
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
        registry.enable_capability(CapabilityType::JobQueue)?;
        
        // 启用WebSocket（如果可用）
        if registry.is_capability_available(&CapabilityType::WebSocket) {
            registry.enable_capability(CapabilityType::WebSocket)?;
            registry.enable_capability(CapabilityType::NotificationTransport)?;
        }
        
        // 根据通知配置启用策略
        if config.notification.enable_friend_notifications || 
           config.notification.enable_group_notifications {
            if registry.is_capability_available(&CapabilityType::NotificationPolicy) {
                registry.enable_capability(CapabilityType::NotificationPolicy)?;
            }
        }
        
        // 启用监控（如果可用）
        if registry.is_capability_available(&CapabilityType::Metrics) {
            registry.enable_capability(CapabilityType::Metrics)?;
        }
        
        Ok(())
    }
    
    /// 获取能力注册表的引用
    pub fn capability_registry(&self) -> &CapabilityRegistry {
        &self.capability_registry
    }
    
    /// 获取配置的引用
    pub fn config(&self) -> &AppConfig {
        &self.config
    }
    
    /// 创建应用状态构建器
    pub fn create_app_state_builder(self) -> AppStateBuilder {
        AppStateBuilder::new(self.config)
            .with_capability_registry(self.capability_registry)
    }
    
    /// 验证依赖完整性
    pub fn validate_dependencies(&self) -> DomainResult<Vec<String>> {
        let mut issues = Vec::new();
        
        // 检查必需的能力
        let required_capabilities = vec![
            CapabilityType::Database,
            CapabilityType::Cache,
            CapabilityType::JobQueue,
        ];
        
        for capability in required_capabilities {
            if !self.capability_registry.is_capability_enabled(&capability) {
                issues.push(format!("Required capability not enabled: {:?}", capability));
            }
        }
        
        // 检查配置一致性
        if self.config.cache.cache_type == "redis" && 
           !self.capability_registry.is_capability_enabled(&CapabilityType::Cache) {
            issues.push("Redis cache configured but cache capability not enabled".to_string());
        }
        
        if self.config.job_queue.queue_type == "redis" && 
           !self.capability_registry.is_capability_enabled(&CapabilityType::JobQueue) {
            issues.push("Redis job queue configured but job queue capability not enabled".to_string());
        }
        
        // 检查WebSocket依赖
        if self.capability_registry.is_capability_enabled(&CapabilityType::NotificationTransport) &&
           !self.capability_registry.is_capability_enabled(&CapabilityType::WebSocket) {
            issues.push("Notification transport enabled but WebSocket not available".to_string());
        }
        
        Ok(issues)
    }
    
    /// 生成依赖注入报告
    pub fn generate_injection_report(&self) -> DependencyInjectionReport {
        let status_report = self.capability_registry.generate_status_report();
        let validation_issues = self.validate_dependencies().unwrap_or_default();
        
        DependencyInjectionReport {
            capability_status: status_report,
            validation_issues,
            config_summary: ConfigSummary {
                database_type: "postgresql".to_string(),
                cache_type: self.config.cache.cache_type.clone(),
                job_queue_type: self.config.job_queue.queue_type.clone(),
                websocket_enabled: self.capability_registry.is_capability_enabled(&CapabilityType::WebSocket),
                metrics_enabled: self.capability_registry.is_capability_enabled(&CapabilityType::Metrics),
            },
        }
    }
}

/// 依赖注入报告
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct DependencyInjectionReport {
    pub capability_status: crate::bootstrap::CapabilityStatusReport,
    pub validation_issues: Vec<String>,
    pub config_summary: ConfigSummary,
}

/// 配置摘要
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConfigSummary {
    pub database_type: String,
    pub cache_type: String,
    pub job_queue_type: String,
    pub websocket_enabled: bool,
    pub metrics_enabled: bool,
}

/// 依赖注入助手函数
pub mod injection_helpers {
    use super::*;
    
    /// 创建默认的依赖注入容器
    pub fn create_default_container() -> DomainResult<DependencyContainer> {
        let config = AppConfig::from_env()?;
        DependencyContainer::new(config)
    }
    
    /// 创建基于文件的依赖注入容器
    pub fn create_container_from_file(config_path: &str) -> DomainResult<DependencyContainer> {
        let config = AppConfig::from_file(config_path)?;
        DependencyContainer::new(config)
    }
    
    /// 创建测试用的依赖注入容器
    #[cfg(test)]
    pub fn create_test_container() -> DomainResult<DependencyContainer> {
        let mut config = AppConfig::default();
        
        // 测试环境配置
        config.database.url = "postgresql://test:test@localhost:5432/test_hotspot".to_string();
        config.cache.cache_type = "memory".to_string();
        config.job_queue.queue_type = "lite".to_string();
        config.notification.enable_global_broadcast = false;
        
        DependencyContainer::new(config)
    }
    
    /// 验证运行环境
    pub fn validate_runtime_environment() -> DomainResult<Vec<String>> {
        let mut warnings = Vec::new();
        
        // 检查环境变量
        if std::env::var("DATABASE_URL").is_err() {
            warnings.push("DATABASE_URL not set, using default".to_string());
        }
        
        // 检查必需的feature
        #[cfg(not(feature = "postgres"))]
        {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "postgres feature is required".to_string(),
            });
        }
        
        // 检查可选feature的一致性
        #[cfg(feature = "cache_redis")]
        {
            // 占位：仅在启用 cache_redis 时提示，但不检查不存在的 redis feature
        }
        
        Ok(warnings)
    }
}
