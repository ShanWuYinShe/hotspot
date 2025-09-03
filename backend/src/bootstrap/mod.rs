/// 启动层模块
/// 
/// 负责应用启动、依赖注入、能力注册和配置管理
/// 实现CapabilityRegistry模式，统一装配系统组件

pub mod capability_registry;
pub mod app_state;
pub mod dependency_injection;
pub mod config_adapter;
pub mod service_factory;
pub mod email_config;

// 重新导出启动层类型
pub use capability_registry::*;
pub use app_state::*;
pub use dependency_injection::*;
pub use config_adapter::*;
pub use service_factory::*;
pub use email_config::*;

