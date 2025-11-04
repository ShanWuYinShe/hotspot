// src/types/mod.rs - 自定义类型模块

pub mod enums;

// 重新导出常用枚举
pub use enums::{
    InboxCategory, InviteStatus, MemberRole, MessageType, RelationshipStatus, ReportStatus,
    RoomType,
};








