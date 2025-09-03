#![allow(async_fn_in_trait)]
//! # HotSpot 后端库
//!
//! 基于 Rust 和 PostgreSQL 构建的地理位置社交平台后端服务
//!
//! ## 🏗️ 架构特性
//!
//! - **端口适配器架构**: 采用Hexagonal架构模式，清晰分离领域、应用、基础设施
//! - **领域驱动设计**: 核心业务逻辑封装在领域层，保持业务纯粹性
//! - **CQRS模式**: 命令查询分离，优化读写性能
//! - **实时通信**: 基于 WebSocket 的实时消息和通知系统
//! - **地理位置**: 集成 PostGIS 的高性能地理位置查询
//! - **异步任务**: 基于 tokio 的异步任务处理框架
//! - **缓存系统**: 多层缓存策略提升性能
//! - **监控指标**: 完整的性能监控和指标收集
//! - **类型安全**: 强类型系统和泛型依赖注入
//! - **混合配置**: 支持传统配置和新架构配置的桥接
//!
//! ## 📦 核心模块
//!
//! ### 新架构模块
//! - [`domain`] - 领域层：实体、值对象、领域服务、领域事件
//! - [`application`] - 应用层：用例服务、命令查询处理器、应用服务
//! - [`ports`] - 端口层：抽象接口定义，定义系统边界
//! - [`adapters`] - 适配器层：具体实现（数据库、缓存、消息队列等）
//! - [`presentation`] - 表现层：HTTP API、WebSocket处理、控制器
//! - [`bootstrap`] - 启动层：依赖注入、应用组装、配置管理
//!
//! ### 已迁移模块
//! - 缓存管理 - 已迁移到新架构的应用服务层
//! - 配置管理 - 已迁移到新架构的启动层
//! - [`utils`] - 通用工具和辅助函数
//!
//! ## 🚀 快速开始
//!
//! ```rust,no_run
//! use backend::prelude::*;
//! use backend::bootstrap::HybridConfigManager;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     // 加载配置
//!     let config_manager = HybridConfigManager::from_env().await?;
//!     
//!     // 检查健康状态
//!     config_manager.check_health().await?;
//!     
//!     // 启动应用...
//!     Ok(())
//! }
//! ```
//!
//! ## 🔧 功能特性
//!
//! ### 用户管理
//! - 用户注册、登录、临时用户
//! - 密码验证、JWT认证
//! - 用户资料管理、位置更新
//!
//! ### 群组功能
//! - 群组创建、管理、成员管理
//! - 地理位置群组、附近群组搜索
//! - 群组权限控制
//!
//! ### 消息系统
//! - 实时消息发送、接收
//! - WebSocket连接管理
//! - 消息历史、分页查询
//!
//! ### 瞬间分享
//! - 生活瞬间发布、查看
//! - 地理位置标记
//! - 附近瞬间发现
//!
//! ### 异步任务
//! - 欢迎邮件发送
//! - 位置索引更新
//! - 通知推送处理
//!
//! ## 📊 监控与健康检查
//!
//! 系统提供多层次的健康检查：
//! - `/health` - 基础健康检查
//! - `/health/detailed` - 详细组件状态检查
//! - `/ready` - 就绪状态检查
//! - `/alive` - 存活状态检查

// 新架构模块
pub mod adapters;
pub mod application;
pub mod bootstrap;
pub mod domain;
pub mod ports;
pub mod presentation;

// 架构迁移完成：
// - 旧的services -> application/services
// - 旧的repositories -> adapters
// - 旧的models -> domain/entities + presentation/http/models  
// - 旧的controllers -> presentation/http
// - 旧的middleware -> presentation/http/middleware
// - 旧的validators -> presentation/http/validators
// - 旧的jobs -> application/services/job_application_service
// - 旧的cache -> application/services/cache_application_service
// - 旧的config -> bootstrap/config_adapter
// WebSocket系统核心模块
pub mod ws;
// utils模块，提供通用工具函数
pub mod utils {
    // 核心工具模块
    pub mod user_id_partition;
    pub mod types;
    pub mod constants;
    
    // 基础设施工具（按需启用）
    #[cfg(feature = "audit")]
    pub mod audit;
    pub mod events;
    pub mod capabilities;
    pub mod validation_pipeline; // 已修复类型转换问题
    pub mod query_monitor;
    pub mod permission;
    
    // 监控和指标
    pub mod metrics;
    pub mod error;
    pub mod error_monitor;
    pub mod performance_monitor;
    pub mod db_optimizer;
    
    // JWT工具
    pub mod jwt;
    
    // 响应映射器
    // response_mapper已删除，功能迁移到presentation层
    pub mod response;
    
    // 结果扩展
    pub mod result_ext;
    
    // 任务队列相关（按需启用）
    #[cfg(feature = "jobs")]
    pub mod job_queue;
    // #[cfg(feature = "jobs")]
    // pub mod scheduler; // 已删除，调度功能迁移到新架构
    
    // JavaScript兼容性类型
    pub mod js_safe_types;
    pub mod conversion_helpers;
    
    // 重新导出常用类型
    pub use user_id_partition::{UserIdPartition, UserType};
    pub use types::{UserId, GroupId, ConversationId, MessageId};
    // ResponseMapper已删除，功能迁移到presentation层
    pub use js_safe_types::JsSafeI64;
}
// 验证器已迁移到presentation/http/validators
// 已迁移到新架构的presentation/ws层

#[cfg(test)]
pub mod tests;

/// 统一导出常用类型，便于外部导入
pub mod prelude {
    // 领域层核心类型
    pub use crate::domain::{
        Coordinates, DomainError, DomainResult, Group, GroupEvent, GroupRole, Location,
        MessageEvent, Moment, MomentEvent, User, UserEvent,
    };

    // 应用层核心类型
    pub use crate::application::{
        CommandHandler, CreateTempUserCommand, GetUserByIdQuery, GetUserProfileQuery,
        PaginatedResult, PaginationQuery, QueryHandler, RegisterUserCommand,
        UserApplicationService, GroupApplicationService, MessageApplicationService,
        MomentApplicationService, NotificationApplicationService,
        CacheApplicationService, JobApplicationService,
    };

    // 端口层接口
    pub use crate::ports::{
        CachePort, EventPublisherPort, JobDispatcherPort, NotificationTransportPort,
        PasswordServicePort, UserRepositoryPort, GroupRepositoryPort,
        MomentRepositoryPort, CacheStats, Job, JobId, JobStatus, QueueStats,
    };

    // 适配器实现
    pub use crate::adapters::{
        Argon2PasswordService, PgUserRepository,
        PgGroupRepository, PgMessageRepository, PgMomentRepository,
        LogEventPublisher,
    };
    
    #[cfg(feature = "jobs_lite")]
    pub use crate::adapters::LiteJobDispatcher;
    
    #[cfg(feature = "cache_in_memory")]
    pub use crate::adapters::MemoryCache;
    
    #[cfg(feature = "ws")]
    pub use crate::adapters::WebSocketNotificationTransport;
    
    // 缓存系统
    // 缓存功能已迁移到新架构
    // pub use crate::application::services::cache_application_service::CacheApplicationService;

    // 表现层类型
    pub use crate::presentation::{
        HealthController, UserController, GroupController, MessageController, MomentController,
        WSError, WSMessageType, WebSocketManager, AdminController, EnhancedHealthController,
    };
    
    // HTTP相关类型
    pub use crate::presentation::http::{
        create_routes,
    };
    pub use crate::presentation::http::auth::JwtConfig;

    // 启动层
    pub use crate::bootstrap::{
        AppConfig, CapabilityRegistry, HexagonalAppState, AppStateBuilder,
        HybridConfigManager,
        ServerConfig, Environment, Database, DbMetrics,
        ServiceFactory, ServiceContainer, ServiceContainerExt,
    };

    // 工具类型
    pub use crate::utils::{
        UserIdPartition, UserType, UserId, GroupId, ConversationId, MessageId,
        // ResponseMapper已迁移到presentation层
    };
    
    // 错误处理和监控
    pub use crate::utils::error::{AppError, ErrorContext};
    pub use crate::utils::error_monitor::ErrorMonitor;
    pub use crate::utils::performance_monitor::PerformanceMonitor;
    
    // 配置和常量
    pub use crate::domain::constants::{
        DEFAULT_LATITUDE, DEFAULT_LONGITUDE, DEFAULT_LOCATION_NAME,
        DEFAULT_NEARBY_RADIUS_KM, MEMBER, ADMIN, OWNER,
    };

    // 新架构WebSocket系统
    pub use crate::presentation::ws::{
        WebSocketManager as NewWebSocketManager,
        WebSocketConnectionHandler,
    };
    
    // 通知系统（新架构）- 补充导出
    pub use crate::ports::{
        NotificationPolicyPort, Notification, 
        NotificationPolicy, NotificationContext, NotificationType, NotificationChannel,
        TransportStats, PolicyStats, LocationContext,
    };
    #[cfg(feature = "ws")]
    pub use crate::adapters::BasicNotificationPolicy;
    
    // 增强健康检查系统（暂时禁用，等待services模块适配）
    // pub use crate::services::health::{
    //     EnhancedHealthService, EnhancedHealthResponse, HealthStatus,
    // };
    
    // WebSocket核心组件（暂时禁用，等待ws模块适配）
    // pub use crate::ws::{
    //     MessageRouter, ConnectionManager, RoomManager,
    //     WebSocketStats, ConnectionStats, RoomStats,
    // };
}

// 新架构主要组件
pub use crate::bootstrap::config_adapter::{Database, HybridConfigManager};
