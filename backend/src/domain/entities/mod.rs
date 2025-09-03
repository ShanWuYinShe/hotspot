/// 领域实体模块
/// 
/// 包含核心业务实体的定义

pub mod user;
pub mod user_location;
pub mod user_settings;
pub mod group;
pub mod group_event;
pub mod group_message;

pub mod conversation;
pub mod direct_message;
pub mod user_message_read_status;
pub mod moment;
pub mod tag;
pub mod moment_tag;
pub mod moment_like;

// 重新导出实体类型
pub use user::*;
pub use user_location::*;
pub use user_settings::*;
pub use group::*;
pub use group_event::{GroupEventRecord, GroupEventRecordType};
pub use group_message::*;

pub use conversation::*;
pub use direct_message::*;
pub use user_message_read_status::*;
pub use moment::*;
pub use tag::*;
pub use moment_tag::*;
pub use moment_like::*;

