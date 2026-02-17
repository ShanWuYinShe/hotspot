//! Group domain repositories

pub mod announcement_repo;
pub mod application_repo;
pub mod bans_repo;
pub mod group_repo;
pub mod invite_repo;
pub mod member_history_repo;
pub mod member_repo;
pub mod message_repo;
pub mod settings_repo;

// Backward compatibility: re-export modules with original names
pub use announcement_repo as group_announcement_repo;
pub use application_repo as group_application_repo;
pub use bans_repo as group_bans_repo;
pub use invite_repo as group_invite_repo;
pub use member_history_repo as group_member_history_repo;
pub use member_repo as group_member_repo;
pub use message_repo as group_message_repo;
pub use settings_repo as group_settings_repo;
