//! 评论相关响应类型

use serde::Serialize;

use super::common::option_vec_id_as_string;
use super::user::UserInfo;

/// 评论列表项 - API 文档: CommentListItem
#[derive(Debug, Serialize)]
pub struct CommentListItem {
    pub comment_id: String,
    pub user: UserInfo,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub replies_count: i32,
    /// @auth-only：登录态返回当前用户是否已点赞
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_liked: Option<bool>,
}

/// 评论点赞项 - API 文档: CommentLikeItem
#[derive(Debug, Serialize)]
pub struct CommentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}
