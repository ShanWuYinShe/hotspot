/// 值对象模块
/// 
/// 包含领域中的值对象定义

pub mod group_role;
pub mod location;
pub mod user_credentials;

// 重新导出值对象类型
pub use group_role::*;
pub use location::*;
pub use user_credentials::*;

