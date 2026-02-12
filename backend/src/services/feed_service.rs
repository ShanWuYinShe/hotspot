use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE},
    dto::{
        request::{FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest, FeedUsersRequest},
        response::{
            FeedGroupItem, FeedMomentItem, FeedTrendingData, FeedUserItem, PaginatedData,
            PaginationMeta, TrendingTagItem, UserSearchItem,
        },
    },
    error::AppError,
    repositories::{
        group_repo::GroupRepository, moment_repo::MomentRepository, tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    types::enums::{FeedSource, TagTrend},
    utils::validation::Validator,
};

#[derive(Clone)]
pub struct FeedService {
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    group_repo: GroupRepository,
    tag_repo: TagRepository,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
}

impl FeedService {
    pub fn new(
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        group_repo: GroupRepository,
        tag_repo: TagRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
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
    ) -> Result<PaginatedData<FeedMomentItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor;

        // 根据 source 筛选（使用新的 Feed 方法，返回带距离的结果）
        let source = req.source.unwrap_or(FeedSource::All);
        let reference_point = req.reference_point;

        let (user_lat, user_lng) = if let Some(point) = reference_point {
            Validator::validate_coordinates(point.latitude, point.longitude)?;
            (point.latitude, point.longitude)
        } else if matches!(source, FeedSource::Following) {
            // 获取用户位置（优先从 Redis 获取），使用常量作为默认值
            let user_location = self.presence_service.get_user_location(user_id).await?;
            user_location.unwrap_or((DEFAULT_LATITUDE, DEFAULT_LONGITUDE))
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

        let mut items = Vec::with_capacity(moments.len());
        for m in moments {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();

            let images = self.sign_moment_images(&m.images).await;

            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;

            items.push(FeedMomentItem {
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
                location: crate::dto::response::RelativeLocation {
                    point: crate::dto::response::GeoPoint {
                        latitude: m.latitude.unwrap_or(0.0),
                        longitude: m.longitude.unwrap_or(0.0),
                    },
                    location_name: m.location_name.unwrap_or_default(),
                    distance: m.distance,
                    bearing_degrees: m.bearing_degrees,
                },
                is_liked: liked_set.contains(&m.moment_id),
                is_favorited: favorited_set.contains(&m.moment_id),
                recommendation: None, // TODO: 实现推荐理由计算逻辑
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
            .search_by_location(None, user_lng, user_lat, radius, limit, cursor)
            .await?;

        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(g.avatar_url.as_deref())
                .await;

            items.push(FeedGroupItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                avatar_url,
                members_count: g.member_count,
                description: if g.description.is_empty() {
                    None
                } else {
                    Some(g.description)
                },
                tags: if g.tags.is_empty() {
                    None
                } else {
                    Some(g.tags)
                },
                has_password: g.has_password,
                location: crate::dto::response::RelativeLocation {
                    point: crate::dto::response::GeoPoint {
                        latitude: g.latitude,
                        longitude: g.longitude,
                    },
                    location_name: g.location_name,
                    distance: Some(g.distance),
                    bearing_degrees: Some(g.bearing_degrees),
                },
                friends_in_group: None, // TODO: Implement social graph logic to populate this
                my_state: None,         // TODO: Implement social graph logic to populate this
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
                user_lng,
                user_lat,
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
                Some(crate::dto::response::RelativeLocation {
                    point: crate::dto::response::GeoPoint {
                        latitude: u.latitude,
                        longitude: u.longitude,
                    },
                    location_name: u.location_name,
                    distance: Some(u.distance),
                    bearing_degrees: Some(u.bearing_degrees),
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
                user: UserSearchItem {
                    user_id: u.user_id,
                    nickname: u.nickname,
                    avatar_url,
                    bio: if u.bio.is_empty() { None } else { Some(u.bio) },
                    tags: Some(u.tags),
                    is_online,
                    last_online,
                    user_type: u.user_type,
                    location,
                    stats: None,
                    relation: None,
                    social_context: None,
                },
                mutual_friends_count: None, // TODO: Implement social graph logic to populate this
                common_tags: None,          // TODO: Implement tag matching logic to populate this
                relation: None, // TODO: Implement relationship status check to populate this
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
        user_id: i64,
        req: FeedTrendingRequest,
    ) -> Result<FeedTrendingData, AppError> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let since =
            chrono::Utc::now() - chrono::Duration::days(crate::constants::TRENDING_WEEKLY_DAYS);

        // 获取热门标签
        let tags = self.tag_repo.get_trending(limit).await?;

        let trending_tags: Vec<TrendingTagItem> = tags
            .into_iter()
            .map(|t| TrendingTagItem {
                name: t.name,
                usage_count: t.usage_count.unwrap_or(0),
                trend: TagTrend::Stable,
                hot_moment_preview: None,
            })
            .collect();

        // 获取热门瞬间（最多5个）
        let (hot_moments_raw, _) = self.moment_repo.get_hot_moments(5, since, None).await?;

        // 批量获取标签
        let hot_moment_ids: Vec<_> = hot_moments_raw.iter().map(|m| m.moment_id).collect();
        let hot_tags_map = self.tag_repo.get_moment_tags_batch(&hot_moment_ids).await?;

        let mut hot_moments = Vec::with_capacity(hot_moments_raw.len());
        let liked_set = self
            .moment_repo
            .batch_check_liked(&hot_moment_ids, user_id)
            .await?;
        let favorited_set = self
            .moment_repo
            .batch_check_favorited(&hot_moment_ids, user_id)
            .await?;
        for m in hot_moments_raw {
            let tags = hot_tags_map.get(&m.moment_id).cloned().unwrap_or_default();

            let images = self.sign_moment_images(&m.images).await;

            // 签名用户头像
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;

            hot_moments.push(FeedMomentItem {
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
                location: crate::dto::response::RelativeLocation {
                    point: crate::dto::response::GeoPoint {
                        latitude: m.latitude.unwrap_or(0.0),
                        longitude: m.longitude.unwrap_or(0.0),
                    },
                    location_name: m.location_name.unwrap_or_default(),
                    distance: m.distance,
                    bearing_degrees: m.bearing_degrees,
                },
                is_liked: liked_set.contains(&m.moment_id),
                is_favorited: favorited_set.contains(&m.moment_id),
                recommendation: None, // TODO: 实现推荐理由计算逻辑
            });
        }

        Ok(FeedTrendingData {
            trending_tags,
            hot_moments,
        })
    }

    async fn sign_moment_images(
        &self,
        images: &[String],
    ) -> Option<Vec<crate::dto::response::MomentImage>> {
        if images.is_empty() {
            return None;
        }
        let mut signed_images = Vec::with_capacity(images.len());
        for path in images {
            let url = match self.media_service.sign_url(path).await {
                Ok(u) => u,
                Err(e) => {
                    tracing::warn!("Failed to sign url for path {}: {}", path, e);
                    path.clone()
                }
            };
            signed_images.push(crate::dto::response::MomentImage {
                url,
                width: None,
                height: None,
                content_type: None,
            });
        }
        Some(signed_images)
    }
}
