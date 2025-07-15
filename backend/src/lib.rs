//! HotSpot 后端库

// 导出模块供其他组件使用
pub mod cache;
pub mod config;
pub mod controllers;
pub mod jobs;
pub mod middleware;
pub mod models;
pub mod repositories;
pub mod routes;
pub mod services;
pub mod utils;
pub mod validators;
pub mod ws;

// 重新导出主要组件
pub use config::db::Database;
pub use services::CheckinService;
pub use services::GroupService;
pub use services::MessageService;
pub use services::UserService;
