/// 适配器层模块
/// 
/// 实现端口接口的具体适配器，连接外部系统和技术栈
/// 包括数据库、缓存、消息队列、通知等基础设施的实现

// PostgreSQL 适配器（默认）
#[cfg(feature = "postgres")]
pub mod adapters_pg;

// 内存缓存适配器（默认）
#[cfg(any(feature = "cache_in_memory", not(feature = "cache_redis")))]
pub mod adapters_cache_memory;

// Redis 缓存适配器（可选）
#[cfg(feature = "cache_redis")]
pub mod adapters_cache_redis;

// 轻量级任务队列适配器（默认）
#[cfg(any(feature = "jobs_lite", not(feature = "jobs_redis")))]
pub mod adapters_jobs_lite;

// Redis 任务队列适配器（可选）
#[cfg(feature = "jobs_redis")]
pub mod adapters_jobs_redis;

// WebSocket 通知适配器（默认且唯一）
#[cfg(feature = "ws")]
pub mod adapters_notify_ws;

// 邮件通知适配器（可选）
#[cfg(feature = "email_channel")]
pub mod adapters_notify_email;

// 复合通知适配器（支持多渠道）
pub mod adapters_notify_composite;

// 日志事件发布器（默认占位实现）
pub mod adapters_events_log;

// 重新导出适配器类型
#[cfg(feature = "postgres")]
pub use adapters_pg::*;

#[cfg(any(feature = "cache_in_memory", not(feature = "cache_redis")))]
pub use adapters_cache_memory::*;

#[cfg(feature = "cache_redis")]
pub use adapters_cache_redis::*;

#[cfg(any(feature = "jobs_lite", not(feature = "jobs_redis")))]
pub use adapters_jobs_lite::*;

#[cfg(feature = "jobs_redis")]
pub use adapters_jobs_redis::*;

#[cfg(feature = "ws")]
pub use adapters_notify_ws::*;

#[cfg(feature = "email_channel")]
pub use adapters_notify_email::*;

pub use adapters_notify_composite::*;

pub use adapters_events_log::*;

