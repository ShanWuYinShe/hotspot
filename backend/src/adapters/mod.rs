/// HotSpot 适配器层模块
/// 
/// 功能职责：
/// - 实现端口层定义的所有抽象接口
/// - 连接外部系统和第三方服务
/// - 处理数据持久化和基础设施集成
/// - 提供可插拔的技术栈实现
/// 
/// 适配器类型：
/// - 数据库适配器：PostgreSQL + PostGIS（地理位置查询）
/// - 缓存适配器：内存缓存（默认）、Redis缓存（可选）
/// - 任务队列适配器：轻量级队列（默认）、Redis队列（可选）
/// - 通知适配器：WebSocket通知（默认）
/// 
/// 设计特点：
/// - 通过 feature flags 控制适配器启用
/// - 支持运行时配置切换不同实现
/// - 完全实现端口接口契约
/// - 独立的错误处理和日志记录
// PostgreSQL 适配器（默认）
#[cfg(feature = "postgres")]
pub mod adapters_pg;

// 内存缓存适配器（默认）
#[cfg(any(feature = "cache_in_memory", not(feature = "cache_redis")))]
// pub mod adapters_cache_memory;  // 暂时禁用，trait实现不完整

// Redis 缓存适配器（可选）
#[cfg(feature = "cache_redis")]
pub mod adapters_cache_redis;


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
// pub use adapters_cache_memory::*;  // 暂时禁用

#[cfg(feature = "cache_redis")]
pub use adapters_cache_redis::*;


#[cfg(feature = "jobs_redis")]
pub use adapters_jobs_redis::*;

#[cfg(feature = "ws")]
pub use adapters_notify_ws::*;

#[cfg(feature = "email_channel")]
pub use adapters_notify_email::*;

pub use adapters_notify_composite::*;

pub use adapters_events_log::*;

