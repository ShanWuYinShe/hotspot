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
pub mod group_data_access;
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

// 重新导出核心类型
pub use errors::*;
pub use adapters::*;
pub use config::*;
pub use monitoring::*;
pub use data_access::*;
pub use group_data_access::*;
pub use query_optimizer::*;
pub use event_system::*;
pub use event_handlers::*;
pub use postgres_event_store::*;
pub use event_integration::*;
pub use actor_system::*;
pub use websocket_actors::*;
pub use websocket_manager::*;
pub use websocket_integration::*;
pub use capability_system::*;
pub use capabilities::*;
pub use capability_manager::*;
pub use application_bootstrap::*;
