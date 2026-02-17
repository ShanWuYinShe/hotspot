//! User domain repositories

pub mod blocks_repo;
pub mod follows_repo;
pub mod friend_remarks_repo;
pub mod location_history_repo;
pub mod online_status_repo;
pub mod settings_repo;
pub mod user_repo;
pub mod verification_repo;

// Backward compatibility: re-export modules with original names
pub use blocks_repo as user_blocks_repo;
pub use follows_repo as user_follows_repo;
pub use friend_remarks_repo as user_friend_remarks_repo;
pub use location_history_repo as user_location_history_repo;
pub use online_status_repo as user_online_status_repo;
pub use settings_repo as user_settings_repo;
