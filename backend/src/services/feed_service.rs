use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetFeedMomentsRequest, GetFeedGroupsRequest, GetFeedUsersRequest, GetFeedTrendingRequest, FeedFeedbackRequest},
        response::{FeedMomentItem, FeedGroupItem, FeedUserItem, FeedTrendingData, TrendingTagItem, PaginatedData, PaginationMeta, MomentSearchItem, GroupSearchItem, UserSearchItem, GeoData},
    },
    error::AppError,
    repositories::{
        feed_repo::FeedRepository,
        moment_repo::MomentRepository,
        user_repo::UserRepository,
        group_repo::GroupRepository,
        tag_repo::TagRepository,
    },
    utils::validation::Validator,
};

#[derive(Clone)]
pub struct FeedService {
    redis: ConnectionManager,
    feed_repo: FeedRepository,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    group_repo: GroupRepository,
    tag_repo: TagRepository,
}

impl FeedService {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self {
            redis,
            feed_repo: FeedRepository::new(pool.clone()),
            moment_repo: MomentRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool.clone()),
            group_repo: GroupRepository::new(pool.clone()),
            tag_repo: TagRepository::new(pool),
        }
    }

    /// 获取瞬间信息流（距离由 PostGIS 计算）
    pub async fn get_moments_feed(
        &self,
        user_id: i64,
        req: GetFeedMomentsRequest,
    ) -> Result<PaginatedData<FeedMomentItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let offset = Validator::parse_offset(req.cursor.as_deref())?;

        // 获取用户位置（用于计算距离）
        let user_location = self.user_repo.get_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));

        // 根据 source 筛选（使用新的 Feed 方法，返回带距离的结果）
        let source = req.source.as_deref().unwrap_or("all");
        
        let moments = match source {
            "following" => {
                self.moment_repo.get_following_feed(user_id, user_lat, user_lng, limit + 1, offset).await?
            }
            "nearby" => {
                let radius = Validator::validate_radius(req.radius, 50000)? as f64;
                self.moment_repo.get_nearby_feed(user_lat, user_lng, radius, limit + 1, offset).await?
            }
            _ => {
                self.moment_repo.list_latest_feed(user_lat, user_lng, limit + 1, offset).await?
            }
        };

        let has_more = moments.len() > limit as usize;
        let items: Vec<FeedMomentItem> = moments.into_iter()
            .take(limit as usize)
            .map(|m| {
                FeedMomentItem {
                    moment: MomentSearchItem {
                        id: m.moment_id.to_string(),
                        user: crate::dto::response::UserInfo {
                            user_id: m.user_id,
                            nickname: m.nickname,
                            avatar_url: m.avatar_url,
                            user_type: m.user_type,
                        },
                        description: m.description,
                        images: Some(m.images),
                        tags: vec![], // TODO: 从 moment_tags 获取
                        created_at: m.created_at.to_rfc3339(),
                        likes_count: m.likes_count as i32,
                        comments_count: m.comments_count as i32,
                        geo: GeoData {
                            latitude: m.latitude,
                            longitude: m.longitude,
                            location_name: m.location_name,
                            distance: m.distance as i32,
                            degrees: m.degrees,
                        },
                    },
                    reason: Some(match source {
                        "following" => "好友动态".to_string(),
                        "nearby" => "附近热门".to_string(),
                        _ => "为你推荐".to_string(),
                    }),
                }
            })
            .collect();

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

    /// 获取群组推荐（距离由 PostGIS 计算）
    pub async fn get_groups_feed(
        &self,
        user_id: i64,
        req: GetFeedGroupsRequest,
    ) -> Result<PaginatedData<FeedGroupItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let offset = Validator::parse_offset(req.cursor.as_deref())?;

        // 获取用户位置
        let user_location = self.user_repo.get_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));
        let radius = Validator::validate_radius(req.radius, 50000)? as f64;

        // 获取附近群组（使用 PostGIS 计算距离）
        let groups = self.group_repo.search_by_location(user_lat, user_lng, radius, limit + 1, offset).await?;

        let has_more = groups.len() > limit as usize;
        let items: Vec<FeedGroupItem> = groups.into_iter()
            .take(limit as usize)
            .map(|g| {
                FeedGroupItem {
                    group: GroupSearchItem {
                        id: g.group_id.to_string(),
                        name: g.name,
                        description: g.description,
                        avatar_url: g.avatar_url,
                        tags: Some(g.tags),
                        members_count: g.member_count,
                        has_password: g.has_password,
                        geo: GeoData {
                            latitude: g.latitude,
                            longitude: g.longitude,
                            location_name: g.location_name,
                            distance: g.distance as i32,
                            degrees: g.degrees,
                        },
                    },
                    reason: Some("附近热门".to_string()),
                    friends_in_group: None, // TODO: 计算群内好友数
                }
            })
            .collect();

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

    /// 获取用户推荐（距离由 PostGIS 计算）
    pub async fn get_users_feed(
        &self,
        user_id: i64,
        req: GetFeedUsersRequest,
    ) -> Result<PaginatedData<FeedUserItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let offset = Validator::parse_offset(req.cursor.as_deref())?;

        // 获取用户位置
        let user_location = self.user_repo.get_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));
        let radius = Validator::validate_radius(req.radius, 50000)? as f64;

        // 获取附近用户（使用 PostGIS 计算距离）
        let users = self.user_repo.search_for_feed(user_lat, user_lng, radius, limit + 1, offset, Some(user_id)).await?;

        let has_more = users.len() > limit as usize;
        let items: Vec<FeedUserItem> = users.into_iter()
            .take(limit as usize)
            .map(|u| {
                FeedUserItem {
                    user: UserSearchItem {
                        user_id: u.user_id,
                        nickname: u.nickname,
                        avatar_url: u.avatar_url,
                        bio: u.bio,
                        tags: Some(u.tags),
                        geo: GeoData {
                            latitude: u.latitude,
                            longitude: u.longitude,
                            location_name: u.location_name,
                            distance: u.distance as i32,
                            degrees: u.degrees,
                        },
                    },
                    reason: Some("附近的人".to_string()),
                    mutual_friends_count: None, // TODO: 计算共同好友数
                    common_tags: None, // TODO: 计算共同标签
                }
            })
            .collect();

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

    /// 获取热门内容
    pub async fn get_trending(
        &self,
        _user_id: i64,
        req: GetFeedTrendingRequest,
    ) -> Result<FeedTrendingData, AppError> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let since = chrono::Utc::now() - chrono::Duration::days(crate::constants::TRENDING_WEEKLY_DAYS);

        // 获取热门标签
        let tags = self.tag_repo.get_trending(limit, since).await?;

        let trending_tags: Vec<TrendingTagItem> = tags.into_iter().map(|t| {
            TrendingTagItem {
                name: t.name,
                usage_count: t.usage_count,
                trend: "stable".to_string(), // TODO: 计算趋势
                hot_moment_preview: None, // TODO: 获取该标签下的热门瞬间
            }
        }).collect();

        // 获取热门瞬间（最多5个）
        let hot_moments_raw = self.moment_repo.get_hot_moments(5, since).await?;
        let hot_moments: Vec<MomentSearchItem> = hot_moments_raw.into_iter().map(|m| {
            MomentSearchItem {
                id: m.moment_id.to_string(),
                user: crate::dto::response::UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url: m.avatar_url,
                    user_type: m.user_type,
                },
                description: m.description,
                images: Some(m.images),
                tags: vec![], // TODO: 从 moment_tags 获取
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                geo: GeoData {
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    distance: m.distance as i32,
                    degrees: m.degrees,
                },
            }
        }).collect();

        Ok(FeedTrendingData {
            trending_tags,
            hot_moments,
        })
    }

    /// 提交反馈
    pub async fn submit_feedback(
        &self,
        user_id: i64,
        req: FeedFeedbackRequest,
    ) -> Result<(), AppError> {
        // 验证 content_id 格式
        let content_uuid = uuid::Uuid::parse_str(&req.content_id)
            .map_err(|_| AppError::InvalidInput("无效的 content_id 格式".to_string()))?;
        
        // 记录反馈到数据库
        self.feed_repo.record_feedback(user_id, content_uuid, &req.action).await?;

        // 更新Redis中的推荐权重
        self.update_recommendation_weights(user_id, &req).await?;

        // 如果是负面反馈，从缓存中移除该内容
        if req.action == "hide" || req.action == "not_interested" {
            self.remove_from_cached_feed(user_id, &req.content_id).await?;
        }

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
            "hide" => -1.0,
            "not_interested" => -2.0,
            "report" => -3.0,
            _ => 0.0,
        };

        let content_key = format!("{}:{}", req.content_type, req.content_id);
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
}
