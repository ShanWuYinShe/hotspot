//! Response DTO 定义
//!
//! 所有响应类型严格遵循 API 规范 (spec/*.mdc)
//! 使用强类型枚举，无向后兼容代码
//!
//! 按领域拆分为子模块，通过 pub use 统一导出以保持 `dto::response::*` 的兼容性。

pub mod admin;
pub mod auth;
pub mod chat;
pub mod comment;
pub mod common;
pub mod config;
pub mod feed;
pub mod group;
pub mod group_member;
pub mod map;
pub mod media;
pub mod message;
pub mod moment;
pub mod notification;
pub mod report;
pub mod share;
pub mod system;
pub mod tag;
pub mod user;

// Re-export all public types for backward compatibility
pub use admin::*;
pub use auth::*;
pub use chat::*;
pub use comment::*;
pub use common::*;
pub use config::*;
pub use feed::*;
pub use group::*;
pub use group_member::*;
pub use map::*;
pub use media::*;
pub use message::*;
pub use moment::*;
pub use notification::*;
pub use report::*;
pub use share::*;
pub use system::*;
pub use tag::*;
pub use user::*;
