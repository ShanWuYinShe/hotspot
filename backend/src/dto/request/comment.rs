//! 评论与点赞 API

use serde::Deserialize;

use crate::dto::request::common::PaginationParams;

#[derive(Debug, Deserialize)]
pub struct CreateCommentRequest {
    pub moment_id: String,
    pub content: String,
    pub parent_id: Option<String>,
    pub mentions: Option<Vec<String>>, // @ 提及的用户 ID 列表（最多 10 个）
}

#[derive(Debug, Deserialize)]
pub struct DeleteCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MomentCommentsRequest {
    pub moment_id: String,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct CommentRepliesRequest {
    pub comment_id: String,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct LikeCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CommentLikesRequest {
    pub comment_id: String,
    pub pagination: Option<PaginationParams>,
}
