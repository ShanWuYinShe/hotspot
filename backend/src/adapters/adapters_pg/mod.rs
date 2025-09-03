/// PostgreSQL适配器模块
/// 
/// 实现基于PostgreSQL和sqlx的数据访问适配器

pub mod user_repository;
pub mod group_repository;
pub mod message_repository;
pub mod moment_repository;
pub mod friend_repository;
pub mod password_service;

// 重新导出适配器类型
pub use user_repository::*;
pub use group_repository::*;
pub use message_repository::*;
pub use moment_repository::*;
pub use friend_repository::*;
pub use password_service::*;

