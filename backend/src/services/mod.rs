pub mod auth_service;
pub mod user_service;
pub mod group_service;
pub mod chat_room_service;
pub mod chat_messages_service;
// message_service 已废弃：使用 chat_messages_service 替代
// pub mod message_service;
pub mod moment_service;
pub mod comment_service;
pub mod media_service;
pub mod inbox_service;
pub mod report_service;
// conversation_settings_service 已废弃：使用统一的 chat_rooms 表设计
// pub mod conversation_settings_service;
pub mod subscription_service;
pub mod leaderboard_service;
pub mod feed_service;
pub mod system_config_service;
pub mod presence_service;

