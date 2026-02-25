//! 消息相关响应类型

use crate::types::enums::MessageType;
use serde::{Deserialize, Serialize};

use super::common::id_as_string;
use super::user::UserInfo;

/// 消息回复引用 - API 文档: MessageReplyTo
///
/// 字段说明：
/// - content: 文本预览
/// - share_code: 若被回复的消息为 share 类型，则包含该 share_code
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MessageReplyTo {
    pub message_id: String,
    #[serde(with = "id_as_string")]
    pub user_id: i64,
    pub nickname: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_code: Option<String>,
}

/// 消息媒体信息 - API 文档: MessageMedia
#[derive(Debug, Serialize, Clone, Deserialize)]
pub struct MessageMedia {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
}

/// 发送私聊消息返回数据 — spec: MessageSent { message_id, sent_at }
#[derive(Debug, Serialize)]
pub struct SendPrivateMessageData {
    pub message_id: String,
    pub sent_at: String,
}

/// 私聊消息项 - API 文档: PrivateMessageItem
#[derive(Debug, Serialize)]
pub struct PrivateMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_code: Option<String>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
    /// spec: is_read: boolean
    pub is_read: bool,
}

/// 发送群聊消息返回数据 — spec: MessageSent { message_id, sent_at }
#[derive(Debug, Serialize)]
pub struct SendGroupMessageData {
    pub message_id: String,
    pub sent_at: String,
}

/// 群聊消息项 - API 文档: GroupMessageItem
#[derive(Debug, Serialize)]
pub struct GroupMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "super::common::option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_code: Option<String>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_count: Option<i32>,
}
