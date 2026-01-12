use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;
use std::str::FromStr;

use crate::{
    constants::{FEED_REASON_NEARBY_HOT, FEED_REASON_NEARBY_PEOPLE},
    dto::{
        request::{
            FeedFeedbackRequest, FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest,
            FeedUsersRequest,
        },
        response::{
            FeedGroupItem, FeedTrendingData, FeedUserItem, GroupItem, MomentItem, PaginatedData,
            PaginationMeta, RelativeLocation, TrendingTagItem, UserItem,
        },
    },
    error::AppError,
    repositories::{
        feed_repo::FeedRepository, group_repo::GroupRepository, moment_repo::MomentRepository,
        tag_repo::TagRepository, user_repo::UserRepository,
    },
    types::enums::{FeedFeedbackAction, FeedSource, Trend},
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
    presence_service: crate::services::presence_service::PresenceService,
}

impl FeedService {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self {
            redis: redis.clone(),
            feed_repo: FeedRepository::new(pool.clone()),
            moment_repo: MomentRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool.clone()),
            group_repo: GroupRepository::new(pool.clone()),
            tag_repo: TagRepository::new(pool.clone()),
            presence_service: crate::services::presence_service::PresenceService::new(
                redis,
                pool,
            ),
        }
    }

    /// 获取瞬间信息流（距离由 PostGIS 计算）
    pub async fn get_moments_feed(
        &self,
        user_id: i64,
        req: FeedMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor;

        // 获取用户位置（优先从 Redis 获取）
        let user_location = self.presence_service.get_user_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));

        // 根据 source 筛选（使用新的 Feed 方法，返回带距离的结果）
        let source_str = req.source.as_deref().unwrap_or("all");
        let source = FeedSource::from_str(source_str).unwrap_or(FeedSource::All);

        let (moments, next_cursor) = match source {
            FeedSource::Following => {
                self.moment_repo
                    .get_following_feed(user_id, user_lat, user_lng, limit, cursor)
                    .await?
            }
            FeedSource::Nearby => {
                let radius = Validator::validate_radius(req.radius, 50000)? as f64;
                self.moment_repo
                    .get_nearby_feed(user_lat, user_lng, radius, limit, cursor)
                    .await?
            }
            _ => {
                self.moment_repo
                    .list_latest_feed(user_lat, user_lng, limit, cursor)
                    .await?
            }
        };

        // 批量获取标签
        let moment_ids: Vec<_> = moments.iter().map(|m| m.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;

        use crate::dto::response::MomentImage;
        let items: Vec<MomentItem> = moments
            .into_iter()
            .map(|m| {
                let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
                let images = if m.images.is_empty() {
                    None
                } else {
                    Some(
                        m.images
                            .iter()
                            .map(|url| MomentImage {
                                url: url.clone(),
                                width: None,
                                height: None,
                                content_type: None,
                            })
                            .collect(),
                    )
                };

                MomentItem {
                    moment_id: m.moment_id.to_string(),
                    user: crate::dto::response::UserInfo {
                        user_id: m.user_id,
                        nickname: m.nickname,
                        avatar_url: m.avatar_url,
                        user_type: m.user_type,
                    },
                    description: m.description,
                    images,
                    tags,
                    created_at: m.created_at.to_rfc3339(),
                    likes_count: m.likes_count as i32,
                    comments_count: m.comments_count as i32,
                    location: Some(RelativeLocation {
                        latitude: m.latitude,
                        longitude: m.longitude,
                        location_name: m.location_name,
                        distance: m.distance as i32,
                        bearing_degrees: m.bearing_degrees,
                    }),
                    is_liked: None,
                    is_favorited: None,
                }
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

    /// 获取群组推荐（距离由 PostGIS 计算）
    pub async fn get_groups_feed(
        &self,
        user_id: i64,
        req: FeedGroupsRequest,
    ) -> Result<PaginatedData<FeedGroupItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor;

        // 获取用户位置
        let user_location = self.user_repo.get_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));
        let radius = Validator::validate_radius(req.radius, 50000)? as f64;

        // 获取附近群组（使用 PostGIS 计算距离）
        let (groups, next_cursor) = self
            .group_repo
            .search_by_location(None, user_lat, user_lng, radius, limit, cursor)
            .await?;

        let items: Vec<FeedGroupItem> = groups
            .into_iter()
            .map(|g| FeedGroupItem {
                group: GroupItem {
                    group_chat_id: g.group_id.to_string(),
                    name: g.name,
                    description: g.description,
                    avatar_url: g.avatar_url,
                    tags: Some(g.tags),
                    members_count: g.member_count,
                    has_password: g.has_password,
                    location: Some(RelativeLocation {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance as i32,
                        bearing_degrees: g.bearing_degrees,
                    }),
                },
                reason: Some(FEED_REASON_NEARBY_HOT.to_string()),
                friends_in_group: None,
                my_role: None,
                is_joined: None,
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

    /// 获取用户推荐（距离由 PostGIS 计算）
    pub async fn get_users_feed(
        &self,
        user_id: i64,
        req: FeedUsersRequest,
    ) -> Result<PaginatedData<FeedUserItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32; // Using i32 for repo method
        let cursor = req.cursor;

        // 获取用户位置
        let user_location = self.user_repo.get_location(user_id).await?;
        let (user_lat, user_lng) = user_location.unwrap_or((35.154922, 112.682524));
        let radius = Validator::validate_radius(req.radius, 50000)? as f64;

        // 获取附近用户（使用 PostGIS 计算距离）
        // UserRepo::search_for_feed limit expects i64
        let (users, next_cursor) = self
            .user_repo
            .search_for_feed(
                user_lat,
                user_lng,
                radius,
                limit as i64,
                cursor,
                Some(user_id),
            )
            .await?;

        // 批量获取用户 ID
        let user_ids: Vec<i64> = users.iter().map(|u| u.user_id).collect();

        // 批量获取在线状态（受隐私设置控制）
        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&user_ids)
            .await;

        // 批量获取位置可见性设置
        let location_visible_map = self
            .presence_service
            .batch_get_location_visible(&user_ids)
            .await;

        let items: Vec<FeedUserItem> = users
            .into_iter()
            .map(|u| {
                let (is_online, last_online) = online_status_map
                    .get(&u.user_id)
                    .cloned()
                    .unwrap_or((None, None));

                // 检查位置是否可见
                let location_visible = location_visible_map
                    .get(&u.user_id)
                    .copied()
                    .unwrap_or(true);

                let location = if location_visible {
                    Some(RelativeLocation {
                        latitude: u.latitude,
                        longitude: u.longitude,
                        location_name: u.location_name,
                        distance: u.distance as i32,
                        bearing_degrees: u.bearing_degrees,
                    })
                } else {
                    None
                };

                FeedUserItem {
                    user: UserItem {
                        user_id: u.user_id,
                        nickname: u.nickname,
                        avatar_url: u.avatar_url,
                        bio: u.bio,
                        tags: Some(u.tags),
                        is_online,
                        last_online,
                        user_type: u.user_type,
                        location,
                        relation: None,
                    },
                    reason: Some(FEED_REASON_NEARBY_PEOPLE.to_string()),
                    mutual_friends_count: None,
                    common_tags: None,
                }
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

    /// 获取热门内容
    pub async fn get_trending(
        &self,
        _user_id: i64,
        req: FeedTrendingRequest,
    ) -> Result<FeedTrendingData, AppError> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let since =
            chrono::Utc::now() - chrono::Duration::days(crate::constants::TRENDING_WEEKLY_DAYS);

        // 获取热门标签
        let tags = self.tag_repo.get_trending(limit, since).await?;

        let trending_tags: Vec<TrendingTagItem> = tags
            .into_iter()
            .map(|t| TrendingTagItem {
                name: t.name,
                usage_count: t.usage_count,
                trend: Trend::Stable.to_string(),
                hot_moment_preview: None,
            })
            .collect();

        // 获取热门瞬间（最多5个）
        let (hot_moments_raw, _) = self.moment_repo.get_hot_moments(5, since, None).await?;

        // 批量获取标签
        let hot_moment_ids: Vec<_> = hot_moments_raw.iter().map(|m| m.moment_id).collect();
        let hot_tags_map = self
            .tag_repo
            .get_moment_tags_batch(&hot_moment_ids)
            .await?;

        use crate::dto::response::MomentImage;
        let hot_moments: Vec<MomentItem> = hot_moments_raw
            .into_iter()
            .map(|m| {
                let tags = hot_tags_map.get(&m.moment_id).cloned().unwrap_or_default();
                let images = if m.images.is_empty() {
                    None
                } else {
                    Some(
                        m.images
                            .iter()
                            .map(|url| MomentImage {
                                url: url.clone(),
                                width: None,
                                height: None,
                                content_type: None,
                            })
                            .collect(),
                    )
                };

                MomentItem {
                    moment_id: m.moment_id.to_string(),
                    user: crate::dto::response::UserInfo {
                        user_id: m.user_id,
                        nickname: m.nickname,
                        avatar_url: m.avatar_url,
                        user_type: m.user_type,
                    },
                    description: m.description,
                    images,
                    tags,
                    created_at: m.created_at.to_rfc3339(),
                    likes_count: m.likes_count as i32,
                    comments_count: m.comments_count as i32,
                    location: Some(RelativeLocation {
                        latitude: m.latitude,
                        longitude: m.longitude,
                        location_name: m.location_name,
                        distance: m.distance as i32,
                        bearing_degrees: m.bearing_degrees,
                    }),
                    is_liked: None,
                    is_favorited: None,
                }
            })
            .collect();

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

        // 验证 action
        let action = FeedFeedbackAction::from_str(&req.action)
            .map_err(|_| AppError::InvalidInput("无效的反馈操作".to_string()))?;

        // 可选原因：用于输入约束（并可作为后续审计字段扩展）
        let reason = req
            .reason
            .as_deref()
            .map(str::trim)
            .filter(|s| !s.is_empty());
        if let Some(r) = reason {
            if r.len() > 200 {
                return Err(AppError::InvalidInput("反馈原因过长".to_string()));
            }
        }
        if matches!(action, FeedFeedbackAction::Report) && reason.is_none() {
            return Err(AppError::InvalidInput("举报需要填写原因".to_string()));
        }

        // 记录反馈到数据库
        self.feed_repo
            .record_feedback(user_id, content_uuid, action.as_str())
            .await?;

        // 更新Redis中的推荐权重
        self.update_recommendation_weights(user_id, &req, action)
            .await?;

        // 如果是负面反馈，从缓存中移除该内容
        if action.should_remove_from_feed() {
            self.remove_from_cached_feed(user_id, &req.content_id)
                .await?;
        }

        Ok(())
    }

    /// 更新推荐权重
    async fn update_recommendation_weights(
        &self,
        user_id: i64,
        req: &FeedFeedbackRequest,
        action: FeedFeedbackAction,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("feed:weights:user:{}", user_id);

        // 根据反馈类型调整权重
        let weight_change = match action {
            FeedFeedbackAction::Hide => -1.0,
            FeedFeedbackAction::NotInterested => -2.0,
            FeedFeedbackAction::Report => -3.0,
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
