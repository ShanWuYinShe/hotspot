use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{ListFeedRequest, FeedFeedbackRequest},
        response::{FeedItem, PaginatedData, PaginationMeta},
    },
    error::AppError,
    repositories::{
        feed_repo::FeedRepository,
        moment_repo::MomentRepository,
        user_repo::UserRepository,
    },
};

#[derive(Clone)]
pub struct FeedService {
    redis: ConnectionManager,
    feed_repo: FeedRepository,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
}

impl FeedService {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self {
            redis,
            feed_repo: FeedRepository::new(pool.clone()),
            moment_repo: MomentRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool),
        }
    }

    /// 获取个性化信息流
    pub async fn get_feed(
        &self,
        user_id: i64,
        req: ListFeedRequest,
    ) -> Result<PaginatedData<FeedItem>, AppError> {
        let offset = if let Some(cursor_str) = req.cursor {
            cursor_str.parse::<i64>()
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?
        } else {
            0
        };
        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 尝试从Redis缓存获取推荐内容
        let cached_feed = self.get_cached_feed(user_id, offset, limit).await?;

        if !cached_feed.is_empty() {
            // 从缓存获取成功
            let has_more = cached_feed.len() > limit as usize;
            let items: Vec<FeedItem> = if has_more {
                cached_feed.into_iter().take(limit as usize).collect()
            } else {
                cached_feed
            };

            let next_cursor = if has_more {
                Some((offset + limit).to_string())
            } else {
                None
            };

            return Ok(PaginatedData {
                items,
                pagination: PaginationMeta {
                    has_more,
                    next_cursor,
                },
            });
        }

        // 缓存未命中，生成推荐内容
        let recommended_items = self.generate_recommendations(user_id, offset, limit).await?;

        // 缓存推荐结果
        self.cache_feed(user_id, &recommended_items).await?;

        let has_more = recommended_items.len() > limit as usize;
        let items: Vec<FeedItem> = if has_more {
            recommended_items.into_iter().take(limit as usize).collect()
        } else {
            recommended_items
        };

        let next_cursor = if has_more {
            Some((offset + limit).to_string())
        } else {
            None
        };

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 提交反馈
    pub async fn submit_feedback(
        &self,
        user_id: i64,
        req: FeedFeedbackRequest,
    ) -> Result<(), AppError> {
        // 验证反馈类型
        let valid_actions = vec!["hide", "not_interested", "clicked", "viewed"];
        if !valid_actions.contains(&req.action.as_str()) {
            return Err(AppError::InvalidInput(format!(
                "无效的反馈动作: {}",
                req.action
            )));
        }

        // 记录反馈到数据库（使用 FeedRepository）
        let item_uuid = uuid::Uuid::parse_str(&req.item_id)
            .map_err(|_| AppError::InvalidInput("无效的 item_id 格式".to_string()))?;
        
        self.feed_repo.record_feedback(user_id, item_uuid, &req.action).await?;

        // 更新Redis中的推荐权重
        self.update_recommendation_weights(user_id, &req).await?;

        // 如果是负面反馈，从缓存中移除该内容
        if req.action == "hide" || req.action == "not_interested" {
            self.remove_from_cached_feed(user_id, &req.item_id)
                .await?;
        }

        Ok(())
    }

    /// 从Redis缓存获取信息流
    async fn get_cached_feed(
        &self,
        user_id: i64,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<FeedItem>, AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:user:{}", user_id);

        // 获取缓存的内容ID列表
        let content_ids: Vec<String> = conn
            .lrange(&redis_key, offset as isize, (offset + limit) as isize)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        if content_ids.is_empty() {
            return Ok(Vec::new());
        }

        // 解析并收集所有 moment IDs
        let mut moment_ids = Vec::new();
        for content_id in &content_ids {
            let parts: Vec<&str> = content_id.split(':').collect();
            if parts.len() == 2 && parts[0] == "moment" {
                if let Ok(moment_id) = uuid::Uuid::parse_str(parts[1]) {
                    moment_ids.push(moment_id);
                }
            }
        }

        if moment_ids.is_empty() {
            return Ok(Vec::new());
        }

        // 批量获取瞬间信息
        let moments = self.moment_repo.find_by_ids(&moment_ids).await?;
        if moments.is_empty() {
            return Ok(Vec::new());
        }

        // 批量获取用户信息
        let user_ids: Vec<i64> = moments.iter().map(|m| m.user_id).collect();
        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 批量获取统计数据
        let stats = self.moment_repo.get_stats_batch(&moment_ids).await?;

        // 构建结果
        let items = moments.into_iter().filter_map(|moment| {
            let user = user_map.get(&moment.user_id)?;
            let (likes_count, comments_count) = stats.get(&moment.moment_id).copied().unwrap_or((0, 0));
            
            Some(FeedItem {
                item_id: format!("feed_item_{}", uuid::Uuid::new_v4()),
                r#type: "moment".to_string(),
                moment: Some(crate::dto::response::MomentInfo {
                    id: moment.moment_id.to_string(),
                    user: crate::dto::response::UserInfo {
                        user_id: user.user_id.to_string(),
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                    },
                    description: moment.description,
                    images: moment.images,
                    location_name: moment.location_name,
                    created_at: moment.created_at.to_rfc3339(),
                    likes_count: likes_count as i32,
                    comments_count: comments_count as i32,
                }),
                group: None,
                system: None,
                reason: "推荐".to_string(),
                score: Some(0.0),
            })
        }).collect();

        Ok(items)
    }

    /// 生成推荐内容
    async fn generate_recommendations(
        &self,
        _user_id: i64,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<FeedItem>, AppError> {
        // 使用优化后的批量查询，一次性获取瞬间、用户信息和统计数据
        let moments_with_stats = self.moment_repo
            .list_latest_with_user_and_stats(limit + 1, offset)
            .await
            .map_err(AppError::from)?;

        let items = moments_with_stats.into_iter().map(|m| {
            FeedItem {
                item_id: format!("feed_item_{}", uuid::Uuid::new_v4()),
                r#type: "moment".to_string(),
                moment: Some(crate::dto::response::MomentInfo {
                    id: m.moment.moment_id.to_string(),
                    user: crate::dto::response::UserInfo {
                        user_id: m.moment.user_id.to_string(),
                        nickname: m.nickname,
                        avatar_url: m.avatar_url,
                    },
                    description: m.moment.description,
                    images: m.moment.images,
                    location_name: m.moment.location_name,
                    created_at: m.moment.created_at.to_rfc3339(),
                    likes_count: m.likes_count as i32,
                    comments_count: m.comments_count as i32,
                }),
                group: None,
                system: None,
                reason: "推荐".to_string(),
                score: Some(1.0),
            }
        }).collect();

        Ok(items)
    }

    /// 缓存信息流
    async fn cache_feed(
        &self,
        user_id: i64,
        items: &[FeedItem],
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:user:{}", user_id);

        // 清空旧缓存
        let _: () = conn
            .del(&redis_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 缓存新内容
        for item in items {
            let content_id = format!("{}:{}", item.r#type, item.item_id);
            let _: () = conn
                .rpush(&redis_key, content_id)
                .await
                .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        }

        // 设置过期时间（1小时）
        let _: () = conn
            .expire(&redis_key, 3600)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }

    /// 更新推荐权重
    async fn update_recommendation_weights(
        &self,
        user_id: i64,
        req: &FeedFeedbackRequest,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:weights:user:{}", user_id);

        // 根据反馈类型调整权重
        let weight_change = match req.action.as_str() {
            "clicked" => 2.0,
            "viewed" => 0.5,
            "hide" => -1.0,
            "not_interested" => -2.0,
            _ => 0.0,
        };

        let content_key = &req.item_id;
        let _: () = redis::cmd("HINCRBYFLOAT")
            .arg(&redis_key)
            .arg(content_key)
            .arg(weight_change)
            .query_async(&mut conn)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }

    /// 从缓存中移除内容
    async fn remove_from_cached_feed(
        &self,
        user_id: i64,
        content_id: &str,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:user:{}", user_id);

        let _: () = conn
            .lrem(&redis_key, 0, content_id)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }

    /// 清空用户的信息流缓存（内部使用）
    pub async fn clear_feed_cache(&self, user_id: i64) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:user:{}", user_id);

        let _: () = conn
            .del(&redis_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }
}

