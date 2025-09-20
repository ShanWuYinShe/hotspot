/// 领域实体模块 - 重新导出所有实体类型
/// 
/// 实体特征：
/// - 具有唯一标识
/// - 生命周期内身份不变
/// - 可变状态
/// - 业务行为

// 用户相关实体
pub mod user;
pub use user::*;

// 群组相关实体
pub mod group;
pub use group::*;

// 统一消息实体
pub mod message;
pub use message::*;

// 瞬间相关实体
pub mod moment;
pub mod moment_like;
pub mod moment_tag;
pub mod tag;
pub use moment::*;
pub use moment_like::*;
pub use moment_tag::*;
pub use tag::*;

// 好友相关实体
pub mod friend;
pub use friend::*;

// 评论相关实体
pub mod comment;
pub use comment::*;

// 屏蔽相关实体
pub mod block;
pub use block::*;

// 群组事件实体
pub mod group_event;
pub use group_event::*;

// 用户位置实体（注释掉以避免与user.rs中的UserLocation冲突）
// pub mod user_location;
// pub use user_location::*;

// 用户设置实体
pub mod user_settings;
pub use user_settings::*;

// 用户消息读取状态实体
pub mod user_message_read_status;
pub use user_message_read_status::*;
