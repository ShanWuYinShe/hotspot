use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateCommentRequest, DeleteCommentRequest, LikeCommentRequest, UnlikeCommentRequest,
            GetCommentsRequest, GetRepliesRequest, GetCommentLikesListRequest,
        },
        response::{CommentListItem, PaginatedData, PaginationMeta, UserInfo, CreateCommentData, CommentLikeItem},
    },
    error::{AppError, Result},
    events::events::AppEvent,
    repositories::{
        comment_repo::{CommentRepository, CommentRow},
        user_repo::UserRepository
    },
    utils::validation::Validator,
    EventBus,
};

pub struct CommentService<'a> {
    db: &'a PgPool,
    event_bus: Arc<EventBus>,
    _redis: &'a RedisConnectionManager,
}

impl<'a> CommentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self { db, event_bus, _redis: redis }
    }

    /// 添加评论
    pub async fn add_comment(&self, user_id: i64, req: &CreateCommentRequest) -> Result<CreateCommentData> {
        let comment_repo = CommentRepository::new(self.db.clone());

        // 验证内容
        if req.content.is_empty() {
            return Err(AppError::BadRequest("评论内容不能为空".to_string()));
        }

        if req.content.len() > 1000 {
            return Err(AppError::BadRequest("评论内容过长".to_string()));
        }

        // 验证瞬间存在
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 如果是回复，验证父评论存在并获取父评论信息
        let parent_comment = if let Some(parent_id_str) = &req.parent_id {
            let parent_id = Uuid::parse_str(parent_id_str)
                .map_err(|_| AppError::BadRequest("无效的父评论ID".to_string()))?;

            let parent = comment_repo.find_by_id(parent_id).await?
                .ok_or_else(|| AppError::NotFound("父评论不存在".to_string()))?;

            Some(parent)
        } else {
            None
        };

        let parent_id = parent_comment.as_ref().map(|p| p.comment_id);

        let row = comment_repo.create(
            moment_id,
            user_id,
            req.content.clone(),
            parent_id,
            None,
        ).await?;

        // 如果是回复评论，发布评论回复事件
        if let Some(parent) = parent_comment {
            self.event_bus.publish(AppEvent::CommentReplied {
                moment_id,
                comment_id: parent.comment_id,  // 原评论ID
                reply_id: row.comment_id,       // 新回复ID
                user_id: parent.user_id,        // 原评论作者
                replier_id: user_id,            // 回复者
                content: req.content.clone(),
            });
        }

        Ok(CreateCommentData {
            id: row.comment_id.to_string(),
        })
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

        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        // 验证评论存在
        if comment_repo.find_by_id(comment_id).await?.is_none() {
            return Err(AppError::NotFound("评论不存在".to_string()));
        }

        comment_repo.like_comment(comment_id, user_id).await?;
        Ok(())
    }

    /// 取消点赞评论
    pub async fn unlike_comment(&self, user_id: i64, req: &UnlikeCommentRequest) -> Result<()> {
        let comment_repo = CommentRepository::new(self.db.clone());

        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        // 验证评论存在
        if comment_repo.find_by_id(comment_id).await?.is_none() {
            return Err(AppError::NotFound("评论不存在".to_string()));
        }

        comment_repo.unlike_comment(comment_id, user_id).await?;
        Ok(())
    }

    /// 获取评论列表
    pub async fn list_comments(
        &self,
        req: &GetCommentsRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let comment_repo = CommentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor_uuid = Validator::parse_cursor(req.cursor.as_deref())?;
        let comments = comment_repo
            .list_by_moment(moment_id, (limit + 1) as i32, cursor_uuid)
            .await?;

        let has_more = comments.len() > limit as usize;
        let comments_to_return: Vec<CommentRow> = comments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            comments_to_return.last().map(|c| c.comment_id.to_string())
        } else {
            None
        };

        let items = self
            .build_comment_responses(comments_to_return)
            .await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 获取回复列表
    pub async fn list_replies(
        &self,
        req: &GetRepliesRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let comment_repo = CommentRepository::new(self.db.clone());

        let parent_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor_uuid = Validator::parse_cursor(req.cursor.as_deref())?;
        let comments = comment_repo
            .list_replies(parent_id, (limit + 1) as i32, cursor_uuid)
            .await?;

        let has_more = comments.len() > limit as usize;
        let comments_to_return: Vec<CommentRow> = comments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            comments_to_return.last().map(|c| c.comment_id.to_string())
        } else {
            None
        };

        let items = self
            .build_comment_responses(comments_to_return)
            .await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 获取评论点赞列表
    pub async fn get_likes_list(
        &self,
        req: &GetCommentLikesListRequest,
    ) -> Result<PaginatedData<CommentLikeItem>> {
        let comment_repo = CommentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let comment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let likes = comment_repo.list_likes(comment_id, limit + 1, cursor).await?;

        let has_more = likes.len() > limit as usize;
        let likes_to_return: Vec<_> = likes.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            likes_to_return.last().map(|l| l.cursor_id.to_string())
        } else {
            None
        };

        // 批量获取用户信息
        let user_ids: Vec<i64> = likes_to_return.iter().map(|l| l.user_id).collect();
        let users = user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        let items: Vec<CommentLikeItem> = likes_to_return.into_iter().filter_map(|like| {
            let user = user_map.get(&like.user_id)?;
            Some(CommentLikeItem {
                user: UserInfo {
                    user_id: user.user_id,
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                    user_type: user.user_type,
                },
                liked_at: like.created_at.to_rfc3339(),
            })
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            }
        })
    }

    /// 构建评论响应（使用批量查询避免 N+1 问题）
    async fn build_comment_responses(
        &self,
        comments: Vec<CommentRow>,
    ) -> Result<Vec<CommentListItem>> {
        if comments.is_empty() {
            return Ok(Vec::new());
        }

        let comment_repo = CommentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        // 收集所有需要查询的用户ID
        let mut user_ids: Vec<i64> = comments.iter().map(|c| c.user_id).collect();
        let reply_to_ids: Vec<i64> = comments.iter()
            .filter_map(|c| c.reply_to_user_id)
            .collect();
        user_ids.extend(reply_to_ids);
        user_ids.sort();
        user_ids.dedup();

        // 批量获取用户信息
        let users = user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 批量获取评论统计数据
        let comment_ids: Vec<Uuid> = comments.iter().map(|c| c.comment_id).collect();
        let stats = comment_repo.get_stats_batch(&comment_ids).await?;

        // 构建响应
        let responses: Vec<CommentListItem> = comments.into_iter().filter_map(|comment| {
            let user_model = user_map.get(&comment.user_id)?;
            let (likes_count, replies_count) = stats.get(&comment.comment_id).copied().unwrap_or((0, 0));

            let reply_to_user = comment.reply_to_user_id.and_then(|reply_to_id| {
                let reply_user = user_map.get(&reply_to_id)?;
                Some(UserInfo {
                    user_id: reply_user.user_id,
                    nickname: reply_user.nickname.clone(),
                    avatar_url: reply_user.avatar_url.clone(),
                    user_type: reply_user.user_type,
                })
            });

            Some(CommentListItem {
                id: comment.comment_id.to_string(),
                user: UserInfo {
                    user_id: user_model.user_id,
                    nickname: user_model.nickname.clone(),
                    avatar_url: user_model.avatar_url.clone(),
                    user_type: user_model.user_type,
                },
                content: comment.content,
                parent_id: comment.parent_id.map(|id| id.to_string()),
                reply_to_user,
                created_at: comment.created_at.to_rfc3339(),
                likes_count,
                replies_count,
            })
        }).collect();

        Ok(responses)
    }

}
