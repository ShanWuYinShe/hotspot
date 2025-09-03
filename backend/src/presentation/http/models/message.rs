/// 消息相关HTTP模型
/// 
/// 迁移自models/api/message.rs，适配新架构

use super::{PaginatedResponse, LocationInfo, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use crate::utils::JsSafeI64;

// ============ 群组消息相关请求/响应模型 ============

/// 发送群组消息请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendGroupMessageRequest {
    pub group_id: String,
    pub content: String,
    pub message_type: String, // 默认为 "text"，可以是 "image" 等
    pub latitude: f64,
    pub longitude: f64,
}

/// 发送群组消息响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendGroupMessageResponse {
    pub group_id: String,
    pub message_id: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
}

/// 群组消息历史请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupMessageHistoryRequest {
    pub group_id: String,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 群组消息历史项
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupMessageHistoryItem {
    pub id: JsSafeI64,
    pub message_id: String,
    pub group_id: String,
    pub sender_id: JsSafeI64,
    pub sender_nickname: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
    #[serde(flatten)]
    pub location: LocationInfo,
}

/// 群组消息历史响应
pub type GroupMessageHistoryResponse = PaginatedResponse<GroupMessageHistoryItem>;

/// 删除群组消息请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteGroupMessageRequest {
    pub message_id: String,
}

// ============ 私聊消息相关请求/响应模型 ============

/// 发送私聊消息请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendDirectMessageRequest {
    pub recipient_id: String,
    pub content: String,
    pub message_type: String, // 默认为 "text"，可以是 "image" 等
    pub latitude: f64,
    pub longitude: f64,
}

/// 发送私聊消息响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SendDirectMessageResponse {
    pub message_id: String,
    pub conversation_id: String,
    pub recipient_id: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
}

/// 私聊消息历史请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DirectMessageHistoryRequest {
    pub conversation_id: Option<String>,
    pub user_id: Option<JsSafeI64>,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 私聊消息历史项
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DirectMessageHistoryItem {
    pub id: JsSafeI64,
    pub message_id: String,
    pub conversation_id: String,
    pub sender_id: JsSafeI64,
    pub recipient_id: JsSafeI64,
    pub sender_nickname: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
    pub read_at: Option<DateTime<Utc>>,
    #[serde(flatten)]
    pub location: LocationInfo,
}

/// 私聊消息历史响应
pub type DirectMessageHistoryResponse = PaginatedResponse<DirectMessageHistoryItem>;

/// 删除私聊消息请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteDirectMessageRequest {
    pub message_id: String,
}

// ============ 通用消息模型 ============

/// 消息信息
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MessageInfo {
    pub id: JsSafeI64,
    pub message_id: String,
    pub sender_id: JsSafeI64,
    pub sender_nickname: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: DateTime<Utc>,
    pub read_at: Option<DateTime<Utc>>,
    #[serde(flatten)]
    pub location: LocationInfo,
}

/// 会话信息
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct ConversationInfo {
    pub conversation_id: String,
    pub conversation_type: String, // "direct" 或 "group"
    pub participant_id: Option<i64>, // 私聊对方ID
    pub group_id: Option<String>,    // 群组ID
    pub last_message: Option<MessageInfo>,
    pub unread_count: i64,
    pub updated_at: DateTime<Utc>,
}

/// 会话列表请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct ConversationListRequest {
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 会话列表响应
pub type ConversationListResponse = PaginatedResponse<ConversationInfo>;

/// 标记消息已读请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MarkMessageReadRequest {
    pub message_id: String,
}

/// 标记会话已读请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MarkConversationReadRequest {
    pub conversation_id: String,
}
