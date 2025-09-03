/// 消息相关查询定义
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListDirectMessagesQuery {
    pub user_id: i64, // 当前用户ID
    pub peer_id: i64, // 会话对端用户ID（简化）
    pub conversation_id: Option<String>, // 可选的会话ID，优先使用
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListConversationsQuery {
    pub user_id: i64,
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConversationSummary {
    pub conversation_id: String,
    pub peer_id: i64,
    pub last_message_preview: Option<String>,
    pub last_message_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListGroupMessagesQuery {
    pub group_id: i64,
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetDirectUnreadCountQuery {
    pub user_id: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetGroupUnreadCountQuery {
    pub user_id: i64,
    pub group_id: i64,
}
