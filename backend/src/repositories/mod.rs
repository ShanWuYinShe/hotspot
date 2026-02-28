//! Domain-organized repositories

pub mod admin;
pub mod chat;
pub mod content;
pub mod feed;
pub mod group;
pub mod system;
pub mod user;

// 统一导出常用仓储模块，避免跨层直接依赖目录结构细节
pub use chat::{chat_unread_repo, private_chat_repo, private_message_repo};
pub use content::{comment_repo, moment_repo, tag_repo};
pub use feed::dismiss_repo;
pub use group::group_repo;
pub use system::{
    announcement_repo, file_repo, map_repo, notification_repo, report_repo, session_repo,
    share_repo, system_config_repo,
};
pub use user::{user_repo, verification_repo};
