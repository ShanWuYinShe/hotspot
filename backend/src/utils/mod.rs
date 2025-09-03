pub mod jwt;
pub mod response;
pub mod error;
pub mod error_monitor;
pub mod performance_monitor;
pub mod business_metrics;
pub mod db_optimizer;
pub mod advanced_geo_optimizer;
pub mod database_optimizer_tool;
#[cfg(feature = "jobs")]
pub mod job_queue;
pub mod metrics;
#[cfg(feature = "jobs")]
// pub mod scheduler; // 已删除，调度功能迁移到新架构
pub mod result_ext;
pub mod validation_pipeline;
pub mod query_monitor;
// pub mod response_mapper; // 已删除，功能迁移到presentation层
pub mod constants;
pub mod user_id_partition;
pub mod permission;
#[allow(dead_code)]
pub mod audit;
#[allow(dead_code)]
pub mod events;
pub mod types;
pub mod js_safe_types;
pub mod conversion_helpers;
#[allow(dead_code)]
pub mod capabilities;

// 重新导出常用结构
// pub use response_mapper::ResponseMapper; // 已删除，功能迁移到presentation层
pub use user_id_partition::{UserIdPartition, UserType};
#[allow(unused_imports)]
pub use types::{UserId, GroupId, ConversationId, MessageId};

// 重新导出监控模块
pub use error_monitor;
pub use performance_monitor;
pub use business_metrics;
