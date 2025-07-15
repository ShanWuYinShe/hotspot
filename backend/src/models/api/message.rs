use crate::utils::response::{PaginatedResponse, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

// ============ 群组消息相关请求/响应模型 ============

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendGroupMessageRequest {
    pub group_id: String,
    pub content: String,
    pub message_type: String, // 默认为 "text"，可以是 "image" 等
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendGroupMessageResponse {
    pub group_id: String,
    pub message_id: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct GroupMessageHistoryRequest {
    pub group_id: String,
    #[serde(flatten)]
    pub pagination: Pagination,
}

pub type GroupMessageHistoryResponse = PaginatedResponse<GroupMessageHistoryItem>;

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteGroupMessageRequest {
    pub message_id: String,
}

// ============ 私聊消息相关请求/响应模型 ============

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendDirectMessageRequest {
    pub recipient_id: String,
    pub content: String,
    pub message_type: String, // 默认为 "text"，可以是 "image" 等
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SendDirectMessageResponse {
    pub message_id: String,
    pub conversation_id: String,
    pub recipient_id: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DirectMessageHistoryRequest {
    pub conversation_id: Option<String>,
    pub user_id: Option<i64>,
    #[serde(flatten)]
    pub pagination: Pagination,
}

pub type DirectMessageHistoryResponse = PaginatedResponse<DirectMessageHistoryItem>;

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteDirectMessageRequest {
    pub message_id: String,
}

/// 标记消息已读请求
/// - conversation_id: 必需，会话ID
/// - message_id: 可选，指定消息ID。如果提供，则只标记该消息及之前的所有消息为已读
#[derive(Debug, Serialize, Deserialize)]
pub struct MarkReadRequest {
    /// 会话ID，必需
    pub conversation_id: String,
    /// 消息ID，可选。如果提供，则只标记该消息及之前的所有消息为已读
    pub message_id: Option<String>,
}

/// 标记消息已读响应
#[derive(Debug, Serialize, Deserialize)]
pub struct MarkReadResponse {
    /// 操作是否成功
    pub success: bool,
    /// 标记为已读的消息数量
    pub marked_count: Option<i64>,
}

/// 获取群组未读消息数量请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GetGroupUnreadCountRequest {
    pub group_id: String,
}

/// 获取未读消息数量响应
#[derive(Debug, Serialize, Deserialize)]
pub struct UnreadCountResponse {
    pub unread_count: i64,
}

// ============ 通用响应模型 ============

#[derive(Debug, Serialize, Deserialize)]
pub struct DirectMessageHistoryItem {
    pub message_id: String,
    pub conversation_id: String,
    pub sender_id: String,
    pub sender_name: String,
    pub recipient_id: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
    pub read_at: Option<DateTime<Utc>>,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct GroupMessageHistoryItem {
    pub message_id: String,
    pub group_id: String,
    pub sender_id: String,
    pub sender_name: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteMessageResponse {
    pub success: bool,
}

// ============ 会话列表相关模型 ============

#[derive(Debug, Serialize, Deserialize)]
pub struct ConversationsRequest {
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ConversationItem {
    pub conversation_id: String,
    pub peer_id: i64,
    pub peer_name: String,
    pub last_message_preview: String,
    pub last_message_time: DateTime<Utc>,
    pub unread_count: i64,
}

pub type ConversationsResponse = PaginatedResponse<ConversationItem>;
