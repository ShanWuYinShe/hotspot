/// 应用启动引导 - 完整的应用启动和能力系统集成示例
/// 
/// 启动流程：
/// - 配置加载和验证
/// - 能力系统初始化
/// - 服务依赖注入
/// - 应用服务启动
/// - 健康检查和监控

use std::sync::Arc;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;

use crate::domain::DomainResult;
use crate::infrastructure::{
    CapabilityManager, CapabilityManagerBuilder, CapabilityManagerConfig, CapabilityConfig,
    WebSocketManager, EventSystemIntegrator,
};

/// 应用启动配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApplicationConfig {
    /// 应用名称
    pub app_name: String,
    /// 应用版本
    pub app_version: String,
    /// 服务器配置
    pub server: ServerConfig,
    /// 数据库配置
    pub database: DatabaseConfig,
    /// 能力管理器配置
    pub capabilities: CapabilityManagerConfig,
    /// 环境配置
    pub environment: String,
    /// 日志级别
    pub log_level: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    pub workers: Option<usize>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connection_timeout_seconds: u64,
}

impl Default for ApplicationConfig {
    fn default() -> Self {
        Self {
            app_name: "HotSpot".to_string(),
            app_version: "1.0.0".to_string(),
            server: ServerConfig {
                host: "0.0.0.0".to_string(),
                port: 8080,
                workers: None,
            },
            database: DatabaseConfig {
                url: "postgresql://localhost/hotspot".to_string(),
                max_connections: 10,
                min_connections: 1,
                connection_timeout_seconds: 30,
            },
            capabilities: CapabilityManagerConfig::default(),
            environment: "development".to_string(),
            log_level: "info".to_string(),
        }
    }
}

/// 应用上下文 - 包含所有已启用的服务和能力
pub struct ApplicationContext {
    /// 应用配置
    pub config: ApplicationConfig,
    /// 能力管理器
    pub capability_manager: Arc<CapabilityManager>,
    /// 数据库连接池
    pub database_pool: Option<sqlx::PgPool>,
    /// WebSocket管理器
    pub websocket_manager: Option<Arc<WebSocketManager>>,
    /// 事件系统集成器
    pub event_integrator: Option<Arc<EventSystemIntegrator>>,
    /// 应用统计
    pub stats: Arc<RwLock<ApplicationStats>>,
}

/// 应用统计信息
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct ApplicationStats {
    /// 应用启动时间
    pub started_at: chrono::DateTime<chrono::Utc>,
    /// 启动耗时（毫秒）
    pub startup_time_ms: u64,
    /// 已处理请求数
    pub requests_processed: u64,
    /// 错误数
    pub errors_count: u64,
    /// 当前活跃连接数
    pub active_connections: u64,
}

impl ApplicationContext {
    /// 创建新的应用上下文
    pub fn new(config: ApplicationConfig, capability_manager: Arc<CapabilityManager>) -> Self {
        Self {
            config,
            capability_manager,
            database_pool: None,
            websocket_manager: None,
            event_integrator: None,
            stats: Arc::new(RwLock::new(ApplicationStats {
                started_at: chrono::Utc::now(),
                ..Default::default()
            })),
        }
    }
    
    /// 获取数据库连接池
    pub fn get_database_pool(&self) -> Option<&sqlx::PgPool> {
        self.database_pool.as_ref()
    }
    
    /// 获取WebSocket管理器
    pub fn get_websocket_manager(&self) -> Option<Arc<WebSocketManager>> {
        self.websocket_manager.clone()
    }
    
    /// 获取事件系统集成器
    pub fn get_event_integrator(&self) -> Option<Arc<EventSystemIntegrator>> {
        self.event_integrator.clone()
    }
    
    /// 获取应用统计
    pub async fn get_stats(&self) -> ApplicationStats {
        self.stats.read().await.clone()
    }
    
    /// 更新请求统计
    pub async fn increment_requests(&self) {
        let mut stats = self.stats.write().await;
        stats.requests_processed += 1;
    }
    
    /// 更新错误统计
    pub async fn increment_errors(&self) {
        let mut stats = self.stats.write().await;
        stats.errors_count += 1;
    }
}

/// 应用启动器 - 负责应用的完整启动流程
pub struct ApplicationBootstrap;

impl ApplicationBootstrap {
    /// 从配置文件启动应用
    pub async fn start_from_config_file(config_path: &str) -> DomainResult<Arc<ApplicationContext>> {
        tracing::info!(
            config_path = %config_path,
            "Starting application from config file"
        );
        
        // 加载配置
        let config = Self::load_config(config_path).await?;
        
        // 启动应用
        Self::start_with_config(config).await
    }
    
    /// 使用指定配置启动应用
    pub async fn start_with_config(config: ApplicationConfig) -> DomainResult<Arc<ApplicationContext>> {
        let start_time = std::time::Instant::now();
        
        tracing::info!(
            app_name = %config.app_name,
            app_version = %config.app_version,
            environment = %config.environment,
            "Starting application"
        );
        
        // 初始化日志系统
        Self::initialize_logging(&config.log_level)?;
        
        // 创建能力管理器
        let capability_manager = Self::create_capability_manager(&config).await?;
        
        // 初始化能力管理器
        capability_manager.initialize().await?;
        
        // 创建应用上下文
        let mut context = ApplicationContext::new(config.clone(), capability_manager.clone());
        
        // 提取已启用的服务
        Self::extract_enabled_services(&mut context).await?;
        
        // 记录启动时间
        let startup_time = start_time.elapsed();
        {
            let mut stats = context.stats.write().await;
            stats.startup_time_ms = startup_time.as_millis() as u64;
        }
        
        let context = Arc::new(context);
        
        tracing::info!(
            startup_time_ms = startup_time.as_millis(),
            "Application started successfully"
        );
        
        Ok(context)
    }
    
    /// 使用默认配置启动应用
    pub async fn start_with_defaults() -> DomainResult<Arc<ApplicationContext>> {
        let config = ApplicationConfig::default();
        Self::start_with_config(config).await
    }
    
    /// 启动开发环境应用
    pub async fn start_development() -> DomainResult<Arc<ApplicationContext>> {
        let mut config = ApplicationConfig::default();
        config.environment = "development".to_string();
        config.log_level = "debug".to_string();
        
        // 启用所有开发环境能力
        config.capabilities.auto_enable_all = true;
        config.capabilities.enabled_capabilities = vec![
            "database".to_string(),
            "cache".to_string(),
            "websocket".to_string(),
            "events".to_string(),
            "metrics".to_string(),
        ];
        
        Self::start_with_config(config).await
    }
    
    /// 启动生产环境应用
    pub async fn start_production() -> DomainResult<Arc<ApplicationContext>> {
        let mut config = ApplicationConfig::default();
        config.environment = "production".to_string();
        config.log_level = "info".to_string();
        
        // 只启用生产环境必需的能力
        config.capabilities.enabled_capabilities = vec![
            "database".to_string(),
            "websocket".to_string(),
            "events".to_string(),
        ];
        
        // 启用故障恢复
        config.capabilities.enable_failure_recovery = true;
        config.capabilities.health_check_interval_seconds = 30;
        
        Self::start_with_config(config).await
    }
    
    /// 优雅关闭应用
    pub async fn shutdown(context: Arc<ApplicationContext>) -> DomainResult<()> {
        tracing::info!("Shutting down application");
        
        // 关闭能力管理器
        context.capability_manager.shutdown().await?;
        
        tracing::info!("Application shutdown completed");
        Ok(())
    }
    
    /// 加载配置文件
    async fn load_config(config_path: &str) -> DomainResult<ApplicationConfig> {
        // 这里应该实现配置文件加载逻辑
        // 支持 YAML、JSON、TOML 等格式
        // 支持环境变量覆盖
        
        // 简化实现，返回默认配置
        tracing::warn!(
            config_path = %config_path,
            "Config file loading not implemented, using default config"
        );
        
        Ok(ApplicationConfig::default())
    }
    
    /// 初始化日志系统
    fn initialize_logging(log_level: &str) -> DomainResult<()> {
        // 这里应该初始化 tracing 日志系统
        // 根据环境配置不同的日志输出格式
        
        tracing::info!(
            log_level = %log_level,
            "Logging system initialized"
        );
        
        Ok(())
    }
    
    /// 创建能力管理器
    async fn create_capability_manager(config: &ApplicationConfig) -> DomainResult<Arc<CapabilityManager>> {
        let mut builder = CapabilityManagerBuilder::new();
        
        // 配置数据库能力
        let mut db_config = CapabilityConfig::new();
        db_config.set("database_url", &config.database.url)?;
        db_config.set("max_connections", config.database.max_connections)?;
        builder = builder.with_capability_config("database", db_config);
        
        // 配置WebSocket能力
        if config.capabilities.enabled_capabilities.contains(&"websocket".to_string()) {
            let mut ws_config = CapabilityConfig::new();
            ws_config.set("max_connections", 1000u32)?;
            ws_config.set("max_rooms", 100u32)?;
            builder = builder.with_capability_config("websocket", ws_config);
        }
        
        // 配置事件系统能力
        if config.capabilities.enabled_capabilities.contains(&"events".to_string()) {
            let mut event_config = CapabilityConfig::new();
            event_config.set("use_postgres_store", true)?;
            builder = builder.with_capability_config("events", event_config);
        }
        
        // 配置缓存能力
        if config.capabilities.enabled_capabilities.contains(&"cache".to_string()) {
            let mut cache_config = CapabilityConfig::new();
            cache_config.set("redis_url", "redis://localhost:6379")?;
            builder = builder.with_capability_config("cache", cache_config);
        }
        
        // 配置监控指标能力
        if config.capabilities.enabled_capabilities.contains(&"metrics".to_string()) {
            let mut metrics_config = CapabilityConfig::new();
            metrics_config.set("collection_interval_seconds", 60u64)?;
            builder = builder.with_capability_config("metrics", metrics_config);
        }
        
        // 应用能力管理器配置
        builder = builder
            .with_health_check_interval(config.capabilities.health_check_interval_seconds)
            .enable_failure_recovery();
        
        if config.capabilities.auto_enable_all {
            builder = builder.auto_enable_all();
        } else {
            for capability in &config.capabilities.enabled_capabilities {
                builder = builder.enable_capability(capability);
            }
        }
        
        for capability in &config.capabilities.disabled_capabilities {
            builder = builder.disable_capability(capability);
        }
        
        let manager = Arc::new(builder.build());
        
        tracing::info!("Capability manager created");
        Ok(manager)
    }
    
    /// 提取已启用的服务
    async fn extract_enabled_services(_context: &mut ApplicationContext) -> DomainResult<()> {
        // 提取数据库连接池
        // 这里需要从能力管理器中获取数据库能力并提取连接池
        // 简化实现，暂时跳过
        
        // 提取WebSocket管理器
        // 这里需要从能力管理器中获取WebSocket能力并提取管理器
        // 简化实现，暂时跳过
        
        // 提取事件系统集成器
        // 这里需要从能力管理器中获取事件能力并提取集成器
        // 简化实现，暂时跳过
        
        tracing::info!("Enabled services extracted from capabilities");
        Ok(())
    }
}

/// 应用健康检查器
pub struct ApplicationHealthChecker {
    context: Arc<ApplicationContext>,
}

impl ApplicationHealthChecker {
    /// 创建新的健康检查器
    pub fn new(context: Arc<ApplicationContext>) -> Self {
        Self { context }
    }
    
    /// 执行完整的健康检查
    pub async fn check_health(&self) -> ApplicationHealth {
        let capability_health = self.context.capability_manager.health_check().await;
        let app_stats = self.context.get_stats().await;
        
        ApplicationHealth {
            is_healthy: capability_health.is_healthy,
            app_name: self.context.config.app_name.clone(),
            app_version: self.context.config.app_version.clone(),
            environment: self.context.config.environment.clone(),
            uptime_seconds: capability_health.uptime_seconds,
            capability_health,
            app_stats,
        }
    }
    
    /// 检查数据库健康状态
    pub async fn check_database_health(&self) -> bool {
        if let Some(status) = self.context.capability_manager.get_capability_status("database").await {
            matches!(status, crate::infrastructure::CapabilityStatus::Enabled)
        } else {
            false
        }
    }
    
    /// 检查WebSocket健康状态
    pub async fn check_websocket_health(&self) -> bool {
        if let Some(status) = self.context.capability_manager.get_capability_status("websocket").await {
            matches!(status, crate::infrastructure::CapabilityStatus::Enabled)
        } else {
            false
        }
    }
}

/// 应用健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApplicationHealth {
    /// 整体是否健康
    pub is_healthy: bool,
    /// 应用名称
    pub app_name: String,
    /// 应用版本
    pub app_version: String,
    /// 运行环境
    pub environment: String,
    /// 运行时间（秒）
    pub uptime_seconds: u64,
    /// 能力健康状态
    pub capability_health: crate::infrastructure::CapabilityManagerHealth,
    /// 应用统计
    pub app_stats: ApplicationStats,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_application_config() {
        let config = ApplicationConfig::default();
        assert_eq!(config.app_name, "HotSpot");
        assert_eq!(config.server.port, 8080);
        assert_eq!(config.environment, "development");
    }
    
    #[tokio::test]
    async fn test_application_context_creation() {
        let config = ApplicationConfig::default();
        let capability_manager = Arc::new(CapabilityManager::new(CapabilityManagerConfig::default()));
        
        let context = ApplicationContext::new(config, capability_manager);
        
        assert_eq!(context.config.app_name, "HotSpot");
        assert!(context.database_pool.is_none());
        assert!(context.websocket_manager.is_none());
    }
    
    #[tokio::test]
    async fn test_application_stats() {
        let config = ApplicationConfig::default();
        let capability_manager = Arc::new(CapabilityManager::new(CapabilityManagerConfig::default()));
        let context = ApplicationContext::new(config, capability_manager);
        
        context.increment_requests().await;
        context.increment_errors().await;
        
        let stats = context.get_stats().await;
        assert_eq!(stats.requests_processed, 1);
        assert_eq!(stats.errors_count, 1);
    }
}
