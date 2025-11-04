use serde::{Deserialize, Serialize};

/// 排行榜条目
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaderboardEntry {
    pub rank: i32,              // 排名
    pub user_id: i64,           // 用户ID
    pub username: String,       // 用户名
    pub nickname: String,       // 昵称
    pub avatar_url: Option<String>, // 头像URL
    pub score: i64,             // 分数
}

/// 排行榜响应
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaderboardResponse {
    pub entries: Vec<LeaderboardEntry>,
    pub has_more: bool,
    pub total: i64,
}











