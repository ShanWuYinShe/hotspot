/// 领域常量模块
/// 
/// 定义业务领域中的常量和枚举值

pub mod group_roles;
pub mod location_constants;

// 重新导出常用常量
pub use group_roles::*;
pub use location_constants::*;
pub use location_constants::radius::*;
