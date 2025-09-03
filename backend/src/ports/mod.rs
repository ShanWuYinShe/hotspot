/// 端口层 - 抽象接口定义
/// 
/// 定义应用层依赖的所有外部接口，遵循依赖倒置原则
/// 这些接口将由适配器层具体实现

pub mod repository_ports;
pub mod repositories;
pub mod cache_ports;
pub mod job_dispatcher_ports;
pub mod notification_ports;
pub mod event_publisher_ports;

// 重新导出主要的端口接口
pub use repository_ports::*;
pub use repositories::*;
pub use cache_ports::*;
pub use job_dispatcher_ports::*;
pub use notification_ports::*;
pub use event_publisher_ports::*;


