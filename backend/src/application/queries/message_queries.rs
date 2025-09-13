/// 消息查询定义 - 消息相关查询操作
/// 
/// 查询特征：
/// - 只读操作
/// - 返回视图模型
/// - 支持分页
/// - 性能优化

use serde::{Deserialize, Serialize};
use crate::domain::{UserId, GroupId};
use crate::application::queries::{Query, PaginationQuery};

// === 消息查询 ===

/// 获取对话消息查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetConversationMessagesQuery {
    /// 对话ID
    pub conversation_id: String,
    /// 分页参数
    pub pagination: PaginationQuery,
}

impl Query for GetConversationMessagesQuery {
    type Result = crate::application::handlers::PaginatedResult<MessageView>;
}

/// 获取群组消息查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetGroupMessagesQuery {
    /// 群组ID
    pub group_id: GroupId,
    /// 分页参数
    pub pagination: PaginationQuery,
}

impl Query for GetGroupMessagesQuery {
    type Result = crate::application::handlers::PaginatedResult<MessageView>;
}

/// 搜索消息查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchMessagesQuery {
    /// 搜索关键词
    pub keyword: String,
    /// 用户ID（可选，限制搜索范围）
    pub user_id: Option<UserId>,
    /// 群组ID（可选，限制搜索范围）
    pub group_id: Option<GroupId>,
    /// 分页参数
    pub pagination: PaginationQuery,
}

impl Query for SearchMessagesQuery {
    type Result = crate::application::handlers::PaginatedResult<MessageView>;
}

/// 获取未读消息数量查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUnreadMessageCountQuery {
    /// 用户ID
    pub user_id: UserId,
}

impl Query for GetUnreadMessageCountQuery {
    type Result = UnreadMessageCount;
}

// === 查询结果视图 ===

/// 消息视图
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageView {
    /// 消息ID
    pub message_id: String,
    /// 发送者ID
    pub sender_id: i64,
    /// 发送者昵称
    pub sender_nickname: String,
    /// 消息内容
    pub content: String,
    /// 消息类型
    pub message_type: String,
    /// 创建时间
    pub created_at: chrono::DateTime<chrono::Utc>,
    /// 是否已读
    pub is_read: bool,
}

/// 未读消息数量
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnreadMessageCount {
    /// 总未读数量
    pub total_count: u64,
    /// 按对话分组的未读数量
    pub by_conversation: std::collections::HashMap<String, u64>,
}

/// 对话摘要（向后兼容）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConversationSummary {
    /// 对话ID
    pub conversation_id: String,
    /// 参与者ID列表
    pub participant_ids: Vec<i64>,
    /// 最后消息内容
    pub last_message_content: Option<String>,
    /// 最后消息时间
    pub last_message_at: Option<chrono::DateTime<chrono::Utc>>,
    /// 未读消息数量
    pub unread_count: u64,
}