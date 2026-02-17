//! Domain-organized repositories

pub mod admin;
pub mod chat;
pub mod content;
pub mod feed;
pub mod group;
pub mod system;
pub mod user;

// Re-export all modules for backward compatibility (explicit to ensure glob works)
pub use admin::{admin_audit_repo, admin_rbac_repo};
pub use chat::{chat_unread_repo, private_chat_repo, private_message_repo};
pub use content::{comment_repo, moment_repo, tag_repo};
pub use feed::dismiss_repo;
pub use group::{
    group_announcement_repo, group_application_repo, group_bans_repo, group_invite_repo,
    group_member_history_repo, group_member_repo, group_message_repo, group_repo,
    group_settings_repo,
};
pub use system::{
    announcement_repo, file_repo, map_repo, notification_repo, report_repo, session_repo,
    share_repo, system_config_repo,
};
pub use user::{
    user_blocks_repo, user_follows_repo, user_friend_remarks_repo, user_location_history_repo,
    user_online_status_repo, user_repo, user_settings_repo, verification_repo,
};
