/// 值对象模块 - 重新导出所有值对象类型

// 坐标相关 - 使用更完整的坐标系统
pub mod coordinates;
pub use coordinates::*;

// 位置相关 - 仅导出Location，避免Coordinates冲突
pub mod location;
pub use location::Location;

// 用户相关
pub mod user_id;
pub use user_id::*;

pub mod user_credentials;
pub use user_credentials::{LoginId, Password};
// 从user_credentials导出Nickname，避免与content中的Nickname冲突
pub use user_credentials::Nickname as UserNickname;

// ID类型
pub mod group_id;
pub use group_id::*;

pub mod message_id;
pub use message_id::*;

pub mod moment_id;
pub use moment_id::*;

// 群组相关
pub mod group_role;
pub use group_role::*;

pub mod group_type;
pub use group_type::*;

// 内容相关
pub mod content;
// 从content导出所有类型，包括Nickname（作为主要的Nickname类型）
pub use content::*;

// 分页相关
pub mod pagination;
pub use pagination::*;
