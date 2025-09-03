/// 应用服务层模块
/// 
/// 实现用例服务、命令查询模型（CQRS）和应用级业务逻辑协调
/// 连接领域层和基础设施层，不包含业务规则，只负责协调

pub mod commands;
pub mod queries;
pub mod services;

// 重新导出应用服务类型
pub use commands::*;
pub use queries::*;
pub use services::*;

