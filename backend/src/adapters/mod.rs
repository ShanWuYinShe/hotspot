/// HotSpot 适配器层模块
/// 
/// 功能职责：
/// - 实现端口层定义的所有抽象接口
/// - 连接外部系统和第三方服务
/// - 处理数据持久化和基础设施集成
/// - 提供可插拔的技术栈实现
/// 
/// 适配器设计原则：
/// - **默认实现**：每个功能都有默认的轻量级实现
/// - **替代实现**：通过 feature flags 启用更强大的替代实现
/// - **互斥选择**：同一功能的不同实现互斥，不能同时启用
/// 
/// 适配器类型：
/// - 数据库适配器：PostgreSQL + PostGIS（核心功能，始终启用）
/// - 缓存适配器：内存缓存（默认） vs Redis缓存（替代）
/// - 任务队列适配器：内存队列（默认） vs Redis队列（替代）
/// - 通知适配器：WebSocket通知（核心功能，始终启用）

// === 核心适配器（始终启用） ===

// PostgreSQL 数据库适配器
pub mod adapters_pg;

// WebSocket 通知适配器
pub mod adapters_notify_ws;

// 复合通知适配器（支持多渠道）
pub mod adapters_notify_composite;

// 日志事件发布器
pub mod adapters_events_log;

// === 可替代的适配器实现 ===

// 缓存适配器：默认使用内存缓存
#[cfg(any(feature = "cache_memory", not(feature = "cache_redis")))]
pub mod adapters_cache_memory;

// 缓存适配器：可选的 Redis 缓存替代实现
#[cfg(feature = "cache_redis")]
pub mod adapters_cache_redis;

// 任务队列适配器：默认使用内存队列
#[cfg(any(feature = "jobs_memory", not(feature = "jobs_redis")))]
pub mod adapters_jobs_memory;

// 任务队列适配器：可选的 Redis 队列替代实现
#[cfg(feature = "jobs_redis")]
pub mod adapters_jobs_redis;

// === 可选增强适配器 ===

// 邮件通知适配器（可选增强功能）
#[cfg(feature = "email_notifications")]
pub mod adapters_notify_email;

// === 重新导出适配器类型 ===

// 核心适配器
pub use adapters_pg::*;
pub use adapters_notify_ws::*;
pub use adapters_notify_composite::*;
pub use adapters_events_log::*;

// 缓存适配器（二选一）
#[cfg(any(feature = "cache_memory", not(feature = "cache_redis")))]
pub use adapters_cache_memory::*;

#[cfg(feature = "cache_redis")]
pub use adapters_cache_redis::*;

// 任务队列适配器（二选一）
#[cfg(any(feature = "jobs_memory", not(feature = "jobs_redis")))]
pub use adapters_jobs_memory::*;

#[cfg(feature = "jobs_redis")]
pub use adapters_jobs_redis::*;

// 可选增强适配器
#[cfg(feature = "email_notifications")]
pub use adapters_notify_email::*;

