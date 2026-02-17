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
        response::{CommentLikeItem, CommentListItem, PaginatedData, UserInfo},
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        comment_repo::{CommentLikeRow, CommentRepository, CommentRow},
        moment_repo::MomentRepository,
        user_repo::UserRepository,
    },
    utils::validation::Validator,
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
            return Ok(UserInfo::new(
                user_id,
                nickname,
                self.media_service
                    .sign_url_if_path(avatar_url.as_deref())
                    .await,
            ));
        }

        let user = self.user_repo.find_by_id(user_id).await?;
        Ok(UserInfo::new(
            user_id,
            user.as_ref()
                .and_then(|u| u.nickname.clone())
                .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
            self.media_service
                .sign_url_if_path(user.and_then(|u| u.avatar_url).as_deref())
                .await,
        ))
    }

    /// 发布评论/回复
    pub async fn add_comment(
        &self,
        user_id: i64,
        req: &CreateCommentRequest,
    ) -> Result<CommentListItem> {
        let mid = Validator::parse_uuid(&req.moment_id, "动态ID")?;

        // 1. 验证动态存在并获取用于事件
        let moment = self
            .moment_repo
            .find_by_id(mid)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("动态不存在".to_string()))?;

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

        let mentions = Validator::parse_mentions(req.mentions.as_deref())?;

        let row = self
            .comment_repo
            .create(
                mid,
                user_id,
                req.content.clone(),
                parent_id,
                reply_to_comment_id,
                reply_to_user_id,
                mentions,
            )
            .await?;

        let summary = crate::events::event_types::MomentSummarySnapshot {
            moment_id: mid,
            cover_url: None,
            text_preview: moment.description.unwrap_or_default(),
        };

        let user_info = self
            .resolve_user_info(
                row.user_id,
                row.user_nickname.clone(),
                row.user_avatar_url.clone(),
            )
            .await?;

        // 3. 发布事件（携带评论者信息，避免 handler 重复查询）
        self.event_bus.publish(AppEvent::MomentCommented {
            moment_id: mid,
            comment_id: row.comment_id,
            user_id: moment.user_id,
            commenter_id: user_id,
            commenter_nickname: Some(user_info.nickname.clone()),
            commenter_avatar_url: user_info.avatar_url.clone(),
            content: row.content.clone(),
            moment_summary: summary,
        });

        Ok(CommentListItem {
            comment_id: row.comment_id.to_string(),
            user: user_info,
            content: row.content,
            parent_id: row.parent_id.map(|id| id.to_string()),
            mentions: row.mentions,
            created_at: row.created_at.to_rfc3339(),
            likes_count: row.like_count,
            replies_count: row.reply_count,
            is_liked: Some(false), // 刚创建的评论，作者未点赞
        })
    }

    /// 获取评论列表
    pub async fn list_comments(
        &self,
        viewer_id: Option<i64>,
        req: &MomentCommentsRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let mid = Validator::parse_uuid(&req.moment_id, "动态ID")?;

        if !self.moment_repo.exists(mid).await? {
            return Err(AppError::MomentNotFound("动态不存在".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_by_moment(mid, limit, req.cursor.clone())
            .await?;

        let liked_set = if let Some(uid) = viewer_id {
            let comment_ids: Vec<Uuid> = rows.iter().map(|r| r.comment_id).collect();
            self.comment_repo
                .batch_check_liked(&comment_ids, uid)
                .await?
        } else {
            std::collections::HashSet::new()
        };

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let is_liked = viewer_id.map(|_| liked_set.contains(&r.comment_id));
            items.push(self.build_comment_list_item(&r, is_liked).await?);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn list_replies(
        &self,
        viewer_id: Option<i64>,
        req: &CommentRepliesRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;

        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_replies(comment_id, limit, req.cursor.clone())
            .await?;

        let liked_set = if let Some(uid) = viewer_id {
            let comment_ids: Vec<Uuid> = rows.iter().map(|r| r.comment_id).collect();
            self.comment_repo
                .batch_check_liked(&comment_ids, uid)
                .await?
        } else {
            std::collections::HashSet::new()
        };

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let is_liked = viewer_id.map(|_| liked_set.contains(&r.comment_id));
            items.push(self.build_comment_list_item(&r, is_liked).await?);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    async fn build_comment_list_item(
        &self,
        row: &CommentRow,
        is_liked: Option<bool>,
    ) -> Result<CommentListItem> {
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
            content: row.content.clone(),
            parent_id: row.parent_id.map(|id| id.to_string()),
            mentions: row.mentions.clone(),
            created_at: row.created_at.to_rfc3339(),
            likes_count: row.like_count,
            replies_count: row.reply_count,
            is_liked,
        })
    }

    pub async fn get_likes_list(
        &self,
        req: &CommentLikesRequest,
    ) -> Result<PaginatedData<CommentLikeItem>> {
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;

        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_likes(comment_id, limit, req.cursor.clone())
            .await?;

        let items = self.build_like_items(rows).await?;

        Ok(PaginatedData { items, next_cursor })
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
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;

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
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;

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
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;

        let _comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;

        self.comment_repo.delete_like(comment_id, user_id).await?;
        Ok(())
    }
}
