pub mod user_repo;
pub mod user_location_history_repo;
pub mod session_repo;
pub mod user_relationship_repo;
pub mod group_repo;
pub mod chat_room_repo;
pub mod group_member_repo;
pub mod group_invite_repo;
pub mod group_application_repo;
pub mod room_location_history_repo;
pub mod message_repo;
pub mod message_read_repo;
// 已废弃：使用统一的 chat_rooms 表设计，不再需要单独的 conversation 表
// pub mod conversation_repo;
// pub mod conversation_settings_repo;
pub mod moment_repo;
pub mod comment_repo;
pub mod inbox_repo;
pub mod report_repo;
pub mod event_subscription_repo;
pub mod system_config_repo;
pub mod device_token_repo;

