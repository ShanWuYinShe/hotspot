use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetFeedRequest, SubmitFeedbackRequest},
        response::{FeedItemResponse, PaginatedResponse},
    },
    error::AppError,
    repositories::{
        moment_repo::MomentRepository,
        user_repo::UserRepository,
    },
};

#[derive(Clone)]
pub struct FeedService {
    db: PgPool,
    redis: ConnectionManager,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
}

impl FeedService {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self {
            db: pool.clone(),
            redis,
            moment_repo: MomentRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool),
        }
    }

    /// 获取个性化信息流
    pub async fn get_feed(
        &self,
        user_id: i64,
        req: GetFeedRequest,
    ) -> Result<PaginatedResponse<FeedItemResponse>, AppError> {
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
            let items: Vec<FeedItemResponse> = if has_more {
                cached_feed.into_iter().take(limit as usize).collect()
            } else {
                cached_feed
            };

            let next_cursor = if has_more {
                Some((offset + limit).to_string())
            } else {
                None
            };

            return Ok(PaginatedResponse {
                items,
                has_more,
                next_cursor,
            });
        }

        // 缓存未命中，生成推荐内容
        let recommended_items = self.generate_recommendations(user_id, offset, limit).await?;

        // 缓存推荐结果
        self.cache_feed(user_id, &recommended_items).await?;

        let has_more = recommended_items.len() > limit as usize;
        let items: Vec<FeedItemResponse> = if has_more {
            recommended_items.into_iter().take(limit as usize).collect()
        } else {
            recommended_items
        };

        let next_cursor = if has_more {
            Some((offset + limit).to_string())
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 提交反馈
    pub async fn submit_feedback(
        &self,
        user_id: i64,
        req: SubmitFeedbackRequest,
    ) -> Result<(), AppError> {
        // 验证反馈类型
        let valid_types = vec!["like", "dislike", "skip", "view", "share"];
        if !valid_types.contains(&req.feedback_type.as_str()) {
            return Err(AppError::InvalidInput(format!(
                "无效的反馈类型: {}",
                req.feedback_type
            )));
        }

        // 记录反馈到数据库（使用feed_feedback表）
        sqlx::query!(
            r#"
            INSERT INTO feed_feedback (user_id, item_type, item_id, action, created_at)
            VALUES ($1, $2, $3, $4, NOW())
            "#,
            user_id,
            req.content_type,
            req.content_id,
            req.feedback_type
        )
        .execute(&self.db)
        .await
        .map_err(AppError::from)?;

        // 更新Redis中的推荐权重
        self.update_recommendation_weights(user_id, &req).await?;

        // 如果是负面反馈，从缓存中移除该内容
        if req.feedback_type == "dislike" || req.feedback_type == "skip" {
            self.remove_from_cached_feed(user_id, &req.content_id)
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
    ) -> Result<Vec<FeedItemResponse>, AppError> {
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

        // 根据内容ID获取详细信息
        let mut items = Vec::new();
        for content_id in content_ids {
            // 解析content_id格式：type:id
            let parts: Vec<&str> = content_id.split(':').collect();
            if parts.len() != 2 {
                continue;
            }

            let content_type = parts[0];
            let id = parts[1];

            match content_type {
                "moment" => {
                    if let Ok(moment_id) = uuid::Uuid::parse_str(id) {
                        if let Ok(Some(moment)) = self.moment_repo.find_by_id(moment_id).await {
                            items.push(FeedItemResponse {
                                item_id: format!("feed_item_{}", uuid::Uuid::new_v4()),
                                content_type: "moment".to_string(),
                                content_id: moment.moment_id.to_string(),
                                title: None,
                                description: moment.description.clone(),
                                author_id: moment.user_id,
                                author_name: "".to_string(), // 需要查询用户信息
                                created_at: moment.created_at,
                                score: 0.0, // 推荐分数
                            });
                        }
                    }
                }
                _ => continue,
            }
        }

        Ok(items)
    }

    /// 生成推荐内容
    async fn generate_recommendations(
        &self,
        user_id: i64,
        offset: i64,
        limit: i64,
    ) -> Result<Vec<FeedItemResponse>, AppError> {
        // 简单的推荐算法：
        // 1. 获取用户关注的人的最新瞬间
        // 2. 获取热门瞬间
        // 3. 获取附近的瞬间
        // 4. 混合排序

        // 获取最新的瞬间（简化实现）
        use crate::models::moment::Moment;
        let moments = sqlx::query_as!(
            Moment,
            r#"
            SELECT moment_id, user_id, description, latitude, longitude,
                   location_name, tags, images, created_at, updated_at,
                   is_deleted, deleted_at
            FROM moments
            WHERE is_deleted = FALSE
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#,
            (limit + 1) as i64,
            offset as i64
        )
        .fetch_all(&self.db)
        .await
        .map_err(AppError::from)?;

        let mut items = Vec::new();
        for moment in moments {
            // 获取作者信息
            let author = self.user_repo.find_by_id(moment.user_id).await?;
            let author_name = author.map(|u| u.nickname).unwrap_or_default();

            items.push(FeedItemResponse {
                item_id: format!("feed_item_{}", uuid::Uuid::new_v4()),
                content_type: "moment".to_string(),
                content_id: moment.moment_id.to_string(),
                title: None,
                description: moment.description.clone(),
                author_id: moment.user_id,
                author_name,
                created_at: moment.created_at,
                score: 1.0, // 基础分数
            });
        }

        Ok(items)
    }

    /// 缓存信息流
    async fn cache_feed(
        &self,
        user_id: i64,
        items: &[FeedItemResponse],
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
            let content_id = format!("{}:{}", item.content_type, item.content_id);
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
        req: &SubmitFeedbackRequest,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:weights:user:{}", user_id);

        // 根据反馈类型调整权重
        let weight_change = match req.feedback_type.as_str() {
            "like" => 2.0,
            "share" => 3.0,
            "view" => 0.5,
            "skip" => -1.0,
            "dislike" => -2.0,
            _ => 0.0,
        };

        let content_key = format!("{}:{}", req.content_type, req.content_id);
        redis::cmd("HINCRBYFLOAT")
            .arg(&redis_key)
            .arg(&content_key)
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

