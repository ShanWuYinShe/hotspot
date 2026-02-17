//! API 请求类型定义
//!
//! 所有请求类型与 API 规范 (spec/api_spec_*.mdc) 一一对应
//! 枚举类型字段使用 crate::types::enums 中定义的类型，确保类型安全

mod admin;
mod auth;
mod chat;
mod comment;
mod common;
mod group;
mod map;
mod media;
mod moment;
mod notification;
mod report;
mod search;
mod share;
mod system;
mod user;

pub use admin::*;
pub use auth::*;
pub use chat::*;
pub use comment::*;
pub use common::*;
pub use group::*;
pub use map::*;
pub use media::*;
pub use moment::*;
pub use notification::*;
pub use report::*;
pub use search::*;
pub use share::*;
pub use system::*;
pub use user::*;
