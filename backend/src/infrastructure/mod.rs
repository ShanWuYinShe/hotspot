/// 基础设施层 - 技术实现和外部系统集成
/// 
/// 架构原则：
/// - 实现端口层定义的接口
/// - 处理技术细节
/// - 错误转换和上下文传播
/// - 资源管理

pub mod errors;
pub mod adapters;
pub mod config;
pub mod monitoring;
pub mod data_access;
// pub mod group_data_access; // 已删除重复实现，使用 adapters_pg 中的版本
pub mod query_optimizer;
pub mod event_system;
pub mod event_handlers;
pub mod postgres_event_store;
pub mod event_integration;
pub mod actor_system;
pub mod websocket_actors;
pub mod websocket_manager;
pub mod websocket_integration;
pub mod capability_system;
pub mod capabilities;
pub mod capability_manager;
pub mod application_bootstrap;
pub mod query_performance_optimizer;
pub mod enhanced_monitoring;
pub mod error_recovery;
pub mod memory_optimizer;

// 重新导出核心类型 - 使用精确导入避免冲突
pub use errors::*;
pub use adapters::*;

// 配置相关 - 使用config模块的版本
pub use config::{
    ServerConfig, DatabaseConfig, CacheConfig, AuthConfig,
    ExternalServicesConfig, LoggingConfig, FeatureConfig,
    AppConfig
};

// 监控相关 - 使用monitoring模块的版本
pub use monitoring::{
    MetricsCollector as MonitoringMetricsCollector, MetricsSnapshot,
    HistogramStats, TimerStats, PerformanceMonitor,
    HealthCheckResult, HealthChecker, DatabaseHealthChecker,
    HealthCheckManager, OverallHealthResult, HealthStatus,
    StructuredLogger
};
pub use data_access::*;
pub use query_optimizer::*;
pub use event_system::*;

// 事件处理 - 使用具体处理器
pub use event_handlers::{
    UserRegistrationHandler, GroupCreationHandler, GroupMembershipHandler,
    AuditEventHandler, StatisticsEventHandler, EventHandlerFactory,
    WebSocketService as EventWebSocketService
};

#[cfg(feature = "event_store")]
pub use postgres_event_store::*;
pub use event_integration::*;
pub use actor_system::*;
pub use websocket_actors::*;
pub use websocket_manager::*;

// WebSocket集成 - 使用具体实现
pub use websocket_integration::{
    WebSocketService, WebSocketServiceConfig, WebSocketServiceStats,
    WebSocketServiceHealth, WebSocketServiceFactory
};

pub use capability_system::*;
// 能力系统 - 使用capabilities模块的版本
pub use capabilities::{
    MetricsCollector as CapabilityMetricsCollector,
    DatabaseCapability, CacheCapability, WebSocketCapability,
    EventCapability, MetricsCapability, CapabilityFactory
};
pub use capability_manager::*;

// 应用启动 - 使用启动相关类型，避免配置冲突
pub use application_bootstrap::{
    ApplicationBootstrap, ApplicationContext, ApplicationConfig
};

// 查询性能优化
pub use query_performance_optimizer::{
    QueryPerformanceOptimizer, QueryStats, OptimizationConfig,
    OptimizationSuggestion, ConnectionPoolHealth, QueryCache
};

// 增强监控系统
pub use enhanced_monitoring::{
    EnhancedMonitoringSystem, BusinessMetrics, SystemMetrics,
    Alert, AlertManager, MonitoringReport, MonitoringConfig
};

// 错误恢复系统
pub use error_recovery::{
    ErrorRecoverySystem, ErrorTracker, ErrorEvent, ErrorType,
    ErrorStatistics, RecoverySuggestion, RecoveryConfig
};

// 内存优化系统
pub use memory_optimizer::{
    MemoryOptimizer, MemoryStats, CacheManager, CacheStats,
    MemoryOptimizationSuggestion, MemoryCleanupResult, MemoryOptimizerConfig
};
