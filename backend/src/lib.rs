#![allow(async_fn_in_trait)]
//! HotSpot 后端库

// 导出模块供其他组件使用
pub mod cache;
pub mod config;
pub mod controllers;
#[cfg(feature = "jobs")]
pub mod jobs;
pub mod middleware;
pub mod models;
pub mod repositories;
pub mod routes;
pub mod services;
pub mod utils;
pub mod validators;
pub mod ws;

/// 统一导出常用类型，便于外部导入
pub mod prelude {
    pub use crate::utils::error::AppError;
    pub use crate::utils::metrics::AppMetrics;
    pub use crate::middleware::auth::AppState;
    pub use crate::ws::message_types::{RoomId, ServerMessage, ClientMessage};
    pub use crate::ws::manager::WebSocketManager;
    pub use crate::ws::message_router::MessageRouter;
    pub use crate::ws::room_manager::RoomManager;
}

// 重新导出主要组件
pub use config::db::Database;
pub use services::GroupService;
pub use services::MessageService;
pub use services::UserService;
pub use services::MomentService;
