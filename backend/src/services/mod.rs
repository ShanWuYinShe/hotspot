/// HotSpot 服务层模块
/// 
/// 提供高级业务服务和跨领域功能：
/// - 缓存服务：统一的缓存管理和策略
/// - 监控服务：性能指标收集和监控
/// - 通知服务：多渠道通知管理
/// - 任务服务：异步任务处理和调度
pub mod batch_notification_service;
pub mod cache_service;
pub mod metrics_service;
pub mod nonce_service;
pub mod simple_cache_adapter;

pub use batch_notification_service::*;
pub use cache_service::*;
pub use metrics_service::*;
pub use nonce_service::*;
pub use simple_cache_adapter::*;
