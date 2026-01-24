use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use crate::{
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
    types::enums::{FeedFeedbackAction, FeedSource, TagTrend},
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
    media_service: crate::services::media_service::MediaService,
}

impl FeedService {
    pub fn new(
        redis: ConnectionManager,
        feed_repo: FeedRepository,
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        group_repo: GroupRepository,
        tag_repo: TagRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            redis,
            feed_repo,
            moment_repo,
            user_repo,
            group_repo,
            tag_repo,
            presence_service,
            media_service,
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

        // 根据 source 筛选（使用新的 Feed 方法，返回带距离的结果）
        let source = req.source.unwrap_or(FeedSource::All);
        let reference_point = req.reference_point;

        let (user_lat, user_lng) = if let Some(point) = reference_point {
            Validator::validate_coordinates(point.latitude, point.longitude)?;
            (point.latitude, point.longitude)
        } else if matches!(source, FeedSource::Following) {
            // 获取用户位置（优先从 Redis 获取）
            let user_location = self.presence_service.get_user_location(user_id).await?;
            user_location.unwrap_or((35.154922, 112.682524))
        } else {
            return Err(AppError::MissingParameter("reference_point".to_string()));
        };

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

        // 批量查询点赞和收藏状态
        let liked_set = self
            .moment_repo
            .batch_check_liked(&moment_ids, user_id)
            .await?;
        let favorited_set = self
            .moment_repo
            .batch_check_favorited(&moment_ids, user_id)
            .await?;

        use crate::dto::response::MomentImage;
        let mut items = Vec::with_capacity(moments.len());
        for m in moments {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();

            // 动态签名 image paths -> fresh URLs
            let images = if m.images.is_empty() {
                None
            } else {
                let mut signed_images = Vec::with_capacity(m.images.len());
                for path in &m.images {
                    let url = match self.media_service.sign_url(path).await {
                        Ok(u) => u,
                        Err(e) => {
                            tracing::warn!("Failed to sign url for path {}: {}", path, e);
                            path.clone() // Fallback to path if signing fails
                        }
                    };
                    signed_images.push(MomentImage {
                        url,
                        width: None,
                        height: None,
                        content_type: None,
                    });
                }
                Some(signed_images)
            };

            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;

            items.push(MomentItem {
                moment_id: m.moment_id.to_string(),
                user: crate::dto::response::UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url,
                    user_type: m.user_type,
                },
                description: m.description,
                images,
                tags,
                mentions: m.mentions.clone(),
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                location: RelativeLocation {
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    distance: m.distance,
                    bearing_degrees: m.bearing_degrees,
                },
                is_liked: Some(liked_set.contains(&m.moment_id)),
                is_favorited: Some(favorited_set.contains(&m.moment_id)),
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

    /// 获取群组推荐（距离由 PostGIS 计算）
    pub async fn get_groups_feed(
        &self,
        _user_id: i64,
        req: FeedGroupsRequest,
    ) -> Result<PaginatedData<FeedGroupItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor;

        // 使用请求传入的 reference_point
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let (user_lat, user_lng) = (req.reference_point.latitude, req.reference_point.longitude);
        let radius = Validator::validate_radius(req.radius, 50000)? as f64;

        // 获取附近群组（使用 PostGIS 计算距离）
        let (groups, next_cursor) = self
            .group_repo
            .search_by_location(None, user_lat, user_lng, radius, limit, cursor)
            .await?;

        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(g.avatar_url.as_deref())
                .await;

            items.push(FeedGroupItem {
                group: GroupItem {
                    group_chat_id: g.group_id.to_string(),
                    name: g.name,
                    description: if g.description.is_empty() {
                        None
                    } else {
                        Some(g.description)
                    },
                    avatar_url,
                    tags: Some(g.tags),
                    members_count: g.member_count,
                    has_password: g.has_password,
                    location: Some(RelativeLocation {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance,
                        bearing_degrees: g.bearing_degrees,
                    }),
                },
                friends_in_group: None,
                my_role: None,
                is_joined: None,
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

    /// 获取用户推荐（距离由 PostGIS 计算）
    pub async fn get_users_feed(
        &self,
        user_id: i64,
        req: FeedUsersRequest,
    ) -> Result<PaginatedData<FeedUserItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32; // Using i32 for repo method
        let cursor = req.cursor;

        // 使用请求传入的 reference_point
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let (user_lat, user_lng) = (req.reference_point.latitude, req.reference_point.longitude);
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

        let mut items = Vec::with_capacity(users.len());
        for u in users {
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
                    distance: u.distance,
                    bearing_degrees: u.bearing_degrees,
                })
            } else {
                None
            };

            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(u.avatar_url.as_deref())
                .await;

            items.push(FeedUserItem {
                user: UserItem {
                    user_id: u.user_id,
                    nickname: u.nickname,
                    avatar_url,
                    bio: if u.bio.is_empty() { None } else { Some(u.bio) },
                    tags: Some(u.tags),
                    is_online,
                    last_online,
                    user_type: u.user_type,
                    location,
                    relation: None,
                },
                mutual_friends_count: None,
                common_tags: None,
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
                trend: TagTrend::Stable,
                hot_moment_preview: None,
            })
            .collect();

        // 获取热门瞬间（最多5个）
        let (hot_moments_raw, _) = self.moment_repo.get_hot_moments(5, since, None).await?;

        // 批量获取标签
        let hot_moment_ids: Vec<_> = hot_moments_raw.iter().map(|m| m.moment_id).collect();
        let hot_tags_map = self.tag_repo.get_moment_tags_batch(&hot_moment_ids).await?;

        use crate::dto::response::MomentImage;
        let mut hot_moments = Vec::with_capacity(hot_moments_raw.len());
        for m in hot_moments_raw {
            let tags = hot_tags_map.get(&m.moment_id).cloned().unwrap_or_default();

            // 签名图片 URLs
            let images = if m.images.is_empty() {
                None
            } else {
                let mut signed_images = Vec::with_capacity(m.images.len());
                for path in &m.images {
                    let url = match self.media_service.sign_url(path).await {
                        Ok(u) => u,
                        Err(e) => {
                            tracing::warn!("Failed to sign url for path {}: {}", path, e);
                            path.clone()
                        }
                    };
                    signed_images.push(MomentImage {
                        url,
                        width: None,
                        height: None,
                        content_type: None,
                    });
                }
                Some(signed_images)
            };

            // 签名用户头像
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;

            hot_moments.push(MomentItem {
                moment_id: m.moment_id.to_string(),
                user: crate::dto::response::UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url,
                    user_type: m.user_type,
                },
                description: m.description,
                images,
                tags,
                mentions: m.mentions.clone(),
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                location: RelativeLocation {
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    distance: m.distance,
                    bearing_degrees: m.bearing_degrees,
                },
                is_liked: None,
                is_favorited: None,
            });
        }

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
            .map_err(|_| AppError::InvalidFormat("无效的 content_id 格式".to_string()))?;

        // action 已是强类型枚举，直接使用
        let action = req.action;

        // 可选原因：用于输入约束（并可作为后续审计字段扩展）
        let reason = req
            .reason
            .as_deref()
            .map(str::trim)
            .filter(|s| !s.is_empty());
        if let Some(r) = reason {
            if r.len() > 200 {
                return Err(AppError::InvalidFormat("反馈原因过长".to_string()));
            }
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
