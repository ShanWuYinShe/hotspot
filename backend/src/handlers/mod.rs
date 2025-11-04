pub mod auth;
pub mod user;
pub mod group;
pub mod chat;
pub mod chat_messages;
// message handler 已废弃：使用 chat_messages 替代
// pub mod message;
// conversation handlers 已废弃：使用统一的 chat_rooms 表设计
// pub mod conversation;
// pub mod conversation_settings;
pub mod system;
pub mod system_config;
pub mod moment;
pub mod comment;
pub mod media;
pub mod inbox;
pub mod report;
pub mod subscription;
pub mod leaderboard;
pub mod feed;

