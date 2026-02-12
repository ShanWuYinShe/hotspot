//! 评论服务层实现
//!
//! 核心业务逻辑：发布评论、回复评论、点赞评论、分页检索

use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CommentLikesRequest, CommentRepliesRequest, CreateCommentRequest, DeleteCommentRequest,
            LikeCommentRequest, MomentCommentsRequest, UnlikeCommentRequest,
        },
        response::{CommentLikeItem, CommentListItem, PaginatedData, PaginationMeta, UserInfo},
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        comment_repo::{CommentLikeRow, CommentRepository},
        moment_repo::MomentRepository,
        user_repo::UserRepository,
    },
    EventBus,
};

#[derive(Clone)]
pub struct CommentService {
    event_bus: Arc<EventBus>,
    comment_repo: CommentRepository,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    media_service: crate::services::media_service::MediaService,
}

impl CommentService {
    pub fn new(
        event_bus: Arc<EventBus>,
        comment_repo: CommentRepository,
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            event_bus,
            comment_repo,
            moment_repo,
            user_repo,
            media_service,
        }
    }

    async fn resolve_user_info(
        &self,
        user_id: i64,
        nickname: Option<String>,
        avatar_url: Option<String>,
    ) -> Result<UserInfo> {
        if let Some(nickname) = nickname {
            return Ok(UserInfo {
                user_id,
                nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(avatar_url.as_deref())
                    .await,
                user_type: crate::types::enums::UserType::Normal,
            });
        }

        let user = self.user_repo.find_by_id(user_id).await?;
        Ok(UserInfo {
            user_id,
            nickname: user
                .as_ref()
                .and_then(|u| u.nickname.clone())
                .unwrap_or_else(|| "未知用户".to_string()),
            avatar_url: self
                .media_service
                .sign_url_if_path(user.and_then(|u| u.avatar_url).as_deref())
                .await,
            user_type: crate::types::enums::UserType::Normal,
        })
    }

    /// 发布评论/回复
    pub async fn add_comment(
        &self,
        user_id: i64,
        req: &CreateCommentRequest,
    ) -> Result<CommentListItem> {
        let mid = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效动态ID".to_string()))?;

        // 1. 验证动态存在
        if !self.moment_repo.exists(mid).await? {
            return Err(AppError::MomentNotFound("动态不存在".to_string()));
        }

        // 2. 创建评论 (Trigger 会自动同步 moment.comments_count)
        let parent_id = req
            .parent_id
            .as_deref()
            .and_then(|id| Uuid::parse_str(id).ok());
        let (reply_to_comment_id, reply_to_user_id) = if let Some(pid) = parent_id {
            let parent = self
                .comment_repo
                .find_by_id(pid)
                .await?
                .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;
            (Some(pid), Some(parent.user_id))
        } else {
            (None, None)
        };

        let row = self
            .comment_repo
            .create(
                mid,
                user_id,
                req.content.clone(),
                parent_id,
                reply_to_comment_id,
                reply_to_user_id,
                req.mentions.clone(),
            )
            .await?;

        let moment = self
            .moment_repo
            .find_by_id(mid)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("动态不存在".to_string()))?;

        let summary = crate::events::event_types::MomentSummarySnapshot {
            moment_id: mid,
            cover_url: None,
            text_preview: moment.description.unwrap_or_default(),
        };

        // 3. 发布事件
        self.event_bus.publish(AppEvent::MomentCommented {
            moment_id: mid,
            comment_id: row.comment_id,
            user_id: moment.user_id,
            commenter_id: user_id,
            content: row.content.clone(),
            moment_summary: summary,
        });

        let user_info = self
            .resolve_user_info(
                row.user_id,
                row.user_nickname.clone(),
                row.user_avatar_url.clone(),
            )
            .await?;

        Ok(CommentListItem {
            comment_id: row.comment_id.to_string(),
            user: user_info,
            content: row.content,
            parent_id: row.parent_id.map(|id| id.to_string()),
            mentions: row.mentions,
            created_at: row.created_at.to_rfc3339(),
            likes_count: row.like_count,
            replies_count: row.reply_count,
        })
    }

    /// 获取评论列表
    pub async fn list_comments(
        &self,
        req: &MomentCommentsRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let mid = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效动态ID".to_string()))?;

        if !self.moment_repo.exists(mid).await? {
            return Err(AppError::MomentNotFound("动态不存在".to_string()));
        }

        let (rows, next_cursor) = self
            .comment_repo
            .list_by_moment(mid, req.limit.unwrap_or(20) as i64, req.cursor.clone())
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let user_info = self
                .resolve_user_info(
                    r.user_id,
                    r.user_nickname.clone(),
                    r.user_avatar_url.clone(),
                )
                .await?;
            items.push(CommentListItem {
                comment_id: r.comment_id.to_string(),
                user: user_info,
                content: r.content,
                parent_id: r.parent_id.map(|id| id.to_string()),
                mentions: r.mentions.clone(),
                created_at: r.created_at.to_rfc3339(),
                likes_count: r.like_count,
                replies_count: r.reply_count,
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn list_replies(
        &self,
        req: &CommentRepliesRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效评论ID".to_string()))?;

        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }

        let (rows, next_cursor) = self
            .comment_repo
            .list_replies(
                comment_id,
                req.limit.unwrap_or(20) as i64,
                req.cursor.clone(),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let user_info = self
                .resolve_user_info(
                    r.user_id,
                    r.user_nickname.clone(),
                    r.user_avatar_url.clone(),
                )
                .await?;
            items.push(CommentListItem {
                comment_id: r.comment_id.to_string(),
                user: user_info,
                content: r.content,
                parent_id: r.parent_id.map(|id| id.to_string()),
                mentions: r.mentions.clone(),
                created_at: r.created_at.to_rfc3339(),
                likes_count: r.like_count,
                replies_count: r.reply_count,
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn get_likes_list(
        &self,
        req: &CommentLikesRequest,
    ) -> Result<PaginatedData<CommentLikeItem>> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效评论ID".to_string()))?;

        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }

        let (rows, next_cursor) = self
            .comment_repo
            .list_likes(
                comment_id,
                req.limit.unwrap_or(20) as i64,
                req.cursor.clone(),
            )
            .await?;

        let items = self.build_like_items(rows).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    async fn build_like_items(&self, rows: Vec<CommentLikeRow>) -> Result<Vec<CommentLikeItem>> {
        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let user_info = self
                .resolve_user_info(
                    r.user_id,
                    r.user_nickname.clone(),
                    r.user_avatar_url.clone(),
                )
                .await?;
            items.push(CommentLikeItem {
                user: user_info,
                liked_at: r.created_at.to_rfc3339(),
            });
        }
        Ok(items)
    }

    pub async fn delete_comment(&self, user_id: i64, req: &DeleteCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效评论ID".to_string()))?;

        let comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;

        if comment.user_id != user_id {
            return Err(AppError::Forbidden("无权删除该评论".to_string()));
        }

        self.comment_repo.delete(comment_id).await?;
        Ok(())
    }

    pub async fn like_comment(&self, user_id: i64, req: &LikeCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效评论ID".to_string()))?;

        let comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;

        let inserted = self.comment_repo.insert_like(comment_id, user_id).await?;
        if inserted {
            self.event_bus.publish(AppEvent::CommentLiked {
                comment_id,
                moment_id: comment.moment_id,
                user_id: comment.user_id,
                liker_id: user_id,
                comment_preview: comment.content.chars().take(100).collect(),
            });
        }

        Ok(())
    }

    pub async fn unlike_comment(&self, user_id: i64, req: &UnlikeCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效评论ID".to_string()))?;

        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }

        self.comment_repo.delete_like(comment_id, user_id).await?;
        Ok(())
    }
}
