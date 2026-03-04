//! 评论服务层实现
//!
//! 核心业务逻辑：发布评论、回复评论、点赞评论、分页检索

use std::sync::Arc;
use tracing::{info, warn};
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
    utils::{bloom_filter::ResourceExistenceChecker, validation::Validator},
    EventBus,
};

#[derive(Clone)]
pub struct CommentService {
    event_bus: Arc<EventBus>,
    comment_repo: CommentRepository,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    media_service: crate::services::media_service::MediaService,
    resource_checker: ResourceExistenceChecker,
}

impl CommentService {
    pub fn new(
        event_bus: Arc<EventBus>,
        comment_repo: CommentRepository,
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        media_service: crate::services::media_service::MediaService,
        resource_checker: ResourceExistenceChecker,
    ) -> Self {
        Self {
            event_bus,
            comment_repo,
            moment_repo,
            user_repo,
            media_service,
            resource_checker,
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
        let started_at = std::time::Instant::now();
        let mid = Validator::parse_uuid(&req.moment_id, "动态ID")?;
        Validator::validate_comment_content(&req.content)?;
        info!(
            user_id,
            moment_id = %mid,
            has_parent_id = req.parent_id.is_some(),
            mentions_count = req.mentions.as_ref().map(|v| v.len()).unwrap_or(0),
            content_len = req.content.chars().count(),
            "comment.add started"
        );

        // 1. 验证动态存在并获取用于事件
        let moment = self.moment_repo.find_by_id(mid).await?.ok_or_else(|| {
            warn!(user_id, moment_id = %mid, "comment.add rejected: moment not found");
            AppError::MomentNotFound("动态不存在".to_string())
        })?;

        // 2. 创建评论 (Trigger 会自动同步 moment.comments_count)
        // 两层评论模型：parent_id 指向回复时，自动改写为楼顶层评论 ID
        let raw_parent_id = req
            .parent_id
            .as_deref()
            .and_then(|id| Uuid::parse_str(id).ok());
        let (reply_to_comment_id, reply_to_user_id, parent_id) = if let Some(pid) = raw_parent_id {
            let parent = self
                .comment_repo
                .find_by_id(pid)
                .await?
                .ok_or_else(|| {
                    warn!(user_id, moment_id = %mid, parent_comment_id = %pid, "comment.add rejected: parent comment not found");
                    AppError::CommentNotFound("评论不存在".to_string())
                })?;
            let effective_parent = parent.parent_id.unwrap_or(pid);
            (Some(pid), Some(parent.user_id), Some(effective_parent))
        } else {
            (None, None, None)
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

        if let Err(e) = self
            .resource_checker
            .record_comment(&row.comment_id.to_string())
            .await
        {
            tracing::warn!(comment_id = %row.comment_id, error = %e, "record_comment bloom failed");
        }

        let summary = crate::events::event_types::MomentSummarySnapshot {
            moment_id: mid,
            cover_url: None,
            description_preview: moment.description.unwrap_or_default(),
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
            moment_summary: summary.clone(),
        });

        // 4. 回复评论时，通知被回复者
        if let (Some(replied_comment_id), Some(replied_user_id)) =
            (reply_to_comment_id, reply_to_user_id)
        {
            if replied_user_id != user_id {
                let replied_comment = self.comment_repo.find_by_id(replied_comment_id).await?;
                let replied_preview = replied_comment
                    .map(|c| c.content.chars().take(100).collect::<String>())
                    .unwrap_or_default();
                let top_level_comment_id = parent_id.unwrap_or(replied_comment_id);
                self.event_bus.publish(AppEvent::CommentReplied {
                    moment_id: mid,
                    moment_summary: summary.clone(),
                    comment_id: top_level_comment_id,
                    replied_comment_preview: replied_preview,
                    reply_id: row.comment_id,
                    user_id: replied_user_id,
                    replier_id: user_id,
                    content: row.content.clone(),
                });
            }
        }
        info!(
            user_id,
            moment_id = %mid,
            comment_id = %row.comment_id,
            is_reply = reply_to_comment_id.is_some(),
            elapsed_ms = started_at.elapsed().as_millis(),
            "comment.add completed"
        );

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

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_by_moment(mid, limit, pagination.and_then(|p| p.cursor.clone()))
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

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_replies(comment_id, limit, pagination.and_then(|p| p.cursor.clone()))
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

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let (rows, next_cursor) = self
            .comment_repo
            .list_likes(comment_id, limit, pagination.and_then(|p| p.cursor.clone()))
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
        let started_at = std::time::Instant::now();
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;
        info!(user_id, comment_id = %comment_id, "comment.delete started");

        let comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;

        if comment.user_id != user_id {
            warn!(
                user_id,
                comment_id = %comment_id,
                owner_id = comment.user_id,
                "comment.delete rejected: not owner"
            );
            return Err(AppError::Forbidden("无权删除该评论".to_string()));
        }

        self.comment_repo.delete(comment_id).await?;
        info!(
            user_id,
            comment_id = %comment_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "comment.delete completed"
        );
        Ok(())
    }

    pub async fn like_comment(&self, user_id: i64, req: &LikeCommentRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;
        info!(user_id, comment_id = %comment_id, "comment.like started");

        let comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| {
                warn!(user_id, comment_id = %comment_id, "comment.like rejected: comment not found");
                AppError::CommentNotFound("评论不存在".to_string())
            })?;

        let inserted = self.comment_repo.insert_like(comment_id, user_id).await?;
        if inserted {
            let moment = self.moment_repo.find_by_id(comment.moment_id).await?;
            let moment_summary = crate::events::event_types::MomentSummarySnapshot {
                moment_id: comment.moment_id,
                cover_url: None,
                description_preview: moment
                    .as_ref()
                    .and_then(|m| m.description.clone())
                    .unwrap_or_default(),
            };
            self.event_bus.publish(AppEvent::CommentLiked {
                comment_id,
                moment_id: comment.moment_id,
                moment_summary,
                user_id: comment.user_id,
                liker_id: user_id,
                comment_preview: comment.content.chars().take(100).collect(),
            });
        }
        info!(
            user_id,
            comment_id = %comment_id,
            inserted,
            elapsed_ms = started_at.elapsed().as_millis(),
            "comment.like completed"
        );

        Ok(())
    }

    pub async fn unlike_comment(&self, user_id: i64, req: &UnlikeCommentRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let comment_id = Validator::parse_uuid(&req.comment_id, "评论ID")?;
        info!(user_id, comment_id = %comment_id, "comment.unlike started");

        let _comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| {
                warn!(user_id, comment_id = %comment_id, "comment.unlike rejected: comment not found");
                AppError::CommentNotFound("评论不存在".to_string())
            })?;

        self.comment_repo.delete_like(comment_id, user_id).await?;
        info!(
            user_id,
            comment_id = %comment_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "comment.unlike completed"
        );
        Ok(())
    }
}
