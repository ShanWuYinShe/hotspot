//! 私聊管理 + 私聊消息 + 群聊消息 API

use serde::Deserialize;

use crate::types::enums::SendMessageType;

#[derive(Debug, Deserialize)]
pub struct MyPrivateChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateChatSettingsRequest {
    pub target_user_id: String,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateChatOpenTheDoorRequest {
    pub target_user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct SendPrivateMessageRequest {
    pub target_user_id: String,
    pub r#type: SendMessageType,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub share_code: Option<String>,
    pub reply_to_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateMessagesHistoryRequest {
    pub target_user_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateMessageRequest {
    pub message_id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallPrivateMessageRequest {
    pub message_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkPrivateMessagesReadRequest {
    pub target_user_id: String,
    pub up_to_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SendGroupMessageRequest {
    pub group_id: String,
    pub r#type: SendMessageType,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub share_code: Option<String>,
    pub reply_to_message_id: Option<String>,
    pub mentions: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct GroupMessagesHistoryRequest {
    pub group_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_system: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupMessageRequest {
    pub message_id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallGroupMessageRequest {
    pub message_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkGroupMessagesReadRequest {
    pub group_id: String,
    pub up_to_message_id: Option<String>,
}
