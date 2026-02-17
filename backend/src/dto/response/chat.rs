//! 私聊相关响应类型

use crate::types::enums::{CantMessageReason, MessageType, UserPresenceStatus};
use serde::Serialize;

use super::common::PaginatedData;
use super::message::PrivateMessageItem;
use super::user::UserInfo;

/// 私聊最后一条消息 - API 文档: PrivateChatLastMessage
#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub message_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_code: Option<String>,
    pub sender: UserInfo,
    pub sent_at: String,
}

/// 私聊统一数据类型 - API 文档: PrivateChatItem (同时用于列表和详情场景)
#[derive(Debug, Serialize)]
pub struct PrivateChatListItem {
    pub other_user: UserInfo,
    pub unread_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_pinned: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    // 以下字段为详情场景返回（列表场景不返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub can_message: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cant_message_reason: Option<CantMessageReason>,
}

/// 私聊项类型别名 - API 文档: PrivateChatItem
pub type PrivateChatItem = PrivateChatListItem;

/// 私聊详情打开结果 - API 文档: PrivateChatOpenResult
#[derive(Debug, Serialize)]
pub struct PrivateChatOpenResult {
    pub chat: PrivateChatItem,
    pub recent_messages: PaginatedData<PrivateMessageItem>,
}
