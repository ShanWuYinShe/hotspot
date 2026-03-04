use tokio::try_join;

use crate::{
    dto::{
        request::{
            DismissFeedItemRequest, FeedGroupsRequest, FeedMomentsRequest, FeedTrendingRequest,
            FeedUsersRequest,
        },
        response::{
            FeedTrendingData, GroupInfo, GroupItem, GroupMyStateData, MomentImage, MomentItem,
            MyMomentState, PaginatedData, RecommendationContext, RelativeLocation, SocialContext,
            TrendingTagItem, UserInfo, UserItem,
        },
    },
    error::AppError,
    repositories::{
        dismiss_repo::FeedDismissRepository,
        group::member_repo::GroupMemberRepository,
        group_repo::GroupRepository,
        moment_repo::{FeedMomentRow, MomentRepository},
        tag_repo::TagRepository,
        user::follows_repo::UserFollowsRepository,
        user_repo::UserRepository,
    },
    types::enums::{
        DismissTargetType, FeedSource, RecommendationType, TagTrend, UserPresenceStatus,
    },
    utils::validation::Validator,
};
use uuid::Uuid;

#[derive(Clone)]
pub struct FeedService {
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    group_repo: GroupRepository,
    tag_repo: TagRepository,
    follows_repo: UserFollowsRepository,
    group_member_repo: GroupMemberRepository,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    dismiss_repo: FeedDismissRepository,
}

impl FeedService {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        group_repo: GroupRepository,
        tag_repo: TagRepository,
        follows_repo: UserFollowsRepository,
        group_member_repo: GroupMemberRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        dismiss_repo: FeedDismissRepository,
    ) -> Self {
        Self {
            moment_repo,
            user_repo,
            group_repo,
            tag_repo,
            follows_repo,
            group_member_repo,
            presence_service,
            media_service,
            dismiss_repo,
        }
    }

    /// 获取瞬间信息流（距离由 PostGIS 计算）
    pub async fn get_moments_feed(
        &self,
        user_id: i64,
        req: FeedMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>, AppError> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        let source = req.source;

        // 根据来源确定推荐理由
        let recommendation = match source {
            FeedSource::Following => Some(RecommendationContext {
                recommendation_type: RecommendationType::Following,
            }),
            FeedSource::Nearby => Some(RecommendationContext {
                recommendation_type: RecommendationType::Nearby,
            }),
            _ => None,
        };

        let (user_lat, user_lng) = if matches!(source, FeedSource::Following) {
            // Following 模式：优先使用 reference_point，否则从 presence 获取用户位置
            Validator::validate_coordinates(
                req.reference_point.latitude,
                req.reference_point.longitude,
            )?;
            (req.reference_point.latitude, req.reference_point.longitude)
        } else {
            Validator::validate_coordinates(
                req.reference_point.latitude,
                req.reference_point.longitude,
            )?;
            (req.reference_point.latitude, req.reference_point.longitude)
        };

        let (moments, next_cursor) = match source {
            FeedSource::Following => {
                self.moment_repo
                    .get_following_feed(user_id, user_lat, user_lng, limit, cursor)
                    .await?
            }
            FeedSource::Nearby => {
                let radius = Validator::validate_radius(
                    req.radius,
                    crate::constants::MAX_SEARCH_RADIUS,
                )?;
                self.moment_repo
                    .get_nearby_feed(user_lat, user_lng, radius, limit, cursor, Some(user_id))
                    .await?
            }
            _ => {
                self.moment_repo
                    .list_latest_feed(user_lat, user_lng, limit, cursor, Some(user_id))
                    .await?
            }
        };

        // 批量获取标签、点赞和收藏状态（并行）
        let moment_ids: Vec<_> = moments.iter().map(|m| m.moment_id).collect();
        let (tags_map, liked_set, favorited_set) = try_join!(
            self.tag_repo.get_moment_tags_batch(&moment_ids),
            self.moment_repo.batch_check_liked(&moment_ids, user_id),
            self.moment_repo.batch_check_favorited(&moment_ids, user_id)
        )?;

        let mut items = Vec::with_capacity(moments.len());
        for m in moments {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            let images = self.sign_moment_images(&m.images).await;
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;
            items.push(Self::build_moment_item(
                &m,
                images,
                avatar_url,
                tags,
                Some(MyMomentState {
                    is_liked: liked_set.contains(&m.moment_id),
                    is_favorited: favorited_set.contains(&m.moment_id),
                }),
                recommendation.clone(),
            ));
        }

        Ok(PaginatedData { items, next_cursor })
    }

    /// 获取群组推荐（距离由 PostGIS 计算）
    pub async fn get_groups_feed(
        &self,
        user_id: i64,
        req: FeedGroupsRequest,
    ) -> Result<PaginatedData<GroupItem>, AppError> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let (user_lat, user_lng) = (req.reference_point.latitude, req.reference_point.longitude);
        let radius =
            Validator::validate_radius(req.radius, crate::constants::MAX_SEARCH_RADIUS)?;

        let (groups, next_cursor) = self
            .group_repo
            .search_by_location(
                None,
                user_lng,
                user_lat,
                radius,
                limit,
                cursor,
                Some(user_id),
            )
            .await?;

        // 批量获取群组 ID
        let group_ids: Vec<i64> = groups.iter().map(|g| g.group_id).collect();

        // 批量查询用户在各群组中的成员状态、好友人数（并行）
        let (member_status_map, friends_in_group_map) = try_join!(
            self.group_member_repo
                .batch_find_member_status(user_id, &group_ids),
            self.follows_repo
                .count_friends_in_groups_batch(user_id, &group_ids)
        )?;

        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(g.avatar_url.as_deref())
                .await;

            // 构建 my_state（仅当用户是群成员时才有值）
            let my_state = member_status_map
                .get(&g.group_id)
                .map(|m| GroupMyStateData {
                    role: m.role,
                    is_joined: true,
                    unread_count: 0, // Feed 场景不加载详细未读数
                    is_muted: false, // Feed 场景使用默认值，详细设置见群组详情
                    is_pinned: false,
                    joined_at: Some(m.joined_at.to_rfc3339()),
                });

            // 好友在群组中的人数
            let friends_in_group = friends_in_group_map.get(&g.group_id).copied();

            items.push(GroupItem {
                group: GroupInfo {
                    group_id: g.group_id.to_string(),
                    name: g.name,
                    avatar_url,
                    members_count: g.member_count,
                },
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
                location: RelativeLocation::with_distance(
                    g.latitude,
                    g.longitude,
                    g.location_name,
                    g.distance,
                    g.bearing_degrees,
                ),
                last_activity_at: None,
                created_at: None,
                owner_id: None,
                my_state,
                friends_in_group,
                join_method: None,
                is_searchable: None,
                max_members: None,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    /// 获取用户推荐（距离由 PostGIS 计算）
    pub async fn get_users_feed(
        &self,
        user_id: i64,
        req: FeedUsersRequest,
    ) -> Result<PaginatedData<UserItem>, AppError> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i32;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let (user_lat, user_lng) = (req.reference_point.latitude, req.reference_point.longitude);
        let radius =
            Validator::validate_radius(req.radius, crate::constants::MAX_SEARCH_RADIUS)?;

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

        // 批量计算共同好友、在线状态、位置可见性（并行）
        let (mutual_friends_map, online_status_map, location_visible_map) = tokio::join!(
            self.follows_repo
                .count_mutual_friends_batch(user_id, &user_ids),
            self.presence_service
                .batch_get_online_status_with_privacy(&user_ids),
            self.presence_service.batch_get_location_visible(&user_ids)
        );
        let mutual_friends_map = mutual_friends_map?;

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
                Some(RelativeLocation::with_distance(
                    u.latitude,
                    u.longitude,
                    u.location_name,
                    u.distance,
                    u.bearing_degrees,
                ))
            } else {
                None
            };

            // 签名 avatar_url
            let avatar_url = self
                .media_service
                .sign_url_if_path(u.avatar_url.as_deref())
                .await;

            items.push(UserItem {
                user: UserInfo::new(u.user_id, u.nickname, avatar_url),
                bio: if u.bio.is_empty() { None } else { Some(u.bio) },
                tags: Some(u.tags),
                presence_status: is_online.map(|online| {
                    if online {
                        UserPresenceStatus::Online
                    } else {
                        UserPresenceStatus::Offline
                    }
                }),
                last_online,
                location,
                stats: None,
                relation: None,
                social_context: Some(SocialContext {
                    mutual_friends_count: mutual_friends_map.get(&u.user_id).copied().unwrap_or(0),
                    mutual_friends_preview: None,
                    common_groups_count: None,
                    common_tags: None,
                }),
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    /// 获取热门内容
    /// user_id: 登录态时传入 Some，未登录时传入 None；未登录时不返回 my_state 等个性化字段
    pub async fn get_trending(
        &self,
        user_id: Option<i64>,
        _req: FeedTrendingRequest,
    ) -> Result<FeedTrendingData, AppError> {
        let limit = crate::constants::DEFAULT_TRENDING_TAGS_LIMIT;
        let since =
            chrono::Utc::now() - chrono::Duration::days(crate::constants::TRENDING_WEEKLY_DAYS);

        // 获取热门标签、热门瞬间（并行）
        let (tags, (hot_moments_raw, _)) = try_join!(
            self.tag_repo.get_trending(limit),
            self.moment_repo.get_hot_moments(5, since, None)
        )?;

        let trending_tags: Vec<TrendingTagItem> = tags
            .into_iter()
            .map(|t| TrendingTagItem {
                name: t.name,
                usage_count: t.usage_count.unwrap_or(0),
                trend: TagTrend::Stable,
                hot_moment_preview: None,
            })
            .collect();

        // 批量获取标签、点赞和收藏状态（并行）
        let hot_moment_ids: Vec<_> = hot_moments_raw.iter().map(|m| m.moment_id).collect();
        let (hot_tags_map, liked_set, favorited_set) = match user_id {
            Some(uid) => {
                let (tags, liked, favorited) = try_join!(
                    self.tag_repo.get_moment_tags_batch(&hot_moment_ids),
                    self.moment_repo.batch_check_liked(&hot_moment_ids, uid),
                    self.moment_repo.batch_check_favorited(&hot_moment_ids, uid)
                )?;
                (tags, liked, favorited)
            }
            None => {
                let hot_tags_map = self.tag_repo.get_moment_tags_batch(&hot_moment_ids).await?;
                (
                    hot_tags_map,
                    std::collections::HashSet::new(),
                    std::collections::HashSet::new(),
                )
            }
        };

        let mut hot_moments = Vec::with_capacity(hot_moments_raw.len());
        for m in hot_moments_raw {
            let tags = hot_tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            let images = self.sign_moment_images(&m.images).await;
            let avatar_url = self
                .media_service
                .sign_url_if_path(m.avatar_url.as_deref())
                .await;
            hot_moments.push(Self::build_moment_item(
                &m,
                images,
                avatar_url,
                tags,
                user_id.map(|_| MyMomentState {
                    is_liked: liked_set.contains(&m.moment_id),
                    is_favorited: favorited_set.contains(&m.moment_id),
                }),
                Some(RecommendationContext {
                    recommendation_type: RecommendationType::Trending,
                }),
            ));
        }

        Ok(FeedTrendingData {
            trending_tags,
            hot_moments,
        })
    }

    fn build_moment_item(
        m: &FeedMomentRow,
        images: Option<Vec<MomentImage>>,
        avatar_url: Option<String>,
        tags: Vec<String>,
        my_state: Option<MyMomentState>,
        recommendation: Option<RecommendationContext>,
    ) -> MomentItem {
        MomentItem {
            moment_id: m.moment_id.to_string(),
            user: UserInfo::new(m.user_id, m.nickname.clone(), avatar_url),
            description: m.description.clone(),
            images,
            tags,
            mentions: m.mentions.clone(),
            created_at: m.created_at.to_rfc3339(),
            likes_count: m.likes_count as i32,
            comments_count: m.comments_count as i32,
            shares_count: 0,
            location: match (m.distance, m.bearing_degrees) {
                (Some(d), Some(b)) => RelativeLocation::with_distance(
                    m.latitude.unwrap_or(0.0),
                    m.longitude.unwrap_or(0.0),
                    m.location_name.clone().unwrap_or_default(),
                    d,
                    b,
                ),
                _ => RelativeLocation::without_distance(
                    m.latitude.unwrap_or(0.0),
                    m.longitude.unwrap_or(0.0),
                    m.location_name.clone().unwrap_or_default(),
                ),
            },
            privacy_level: None,
            my_state,
            recommendation,
        }
    }

    async fn sign_moment_images(&self, images: &[String]) -> Option<Vec<MomentImage>> {
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
            signed_images.push(MomentImage {
                url,
                thumbnail_url: None,
                width: None,
                height: None,
                mime_type: None,
            });
        }
        Some(signed_images)
    }

    /// 标记 Feed 项为“不感兴趣”
    pub async fn dismiss_item(
        &self,
        user_id: i64,
        req: &DismissFeedItemRequest,
    ) -> Result<(), AppError> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            user_id,
            target_type = ?req.target_type,
            target_id = %req.target_id,
            "feed.dismiss_item started"
        );
        if req.target_id.trim().is_empty() {
            tracing::warn!(user_id, target_type = ?req.target_type, "feed.dismiss_item rejected: empty target_id");
            return Err(AppError::BadRequest("target_id 不能为空".to_string()));
        }

        match req.target_type {
            DismissTargetType::Moment => {
                let moment_id = Uuid::parse_str(&req.target_id)
                    .map_err(|_| AppError::InvalidFormat("moment_id 格式错误".to_string()))?;
                let exists = self.moment_repo.find_by_id(moment_id).await?;
                if exists.is_none() {
                    tracing::warn!(user_id, moment_id = %moment_id, "feed.dismiss_item rejected: moment not found");
                    return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
                }
            }
            DismissTargetType::User => {
                let target_user_id: i64 = req
                    .target_id
                    .parse()
                    .map_err(|_| AppError::InvalidFormat("user_id 格式错误".to_string()))?;
                if self.user_repo.find_by_id(target_user_id).await?.is_none() {
                    tracing::warn!(
                        user_id,
                        target_user_id,
                        "feed.dismiss_item rejected: user not found"
                    );
                    return Err(AppError::UserNotFound("用户不存在".to_string()));
                }
            }
            DismissTargetType::Group => {
                let target_group_id: i64 = req
                    .target_id
                    .parse()
                    .map_err(|_| AppError::InvalidFormat("group_id 格式错误".to_string()))?;
                if self.group_repo.find_by_id(target_group_id).await?.is_none() {
                    tracing::warn!(
                        user_id,
                        target_group_id,
                        "feed.dismiss_item rejected: group not found"
                    );
                    return Err(AppError::GroupNotFound("群组不存在".to_string()));
                }
            }
        }

        self.dismiss_repo
            .upsert(user_id, req.target_type, req.target_id.as_str())
            .await?;
        tracing::info!(
            user_id,
            target_type = ?req.target_type,
            target_id = %req.target_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "feed.dismiss_item completed"
        );

        Ok(())
    }
}
