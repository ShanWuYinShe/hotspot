use redis::Client as RedisClient;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AddCommentRequest, DeleteCommentRequest, LikeCommentRequest,
            UnlikeCommentRequest, ListCommentsRequest, ListRepliesRequest,
        },
        response::{CommentResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::comment::Comment,
    repositories::comment_repo::CommentRepository,
};

pub struct CommentService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisClient,
}

impl<'a> CommentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisClient) -> Self {
        Self { db, redis }
    }

    /// 添加评论
    pub async fn add_comment(&self, user_id: i64, req: &AddCommentRequest) -> Result<String> {
        let comment_repo = CommentRepository::new(self.db);

        // 验证内容
        if req.content.is_empty() {
            return Err(AppError::BadRequest("评论内容不能为空".to_string()));
        }

        if req.content.len() > 1000 {
            return Err(AppError::BadRequest("评论内容过长".to_string()));
        }

        // 如果是回复，验证父评论存在
        let parent_id = if let Some(parent_id_str) = &req.parent_id {
            let parent_id = Uuid::parse_str(parent_id_str)
                .map_err(|_| AppError::BadRequest("无效的父评论ID".to_string()))?;

            if comment_repo.find_by_id(parent_id).await?.is_none() {
                return Err(AppError::NotFound("父评论不存在".to_string()));
            }

            Some(parent_id)
        } else {
            None
        };

        // TODO: 验证目标存在（瞬间/群组等）

        let comment_id = Uuid::new_v4();
        let comment = Comment {
            comment_id,
            user_id,
            target_id: req.target_id.clone(),
            target_type: req.target_type.clone(),
            content: req.content.clone(),
            parent_id,
            created_at: chrono::Utc::now().naive_utc(),
            deleted_at: None,
        };

        comment_repo.create(&comment).await?;

        // TODO: 触发通知（如果不是自己的内容）
        // TODO: 触发WebSocket推送

        Ok(comment_id.to_string())
    }

    /// 删除评论
    pub async fn delete_comment(&self, user_id: i64, req: &DeleteCommentRequest) -> Result<()> {
        let comment_repo = CommentRepository::new(self.db);

        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let comment = comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::NotFound("评论不存在".to_string()))?;

        // 只能删除自己的评论
        if comment.user_id != user_id {
            return Err(AppError::Forbidden("无权删除该评论".to_string()));
        }

        comment_repo.soft_delete(comment_id).await?;

        Ok(())
    }

    /// 点赞评论
    pub async fn like_comment(&self, user_id: i64, req: &LikeCommentRequest) -> Result<()> {
        let comment_repo = CommentRepository::new(self.db);

        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        // 验证评论存在
        if comment_repo.find_by_id(comment_id).await?.is_none() {
            return Err(AppError::NotFound("评论不存在".to_string()));
        }

        comment_repo.like_comment(comment_id, user_id).await?;

        // TODO: 触发通知（如果不是自己的评论）

        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_comment(&self, user_id: i64, req: &UnlikeCommentRequest) -> Result<()> {
        let comment_repo = CommentRepository::new(self.db);

        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        comment_repo.unlike_comment(comment_id, user_id).await?;

        Ok(())
    }

    /// 获取评论列表
    pub async fn list_comments(
        &self,
        req: &ListCommentsRequest,
        current_user_id: Option<i64>,
    ) -> Result<PaginatedResponse<CommentResponse>> {
        let comment_repo = CommentRepository::new(self.db);

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let comments = comment_repo
            .list_by_target(&req.target_id, &req.target_type, limit + 1, offset)
            .await?;

        let has_more = comments.len() > limit as usize;
        let items = self
            .build_comment_responses(comments.into_iter().take(limit as usize).collect(), current_user_id)
            .await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 获取回复列表
    pub async fn list_replies(
        &self,
        req: &ListRepliesRequest,
        current_user_id: Option<i64>,
    ) -> Result<PaginatedResponse<CommentResponse>> {
        let comment_repo = CommentRepository::new(self.db);

        let parent_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let comments = comment_repo
            .list_replies(parent_id, limit + 1, offset)
            .await?;

        let has_more = comments.len() > limit as usize;
        let items = self
            .build_comment_responses(comments.into_iter().take(limit as usize).collect(), current_user_id)
            .await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 构建评论响应（包含点赞数、回复数、是否已点赞等）
    async fn build_comment_responses(
        &self,
        comments: Vec<Comment>,
        current_user_id: Option<i64>,
    ) -> Result<Vec<CommentResponse>> {
        let comment_repo = CommentRepository::new(self.db);
        let mut responses = Vec::new();

        for comment in comments {
            let like_count = comment_repo.get_like_count(comment.comment_id).await?;
            let reply_count = comment_repo.get_reply_count(comment.comment_id).await?;

            let is_liked = if let Some(user_id) = current_user_id {
                comment_repo.is_liked(comment.comment_id, user_id).await?
            } else {
                false
            };

            responses.push(CommentResponse {
                id: comment.comment_id.to_string(),
                user_id: comment.user_id.to_string(),
                target_id: comment.target_id,
                target_type: comment.target_type,
                content: comment.content,
                parent_id: comment.parent_id.map(|id| id.to_string()),
                like_count,
                reply_count,
                is_liked,
                created_at: comment.created_at.and_utc().timestamp(),
            });
        }

        Ok(responses)
    }
}

