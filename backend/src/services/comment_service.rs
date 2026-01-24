use redis::aio::ConnectionManager as RedisConnectionManager;
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
        comment_repo::{CommentRepository, CommentRow},
        moment_repo::MomentRepository,
        user_repo::UserRepository,
    },
    utils::{
        bloom_filter::ResourceExistenceChecker,
        cache::{cache_keys, CacheManager},
        validation::Validator,
    },
    EventBus,
};

#[derive(Clone)]
pub struct CommentService {
    event_bus: Arc<EventBus>,
    redis: RedisConnectionManager,
    comment_repo: CommentRepository,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    cache_manager: CacheManager,
}

impl CommentService {
    pub fn new(
        event_bus: Arc<EventBus>,
        redis: RedisConnectionManager,
        comment_repo: CommentRepository,
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            event_bus,
            redis,
            comment_repo,
            moment_repo,
            user_repo,
            cache_manager,
        }

    }

    /// 添加评论
    pub async fn add_comment(
        &self,
        user_id: i64,
        req: &CreateCommentRequest,
    ) -> Result<CommentListItem> {
        // 验证内容
        if req.content.is_empty() {
            return Err(AppError::BadRequest("评论内容不能为空".to_string()));
        }


        if req.content.len() > 1000 {
            return Err(AppError::BadRequest("评论内容过长".to_string()));
        }


        // 验证 mentions（最多 10 个）
        if let Some(mentions) = &req.mentions {
            if mentions.len() > 10 {
                return Err(AppError::BadRequest("最多只能提及10个用户".to_string()));
            }

            // 验证所有被提及的用户是否存在
            for &mentioned_user_id in mentions {
                if !self.user_repo.exists(mentioned_user_id).await? {
                    return Err(AppError::UserNotFound(format!(
                        "被提及的用户 {} 不存在",
                        mentioned_user_id
                    )));
                }

            }

        }


        // 验证瞬间存在
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 如果是回复，验证父评论存在并获取父评论信息
        let parent_comment = if let Some(parent_id_str) = &req.parent_id {
            let parent_id = Uuid::parse_str(parent_id_str)
                .map_err(|_| AppError::BadRequest("无效的父评论ID".to_string()))?;

            let parent = self
                .comment_repo
                .find_by_id(parent_id)
                .await?
                .ok_or_else(|| AppError::CommentNotFound("父评论不存在".to_string()))?;

            Some(parent)
        } else {
            None
        };

        let parent_id = parent_comment.as_ref().map(|p| p.comment_id);

        let row = self
            .comment_repo
            .create(
                moment_id,
                user_id,
                req.content.clone(),
                parent_id,
                None,
                req.mentions.clone(),
            )
            .await?;


        // 记录评论到布隆过滤器
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_comment(&row.comment_id.to_string())
            .await
        {
            tracing::warn!(comment_id = %row.comment_id, error = %e, "record_comment bloom failed");
        }

        // 发布“瞬间被评论”事件：通知瞬间作者（回复/普通评论都算评论）
        if let Some(moment_owner_id) = self.get_moment_owner_cached(moment_id).await? {
            if moment_owner_id != user_id {
                let summary_snapshot = self.moment_repo.get_summary_snapshot(moment_id).await?;
                if let Some(summary) = summary_snapshot {
                    self.event_bus.publish(AppEvent::MomentCommented {
                        moment_id,
                        moment_summary: summary,
                        comment_id: row.comment_id,
                        user_id: moment_owner_id,
                        commenter_id: user_id,
                        content: req.content.clone(),
                    });
                }

            }

        }


        // 如果是回复评论，发布评论回复事件
        if let Some(parent) = parent_comment {
            self.event_bus.publish(AppEvent::CommentReplied {
                moment_id,
                comment_id: parent.comment_id,    // 原评论ID
                reply_to_content: parent.content, // 原评论内容摘要
                reply_id: row.comment_id,         // 新回复ID
                user_id: parent.user_id,          // 原评论作者
                replier_id: user_id,              // 回复者
                content: req.content.clone(),
            });
        }


        // 触发 @ 提及事件
        if let Some(mentions) = &req.mentions {
            for &mentioned_user_id in mentions {
                if mentioned_user_id != user_id {
                    // 发送提及通知
                    let _ = self.event_bus.publish(AppEvent::CommentMentioned {
                        moment_id,
                        comment_id: row.comment_id,
                        mentioned_user_id,
                        mentioning_user_id: user_id,
                        content_preview: if req.content.len() > 50 {
                            format!("{}...", &req.content[..50])
                        } else {
                            req.content.clone()
                        },
                    });
                }

            }

        }


        // 构造响应
        let users = self.user_repo.find_profiles(&[user_id]).await?;
        let user_info = users
            .first()
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        Ok(CommentListItem {
            comment_id: row.comment_id.to_string(),
            user: UserInfo {
                user_id: user_info.user_id,
                nickname: user_info.nickname.clone(),
                avatar_url: user_info.avatar_url.clone(),
                user_type: user_info.user_type,
            },
            content: row.content.clone(),
            mentions: row.mentions.clone(),
            parent_id: row.parent_id.map(|id| id.to_string()),
            created_at: row.created_at.to_rfc3339(),
            likes_count: 0,
            replies_count: 0,
        })
    }

    /// 删除评论
    pub async fn delete_comment(&self, user_id: i64, req: &DeleteCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let comment = self
            .comment_repo
            .find_by_id(comment_id)
            .await?
            .ok_or_else(|| AppError::CommentNotFound("评论不存在".to_string()))?;

        // 只能删除自己的评论
        if comment.user_id != user_id {
            return Err(AppError::Forbidden("无权删除该评论".to_string()));
        }


        self.comment_repo.soft_delete(comment_id).await?;

        Ok(())
    }

    /// 点赞评论
    pub async fn like_comment(&self, user_id: i64, req: &LikeCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        // 验证评论存在
        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }


        let inserted = self.comment_repo.like_comment(comment_id, user_id).await?;
        if !inserted {
            return Err(AppError::Conflict("已点赞过该评论".to_string()));
        }

        Ok(())
    }

    /// 取消点赞评论
    pub async fn unlike_comment(&self, user_id: i64, req: &UnlikeCommentRequest) -> Result<()> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        // 验证评论存在
        if !self.comment_repo.exists(comment_id).await? {
            return Err(AppError::CommentNotFound("评论不存在".to_string()));
        }


        let deleted = self
            .comment_repo
            .unlike_comment(comment_id, user_id)
            .await?;
        if !deleted {
            return Err(AppError::Conflict("尚未点赞该评论".to_string()));
        }

        Ok(())
    }

    /// 获取评论列表
    pub async fn list_comments(
        &self,
        req: &MomentCommentsRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (comments, next_cursor) = self
            .comment_repo
            .list_by_moment(moment_id, limit as i32, cursor)
            .await?;

        let items = self.build_comment_responses(comments).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 获取回复列表
    pub async fn list_replies(
        &self,
        req: &CommentRepliesRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        let parent_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (comments, next_cursor) = self
            .comment_repo
            .list_replies(parent_id, limit as i32, cursor)
            .await?;

        let items = self.build_comment_responses(comments).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 获取评论点赞列表
    pub async fn get_likes_list(
        &self,
        req: &CommentLikesRequest,
    ) -> Result<PaginatedData<CommentLikeItem>> {
        let comment_id = Uuid::parse_str(&req.comment_id)
            .map_err(|_| AppError::BadRequest("无效的评论ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (likes, next_cursor) = self
            .comment_repo
            .list_likes(comment_id, limit as i64, cursor)
            .await?;

        // 批量获取用户信息
        let user_ids: Vec<i64> = likes.iter().map(|l| l.user_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let items: Vec<CommentLikeItem> = likes
            .into_iter()
            .filter_map(|like| {
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
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
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


        // 收集所有需要查询的用户ID
        let mut user_ids: Vec<i64> = comments.iter().map(|c| c.user_id).collect();
        user_ids.sort();
        user_ids.dedup();

        // 批量获取用户信息
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取评论统计数据
        let comment_ids: Vec<Uuid> = comments.iter().map(|c| c.comment_id).collect();
        let stats = self.comment_repo.get_stats_batch(&comment_ids).await?;

        // 构建响应
        let responses: Vec<CommentListItem> = comments
            .into_iter()
            .filter_map(|comment| {
                let user_model = user_map.get(&comment.user_id)?;
                let (likes_count, replies_count) =
                    stats.get(&comment.comment_id).copied().unwrap_or((0, 0));

                Some(CommentListItem {
                    comment_id: comment.comment_id.to_string(),
                    user: UserInfo {
                        user_id: user_model.user_id,
                        nickname: user_model.nickname.clone(),
                        avatar_url: user_model.avatar_url.clone(),
                        user_type: user_model.user_type,
                    },
                    content: comment.content,
                    parent_id: comment.parent_id.map(|id| id.to_string()),
                    mentions: comment.mentions,
                    created_at: comment.created_at.to_rfc3339(),
                    likes_count,
                    replies_count,
                })
            })
            .collect();

        Ok(responses)
    }

    async fn get_moment_owner_cached(&self, moment_id: Uuid) -> Result<Option<i64>> {
        let key = moment_id.to_string();
        let cached = self
            .cache_manager
            .get_or_set_with_lock_with_ttl(
                cache_keys::MOMENT_OWNER,
                &key,
                3600,
                300,
                || async {
                    let owner = self
                        .moment_repo
                        .get_owner(moment_id)
                        .await
                        .map_err(|e| Box::new(e) as Box<dyn std::error::Error + Send + Sync>)?;
                    Ok(owner)
                },
            )
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        Ok(cached)
    }
}
