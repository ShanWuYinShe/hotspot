/// HTTP验证器模块
/// 
/// 迁移自传统validators模块，适配新的端口适配器架构
/// 提供HTTP请求参数的验证功能
pub mod common_validator;
pub mod user_validator;
pub mod group_validator;
pub mod message_validator;
pub mod moment_validator;

// 重新导出验证器
pub use common_validator::*;
pub use user_validator::*;
pub use group_validator::*;
pub use message_validator::*;
pub use moment_validator::*;
