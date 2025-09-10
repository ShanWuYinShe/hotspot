/// 端口层 - 抽象接口定义
/// 
/// 架构原则：
/// - 定义系统边界
/// - 不包含实现细节
/// - 支持依赖倒置
/// - 便于测试和替换

pub mod repositories;
pub mod repository_ports;
pub mod services;
pub mod events;
pub mod cache;
pub mod cache_ports;
pub mod jobs;
pub mod job_types;
pub mod job_dispatcher_ports;
pub mod notification_ports;
pub mod notification;

// 重新导出核心接口
pub use repositories::*;
pub use services::*;
pub use events::*;
// 只导出 cache_ports 中的 trait，避免与 cache 中的类型别名冲突
pub use cache_ports::{CachePort as CachePortTrait, CacheKeyBuilder as CacheKeyBuilderTrait, CacheStats as CacheStatsTrait};
pub use cache::{CachePort, CacheKeyBuilder, CacheStats};
// 只导出 job_types 中的类型，避免与 jobs 中的重复
pub use job_types::{JobStatus as JobStatusType, JobPriority as JobPriorityType};
pub use jobs::*;
pub use notification_ports::*;
pub use notification::*;