/// 具体能力实现 - 系统各种能力的具体实现
/// 
/// 能力类型：
/// - DatabaseCapability: 数据库连接能力
/// - CacheCapability: 缓存服务能力
/// - WebSocketCapability: WebSocket服务能力
/// - EventCapability: 事件系统能力
/// - MetricsCapability: 监控指标能力
/// - EmailCapability: 邮件服务能力

use std::collections::HashMap;
use std::sync::Arc;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;

use crate::domain::DomainResult;
use crate::infrastructure::{
    Capability, CapabilityMetadata, CapabilityId, CapabilityVersion, CapabilityDependency,
    CapabilityConfig, CapabilityHealthStatus, CapabilityStats,
    WebSocketManager, WebSocketManagerConfig,
    EventSystemIntegrator, EventSystemConfig,
};

// === 数据库能力 ===

/// 数据库连接能力
pub struct DatabaseCapability {
    metadata: CapabilityMetadata,
    pool: Option<sqlx::PgPool>,
    connection_stats: Arc<RwLock<DatabaseStats>>,
}

#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct DatabaseStats {
    pub total_connections: u32,
    pub active_connections: u32,
    pub queries_executed: u64,
    pub query_errors: u64,
    pub avg_query_time_ms: f64,
}

impl DatabaseCapability {
    /// 创建新的数据库能力
    pub fn new() -> Self {
        let metadata = CapabilityMetadata {
            id: CapabilityId::new("database"),
            name: "Database Connection".to_string(),
            description: "PostgreSQL database connection and query execution".to_string(),
            version: CapabilityVersion::new(1, 0, 0),
            dependencies: vec![],
            provides: vec!["database".to_string(), "persistence".to_string()],
            required_config: vec!["database_url".to_string()],
            is_core: true,
            feature_flag: None, // 数据库是核心能力，总是启用
        };
        
        Self {
            metadata,
            pool: None,
            connection_stats: Arc::new(RwLock::new(DatabaseStats::default())),
        }
    }
    
    /// 获取数据库连接池
    pub fn get_pool(&self) -> Option<&sqlx::PgPool> {
        self.pool.as_ref()
    }
}

#[async_trait]
impl Capability for DatabaseCapability {
    fn metadata(&self) -> &CapabilityMetadata {
        &self.metadata
    }
    
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()> {
        let database_url: String = config.get_required("database_url")?;
        
        let pool = sqlx::postgres::PgPoolOptions::new()
            .max_connections(config.get("max_connections")?.unwrap_or(10))
            .connect(&database_url)
            .await
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Failed to connect to database: {}", e),
            })?;
        
        self.pool = Some(pool);
        
        tracing::info!(
            database_url = %database_url,
            "Database capability initialized"
        );
        
        Ok(())
    }
    
    async fn start(&mut self) -> DomainResult<()> {
        if let Some(pool) = &self.pool {
            // 测试数据库连接
            sqlx::query("SELECT 1")
                .execute(pool)
                .await
                .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Database connection test failed: {}", e),
                })?;
            
            tracing::info!("Database capability started successfully");
        }
        
        Ok(())
    }
    
    async fn stop(&mut self) -> DomainResult<()> {
        if let Some(pool) = &self.pool {
            pool.close().await;
            self.pool = None;
            
            tracing::info!("Database capability stopped");
        }
        
        Ok(())
    }
    
    async fn health_check(&self) -> CapabilityHealthStatus {
        if let Some(pool) = &self.pool {
            match sqlx::query("SELECT 1").execute(pool).await {
                Ok(_) => {
                    let stats = self.connection_stats.read().await;
                    CapabilityHealthStatus::healthy("Database connection is healthy")
                        .with_detail("active_connections", stats.active_connections)
                        .with_detail("queries_executed", stats.queries_executed)
                }
                Err(e) => {
                    CapabilityHealthStatus::unhealthy(&format!("Database connection failed: {}", e))
                }
            }
        } else {
            CapabilityHealthStatus::unhealthy("Database pool not initialized")
        }
    }
    
    async fn get_stats(&self) -> CapabilityStats {
        let db_stats = self.connection_stats.read().await;
        let mut stats = CapabilityStats::default();
        
        stats.custom_metrics.insert("active_connections".to_string(), db_stats.active_connections as f64);
        stats.custom_metrics.insert("queries_executed".to_string(), db_stats.queries_executed as f64);
        stats.custom_metrics.insert("query_errors".to_string(), db_stats.query_errors as f64);
        stats.custom_metrics.insert("avg_query_time_ms".to_string(), db_stats.avg_query_time_ms);
        
        stats
    }
}

// === 缓存能力 ===

/// 缓存服务能力
pub struct CacheCapability {
    metadata: CapabilityMetadata,
    redis_client: Option<redis::Client>,
    cache_stats: Arc<RwLock<CacheStats>>,
}

#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct CacheStats {
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub cache_sets: u64,
    pub cache_deletes: u64,
    pub hit_rate: f64,
}

impl CacheCapability {
    /// 创建新的缓存能力
    pub fn new() -> Self {
        let metadata = CapabilityMetadata {
            id: CapabilityId::new("cache"),
            name: "Cache Service".to_string(),
            description: "Redis-based caching service".to_string(),
            version: CapabilityVersion::new(1, 0, 0),
            dependencies: vec![],
            provides: vec!["cache".to_string(), "session".to_string()],
            required_config: vec!["redis_url".to_string()],
            is_core: false,
            feature_flag: Some("cache".to_string()),
        };
        
        Self {
            metadata,
            redis_client: None,
            cache_stats: Arc::new(RwLock::new(CacheStats::default())),
        }
    }
}

#[async_trait]
impl Capability for CacheCapability {
    fn metadata(&self) -> &CapabilityMetadata {
        &self.metadata
    }
    
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()> {
        let redis_url: String = config.get_required("redis_url")?;
        
        let client = redis::Client::open(redis_url.as_str())
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Failed to create Redis client: {}", e),
            })?;
        
        self.redis_client = Some(client);
        
        tracing::info!(
            redis_url = %redis_url,
            "Cache capability initialized"
        );
        
        Ok(())
    }
    
    async fn start(&mut self) -> DomainResult<()> {
        if let Some(client) = &self.redis_client {
            // 测试Redis连接
            let mut conn = client.get_async_connection().await
                .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Failed to connect to Redis: {}", e),
                })?;
            
            redis::cmd("PING").query_async::<String>(&mut conn).await
                .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Redis ping failed: {}", e),
                })?;
            
            tracing::info!("Cache capability started successfully");
        }
        
        Ok(())
    }
    
    async fn stop(&mut self) -> DomainResult<()> {
        self.redis_client = None;
        tracing::info!("Cache capability stopped");
        Ok(())
    }
    
    async fn health_check(&self) -> CapabilityHealthStatus {
        if let Some(client) = &self.redis_client {
            match client.get_async_connection().await {
                Ok(mut conn) => {
                    match redis::cmd("PING").query_async::<String>(&mut conn).await {
                        Ok(_) => {
                            let stats = self.cache_stats.read().await;
                            CapabilityHealthStatus::healthy("Cache service is healthy")
                                .with_detail("hit_rate", stats.hit_rate)
                                .with_detail("cache_hits", stats.cache_hits)
                        }
                        Err(e) => {
                            CapabilityHealthStatus::unhealthy(&format!("Cache ping failed: {}", e))
                        }
                    }
                }
                Err(e) => {
                    CapabilityHealthStatus::unhealthy(&format!("Cache connection failed: {}", e))
                }
            }
        } else {
            CapabilityHealthStatus::unhealthy("Cache client not initialized")
        }
    }
    
    async fn get_stats(&self) -> CapabilityStats {
        let cache_stats = self.cache_stats.read().await;
        let mut stats = CapabilityStats::default();
        
        stats.custom_metrics.insert("cache_hits".to_string(), cache_stats.cache_hits as f64);
        stats.custom_metrics.insert("cache_misses".to_string(), cache_stats.cache_misses as f64);
        stats.custom_metrics.insert("hit_rate".to_string(), cache_stats.hit_rate);
        
        stats
    }
}

// === WebSocket能力 ===

/// WebSocket服务能力
pub struct WebSocketCapability {
    metadata: CapabilityMetadata,
    manager: Option<Arc<WebSocketManager>>,
    websocket_stats: Arc<RwLock<WebSocketCapabilityStats>>,
}

#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct WebSocketCapabilityStats {
    pub total_connections: u64,
    pub active_connections: u64,
    pub messages_sent: u64,
    pub messages_received: u64,
    pub connection_errors: u64,
}

impl WebSocketCapability {
    /// 创建新的WebSocket能力
    pub fn new() -> Self {
        let metadata = CapabilityMetadata {
            id: CapabilityId::new("websocket"),
            name: "WebSocket Service".to_string(),
            description: "Real-time WebSocket communication service".to_string(),
            version: CapabilityVersion::new(1, 0, 0),
            dependencies: vec![],
            provides: vec!["websocket".to_string(), "realtime".to_string()],
            required_config: vec!["max_connections".to_string()],
            is_core: false,
            feature_flag: Some("websocket".to_string()),
        };
        
        Self {
            metadata,
            manager: None,
            websocket_stats: Arc::new(RwLock::new(WebSocketCapabilityStats::default())),
        }
    }
    
    /// 获取WebSocket管理器
    pub fn get_manager(&self) -> Option<Arc<WebSocketManager>> {
        self.manager.clone()
    }
}

#[async_trait]
impl Capability for WebSocketCapability {
    fn metadata(&self) -> &CapabilityMetadata {
        &self.metadata
    }
    
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()> {
        let max_connections: usize = config.get("max_connections")?.unwrap_or(1000);
        let max_rooms: usize = config.get("max_rooms")?.unwrap_or(100);
        
        let ws_config = WebSocketManagerConfig {
            max_connections,
            max_rooms,
            connection_timeout_seconds: 300,
            heartbeat_interval_seconds: 30,
            enable_presence: true,
            enable_message_persistence: false,
        };
        
        let manager = WebSocketManager::new(ws_config).await?;
        self.manager = Some(manager);
        
        tracing::info!(
            max_connections = max_connections,
            max_rooms = max_rooms,
            "WebSocket capability initialized"
        );
        
        Ok(())
    }
    
    async fn start(&mut self) -> DomainResult<()> {
        if self.manager.is_some() {
            tracing::info!("WebSocket capability started successfully");
        }
        Ok(())
    }
    
    async fn stop(&mut self) -> DomainResult<()> {
        if let Some(manager) = &self.manager {
            manager.shutdown().await?;
            self.manager = None;
            tracing::info!("WebSocket capability stopped");
        }
        Ok(())
    }
    
    async fn health_check(&self) -> CapabilityHealthStatus {
        if let Some(manager) = &self.manager {
            let stats = manager.get_stats().await;
            CapabilityHealthStatus::healthy("WebSocket service is healthy")
                .with_detail("active_connections", stats.active_connections)
                .with_detail("active_rooms", stats.active_rooms)
                .with_detail("messages_processed", stats.messages_processed)
        } else {
            CapabilityHealthStatus::unhealthy("WebSocket manager not initialized")
        }
    }
    
    async fn get_stats(&self) -> CapabilityStats {
        let ws_stats = self.websocket_stats.read().await;
        let mut stats = CapabilityStats::default();
        
        stats.custom_metrics.insert("active_connections".to_string(), ws_stats.active_connections as f64);
        stats.custom_metrics.insert("messages_sent".to_string(), ws_stats.messages_sent as f64);
        stats.custom_metrics.insert("connection_errors".to_string(), ws_stats.connection_errors as f64);
        
        stats
    }
}

// === 事件系统能力 ===

/// 事件系统能力
pub struct EventCapability {
    metadata: CapabilityMetadata,
    integrator: Option<Arc<EventSystemIntegrator>>,
    event_stats: Arc<RwLock<EventCapabilityStats>>,
}

#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct EventCapabilityStats {
    pub events_published: u64,
    pub events_processed: u64,
    pub events_failed: u64,
    pub handlers_registered: u32,
    pub avg_processing_time_ms: f64,
}

impl EventCapability {
    /// 创建新的事件系统能力
    pub fn new() -> Self {
        let metadata = CapabilityMetadata {
            id: CapabilityId::new("events"),
            name: "Event System".to_string(),
            description: "Domain event publishing and processing system".to_string(),
            version: CapabilityVersion::new(1, 0, 0),
            dependencies: vec![
                CapabilityDependency {
                    capability_id: CapabilityId::new("database"),
                    min_version: CapabilityVersion::new(1, 0, 0),
                    optional: false,
                },
            ],
            provides: vec!["events".to_string(), "messaging".to_string()],
            required_config: vec!["use_postgres_store".to_string()],
            is_core: false,
            feature_flag: Some("events".to_string()),
        };
        
        Self {
            metadata,
            integrator: None,
            event_stats: Arc::new(RwLock::new(EventCapabilityStats::default())),
        }
    }
    
    /// 获取事件系统集成器
    pub fn get_integrator(&self) -> Option<Arc<EventSystemIntegrator>> {
        self.integrator.clone()
    }
}

#[async_trait]
impl Capability for EventCapability {
    fn metadata(&self) -> &CapabilityMetadata {
        &self.metadata
    }
    
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()> {
        let use_postgres_store: bool = config.get("use_postgres_store")?.unwrap_or(true);
        
        let event_config = EventSystemConfig {
            enabled: true,
            use_postgres_store,
            enable_audit_handler: true,
            enable_statistics_handler: true,
            enable_email_service: false,
            batch_size: 100,
            max_retries: 3,
        };
        
        let mut integrator = EventSystemIntegrator::new(event_config);
        integrator.initialize(None).await?; // 这里应该传入数据库连接池
        
        self.integrator = Some(Arc::new(integrator));
        
        tracing::info!(
            use_postgres_store = use_postgres_store,
            "Event system capability initialized"
        );
        
        Ok(())
    }
    
    async fn start(&mut self) -> DomainResult<()> {
        if self.integrator.is_some() {
            tracing::info!("Event system capability started successfully");
        }
        Ok(())
    }
    
    async fn stop(&mut self) -> DomainResult<()> {
        self.integrator = None;
        tracing::info!("Event system capability stopped");
        Ok(())
    }
    
    async fn health_check(&self) -> CapabilityHealthStatus {
        if let Some(integrator) = &self.integrator {
            if let Some(stats) = integrator.get_processing_stats().await {
                CapabilityHealthStatus::healthy("Event system is healthy")
                    .with_detail("events_published", stats.total_published)
                    .with_detail("events_processed", stats.total_processed)
                    .with_detail("events_failed", stats.total_failed)
            } else {
                CapabilityHealthStatus::unhealthy("Event system statistics not available")
            }
        } else {
            CapabilityHealthStatus::unhealthy("Event system integrator not initialized")
        }
    }
    
    async fn get_stats(&self) -> CapabilityStats {
        let event_stats = self.event_stats.read().await;
        let mut stats = CapabilityStats::default();
        
        stats.custom_metrics.insert("events_published".to_string(), event_stats.events_published as f64);
        stats.custom_metrics.insert("events_processed".to_string(), event_stats.events_processed as f64);
        stats.custom_metrics.insert("events_failed".to_string(), event_stats.events_failed as f64);
        stats.custom_metrics.insert("handlers_registered".to_string(), event_stats.handlers_registered as f64);
        
        stats
    }
}

// === 监控指标能力 ===

/// 监控指标能力
pub struct MetricsCapability {
    metadata: CapabilityMetadata,
    metrics_collector: Option<Arc<MetricsCollector>>,
    metrics_stats: Arc<RwLock<MetricsCapabilityStats>>,
}

#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct MetricsCapabilityStats {
    pub metrics_collected: u64,
    pub metrics_exported: u64,
    pub collection_errors: u64,
    pub active_collectors: u32,
}

/// 简单的指标收集器
pub struct MetricsCollector {
    metrics: Arc<RwLock<HashMap<String, f64>>>,
}

impl MetricsCollector {
    pub fn new() -> Self {
        Self {
            metrics: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    pub async fn record_metric(&self, name: &str, value: f64) {
        let mut metrics = self.metrics.write().await;
        metrics.insert(name.to_string(), value);
    }
    
    pub async fn get_metrics(&self) -> HashMap<String, f64> {
        self.metrics.read().await.clone()
    }
}

impl MetricsCapability {
    /// 创建新的监控指标能力
    pub fn new() -> Self {
        let metadata = CapabilityMetadata {
            id: CapabilityId::new("metrics"),
            name: "Metrics Collection".to_string(),
            description: "Application metrics collection and export".to_string(),
            version: CapabilityVersion::new(1, 0, 0),
            dependencies: vec![],
            provides: vec!["metrics".to_string(), "monitoring".to_string()],
            required_config: vec!["collection_interval_seconds".to_string()],
            is_core: false,
            feature_flag: Some("metrics".to_string()),
        };
        
        Self {
            metadata,
            metrics_collector: None,
            metrics_stats: Arc::new(RwLock::new(MetricsCapabilityStats::default())),
        }
    }
    
    /// 获取指标收集器
    pub fn get_collector(&self) -> Option<Arc<MetricsCollector>> {
        self.metrics_collector.clone()
    }
}

#[async_trait]
impl Capability for MetricsCapability {
    fn metadata(&self) -> &CapabilityMetadata {
        &self.metadata
    }
    
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()> {
        let _collection_interval: u64 = config.get("collection_interval_seconds")?.unwrap_or(60);
        
        let collector = Arc::new(MetricsCollector::new());
        self.metrics_collector = Some(collector);
        
        tracing::info!("Metrics capability initialized");
        Ok(())
    }
    
    async fn start(&mut self) -> DomainResult<()> {
        if self.metrics_collector.is_some() {
            tracing::info!("Metrics capability started successfully");
        }
        Ok(())
    }
    
    async fn stop(&mut self) -> DomainResult<()> {
        self.metrics_collector = None;
        tracing::info!("Metrics capability stopped");
        Ok(())
    }
    
    async fn health_check(&self) -> CapabilityHealthStatus {
        if let Some(collector) = &self.metrics_collector {
            let metrics = collector.get_metrics().await;
            CapabilityHealthStatus::healthy("Metrics collection is healthy")
                .with_detail("metrics_count", metrics.len())
        } else {
            CapabilityHealthStatus::unhealthy("Metrics collector not initialized")
        }
    }
    
    async fn get_stats(&self) -> CapabilityStats {
        let metrics_stats = self.metrics_stats.read().await;
        let mut stats = CapabilityStats::default();
        
        stats.custom_metrics.insert("metrics_collected".to_string(), metrics_stats.metrics_collected as f64);
        stats.custom_metrics.insert("metrics_exported".to_string(), metrics_stats.metrics_exported as f64);
        stats.custom_metrics.insert("collection_errors".to_string(), metrics_stats.collection_errors as f64);
        
        stats
    }
}

// === 能力工厂 ===

/// 能力工厂 - 创建各种能力实例
pub struct CapabilityFactory;

impl CapabilityFactory {
    /// 创建数据库能力
    pub fn create_database_capability() -> Box<dyn Capability> {
        Box::new(DatabaseCapability::new())
    }
    
    /// 创建缓存能力
    #[cfg(feature = "cache")]
    pub fn create_cache_capability() -> Box<dyn Capability> {
        Box::new(CacheCapability::new())
    }
    
    /// 创建WebSocket能力
    #[cfg(feature = "websocket")]
    pub fn create_websocket_capability() -> Box<dyn Capability> {
        Box::new(WebSocketCapability::new())
    }
    
    /// 创建事件系统能力
    #[cfg(feature = "events")]
    pub fn create_event_capability() -> Box<dyn Capability> {
        Box::new(EventCapability::new())
    }
    
    /// 创建监控指标能力
    #[cfg(feature = "metrics")]
    pub fn create_metrics_capability() -> Box<dyn Capability> {
        Box::new(MetricsCapability::new())
    }
    
    /// 创建所有可用能力
    pub fn create_all_capabilities() -> Vec<Box<dyn Capability>> {
        let mut capabilities = Vec::new();
        
        // 核心能力（总是创建）
        capabilities.push(Self::create_database_capability());
        
        // 可选能力（基于编译期feature）
        #[cfg(feature = "cache")]
        capabilities.push(Self::create_cache_capability());
        
        #[cfg(feature = "websocket")]
        capabilities.push(Self::create_websocket_capability());
        
        #[cfg(feature = "events")]
        capabilities.push(Self::create_event_capability());
        
        #[cfg(feature = "metrics")]
        capabilities.push(Self::create_metrics_capability());
        
        capabilities
    }
    
    /// 根据名称创建能力
    pub fn create_capability_by_name(name: &str) -> Option<Box<dyn Capability>> {
        match name {
            "database" => Some(Self::create_database_capability()),
            #[cfg(feature = "cache")]
            "cache" => Some(Self::create_cache_capability()),
            #[cfg(feature = "websocket")]
            "websocket" => Some(Self::create_websocket_capability()),
            #[cfg(feature = "events")]
            "events" => Some(Self::create_event_capability()),
            #[cfg(feature = "metrics")]
            "metrics" => Some(Self::create_metrics_capability()),
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_database_capability_creation() {
        let capability = DatabaseCapability::new();
        assert_eq!(capability.metadata().id.as_str(), "database");
        assert_eq!(capability.metadata().name, "Database Connection");
        assert!(capability.metadata().is_core);
    }
    
    #[test]
    fn test_capability_factory() {
        let capabilities = CapabilityFactory::create_all_capabilities();
        assert!(!capabilities.is_empty());
        
        // 数据库能力应该总是存在
        let has_database = capabilities.iter()
            .any(|cap| cap.metadata().id.as_str() == "database");
        assert!(has_database);
    }
    
    #[tokio::test]
    async fn test_capability_config() {
        let mut config = CapabilityConfig::new();
        config.set("test_key", 42).unwrap();
        
        let value: i32 = config.get("test_key").unwrap().unwrap();
        assert_eq!(value, 42);
    }
}

