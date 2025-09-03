/// HTTP API 模型
/// 
/// 定义HTTP请求和响应的数据结构
/// 迁移自传统models/api模块，适配新的端口适配器架构

pub mod user;
pub mod group;
pub mod message;
pub mod moment;
pub mod common;

// 重新导出所有API模型
pub use user::*;
pub use group::*;
pub use message::*;
pub use moment::*;
pub use common::*;
