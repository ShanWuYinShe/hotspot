use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AddCommentRequest, DeleteCommentRequest, LikeCommentRequest,
            UnlikeCommentRequest, ListCommentsRequest, ListRepliesRequest,
        },
        response::{CommentResponse, PaginatedResponse, UserResponse},
    },
    error::{AppError, Result},
    models::comment::Comment,
    repositories::{comment_repo::CommentRepository, user_repo::UserRepository},
};

pub struct CommentService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisConnectionManager,
}

impl<'a> CommentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisConnectionManager) -> Self {
        Self { db, redis }
    }

    /// 添加评论
    pub async fn add_comment(&self, user_id: i64, req: &AddCommentRequest) -> Result<String> {
        let comment_repo = CommentRepository::new(self.db.clone());

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

        // 验证瞬间存在
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let comment_id = Uuid::new_v4();
        let now = chrono::Utc::now();
        let comment = Comment {
            comment_id,
            moment_id,
            user_id,
            content: req.content.clone(),
            parent_id,
            reply_to_user_id: None, // TODO: 如果是回复，设置被回复者ID
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
        };

        comment_repo.create(&comment).await?;

        // TODO: 触发通知（如果不是自己的内容）
        // TODO: 触发WebSocket推送

        Ok(comment_id.to_string())
    }

    /// 删除评论
    pub async fn delete_comment(&self, user_id: i64, req: &DeleteCommentRequest) -> Result<()> {
        let comment_repo = CommentRepository::new(self.db.clone());

        let comment_id = Uuid::parse_str(&req.id)
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
        let comment_repo = CommentRepository::new(self.db.clone());

        let comment_id = Uuid::parse_str(&req.id)
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
        let comment_repo = CommentRepository::new(self.db.clone());

        let comment_id = Uuid::parse_str(&req.id)
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
        let comment_repo = CommentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let comments = comment_repo
            .list_by_moment(moment_id, req.cursor.clone(), limit + 1)
            .await?;

        let has_more = comments.len() > limit as usize;
        let comments_to_return: Vec<Comment> = comments.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的comment_id（业务ID）
        let next_cursor = if has_more {
            comments_to_return.last().map(|c| c.comment_id.to_string())
        } else {
            None
        };

        let items = self
            .build_comment_responses(comments_to_return, current_user_id)
            .await?;

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
        let comment_repo = CommentRepository::new(self.db.clone());

        let parent_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let comments = comment_repo
            .list_replies(parent_id, req.cursor.clone(), limit + 1)
            .await?;

        let has_more = comments.len() > limit as usize;
        let comments_to_return: Vec<Comment> = comments.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的comment_id（业务ID）
        let next_cursor = if has_more {
            comments_to_return.last().map(|c| c.comment_id.to_string())
        } else {
            None
        };

        let items = self
            .build_comment_responses(comments_to_return, current_user_id)
            .await?;

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
        let comment_repo = CommentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());
        let mut responses = Vec::new();

        for comment in comments {
            let likes_count = comment_repo.get_like_count(comment.comment_id).await? as i32;
            let replies_count = comment_repo.get_reply_count(comment.comment_id).await? as i32;

            let liked_by_me = if let Some(user_id) = current_user_id {
                Some(comment_repo.is_liked(comment.comment_id, user_id).await?)
            } else {
                None
            };

            // 获取评论者用户信息
            let user_model = user_repo.find_by_id(comment.user_id).await?
                .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;
            let user = UserResponse {
                user_id: user_model.user_id.to_string(),
                nickname: user_model.nickname,
                avatar_url: user_model.avatar_url,
            };

            // 获取被回复用户信息（如果有）
            let reply_to_user = if let Some(reply_to_id) = comment.reply_to_user_id {
                let reply_user = user_repo.find_by_id(reply_to_id).await?
                    .ok_or_else(|| AppError::NotFound("被回复用户不存在".to_string()))?;
                Some(UserResponse {
                    user_id: reply_user.user_id.to_string(),
                    nickname: reply_user.nickname,
                    avatar_url: reply_user.avatar_url,
                })
            } else {
                None
            };

            responses.push(CommentResponse {
                id: comment.comment_id.to_string(),
                moment_id: comment.moment_id.to_string(),
                user,
                content: comment.content,
                parent_id: comment.parent_id.map(|id| id.to_string()),
                reply_to_user,
                created_at: comment.created_at,
                likes_count,
                replies_count,
                liked_by_me,
            });
        }

        Ok(responses)
    }
}

