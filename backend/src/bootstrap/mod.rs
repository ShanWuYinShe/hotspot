/// HotSpot 启动层模块
/// 
/// 功能职责：
/// - 应用程序启动和初始化流程
/// - 依赖注入容器管理和组装
/// - 系统能力注册和配置
/// - 服务工厂和生命周期管理
/// 
/// 核心组件：
/// - capability_registry: 能力注册表，管理系统功能模块
/// - app_state: 应用状态管理和共享数据
/// - dependency_injection: 依赖注入容器和服务装配
/// - config_adapter: 配置适配器和环境管理
/// - service_factory: 服务工厂和实例创建
/// 
/// 设计模式：
/// - 服务定位器模式：统一的服务发现和获取
/// - 工厂模式：标准化的服务实例创建
/// - 注册表模式：集中的能力和配置管理
/// - 构建者模式：复杂对象的分步构建
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

