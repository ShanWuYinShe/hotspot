//! 用户服务层实现
//!
//! 核心业务逻辑：资料、关系链、搜索、设置
//!
//! ## 有意延期的端点
//!
//! 以下 4 个端点在 spec 中已定义（幂等 MUST 列表），但实现有意延期：
//! - `users/me/email/request_change` — 请求邮箱变更验证码
//! - `users/me/email/confirm_change` — 确认邮箱变更
//! - `users/me/phone/request_change` — 请求手机号变更验证码
//! - `users/me/phone/confirm_change` — 确认手机号变更

use std::collections::HashMap;
use std::str::FromStr;
use std::sync::Arc;
use std::time::Duration;

use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AdminUserSearchRequest, BlockRequest, DeleteAccountRequest, FollowRequest,
            GetRelationshipRequest, ListFollowersRequest, ListFollowingRequest, ListFriendsRequest,
            MyBlockedRequest, SearchUsersRequest, SetRemarkRequest, UnblockRequest,
            UnfollowRequest, UpdateLocationRequest, UpdateProfileRequest, UpdateSettingsRequest,
            UpgradeAccountRequest, UserGroupsRequest, UserLikedMomentsRequest,
            UserOpenTheDoorRequest,
        },
        response::{
            AdminUserDetail, AdminUserSearchItem, BlockedUserSearchItem, FollowerItem,
            FollowingItem, FriendItem, GroupInfo, GroupItem, Location, MomentItem, MyProfile,
            PaginatedData, Relationship, RelativeLocation, SocialContext, UserInfo, UserItem,
            UserSettingsData, UserStats,
        },
    },
    error::{ApiError, Result},
    events::event_types::AppEvent,
    repositories::{
        group::member_repo::GroupMemberRepository,
        moment_repo::{MomentDetailRow, MomentImageRow, MomentRepository},
        session_repo::SessionRepository,
        tag_repo::TagRepository,
        user::blocks_repo::UserBlocksRepository,
        user::follows_repo::UserFollowsRepository,
        user::friend_remarks_repo::UserFriendRemarksRepository,
        user::location_history_repo::UserLocationHistoryRepository,
        user::settings_repo::{UpdateSettingsParams, UserSettingsRepository, UserSettingsRow},
        user_repo::{UserRepository, UserRow},
    },
    types::enums::{PrivacyLevel, PrivateChatPermission, UserPresenceStatus, UserStatus, UserType},
    utils::{
        bloom_filter::ResourceExistenceChecker, cache::CacheManager, password::PasswordUtil,
        validation::Validator,
    },
    EventBus,
};

#[derive(Clone)]
pub struct UserService {
		event_bus: Arc<EventBus>,
		user_repo: UserRepository,
		user_settings_repo: UserSettingsRepository,
		user_location_history_repo: UserLocationHistoryRepository,
    follows_repo: UserFollowsRepository,
    blocks_repo: UserBlocksRepository,
    user_friend_remarks_repo: UserFriendRemarksRepository,
    session_repo: SessionRepository,
    tag_repo: TagRepository,
    group_member_repo: GroupMemberRepository,
		moment_repo: MomentRepository,
		presence_service: crate::services::presence_service::PresenceService,
		media_service: crate::services::media_service::MediaService,
		cache_manager: CacheManager,
		_resource_checker: ResourceExistenceChecker,
		region_boundary_service: crate::services::region_boundary_service::RegionBoundaryService,
}

const SEARCH_QUERY_TIMEOUT: Duration = Duration::from_secs(8);

impl UserService {
    #[allow(clippy::too_many_arguments)]
	pub fn new(
			event_bus: Arc<EventBus>,
			user_repo: UserRepository,
			user_settings_repo: UserSettingsRepository,
			user_location_history_repo: UserLocationHistoryRepository,
        follows_repo: UserFollowsRepository,
        blocks_repo: UserBlocksRepository,
        user_friend_remarks_repo: UserFriendRemarksRepository,
        session_repo: SessionRepository,
        tag_repo: TagRepository,
        group_member_repo: GroupMemberRepository,
			moment_repo: MomentRepository,
			presence_service: crate::services::presence_service::PresenceService,
			media_service: crate::services::media_service::MediaService,
			cache_manager: CacheManager,
			resource_checker: ResourceExistenceChecker,
			region_boundary_service: crate::services::region_boundary_service::RegionBoundaryService,
		) -> Self {
			Self {
					event_bus,
					user_repo,
					user_settings_repo,
            user_location_history_repo,
            follows_repo,
            blocks_repo,
            user_friend_remarks_repo,
            session_repo,
            tag_repo,
            group_member_repo,
					moment_repo,
					presence_service,
				media_service,
				cache_manager,
				_resource_checker: resource_checker,
				region_boundary_service,
			}
		}

    pub async fn search(
        &self,
        viewer_id: Option<i64>,
        req: &SearchUsersRequest,
    ) -> Result<PaginatedData<UserItem>> {
        let query_started_at = std::time::Instant::now();
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let cursor = pagination.and_then(|p| p.cursor.clone());
        let radius = Validator::validate_radius_required(
            req.radius.unwrap_or(5000.0),
            crate::constants::MAX_SEARCH_RADIUS,
        )?;
        let has_cursor = cursor.is_some();
        let query_mode = if req.radius.is_some() {
            "nearby"
        } else if req.tag.is_some() {
            "tag"
        } else if req.keyword.is_some() {
            "keyword"
        } else {
            "invalid"
        };
        tracing::info!(
            viewer_id,
            query_mode,
            has_keyword = req.keyword.is_some(),
            has_tag = req.tag.is_some(),
            radius,
            limit,
            has_cursor,
            "users.search query started"
        );

        let query_future = async {
            if req.radius.is_some() {
                self.user_repo
                    .search_nearby(
                        req.reference_point.longitude,
                        req.reference_point.latitude,
                        radius,
                        limit,
                        cursor.clone(),
                        req.keyword.as_deref(),
                    )
                    .await
                    .map_err(|err| {
                        tracing::error!(
                            viewer_id,
                            query_mode = "nearby",
                            radius,
                            limit,
                            has_cursor,
                            error = ?err,
                            "users.search query failed"
                        );
                        err
                    })
            } else if let Some(ref tag) = req.tag {
                Validator::validate_not_empty(tag, "tag")?;
                self.user_repo
                    .search_by_tag(
                        tag,
                        req.reference_point.longitude,
                        req.reference_point.latitude,
                        radius,
                        limit,
                        cursor.clone(),
                    )
                    .await
                    .map_err(|err| {
                        tracing::error!(
                            viewer_id,
                            query_mode = "tag",
                            tag = %tag,
                            radius,
                            limit,
                            has_cursor,
                            error = ?err,
                            "users.search query failed"
                        );
                        err
                    })
            } else if let Some(ref keyword) = req.keyword {
                Validator::validate_not_empty(keyword, "keyword")?;
                self.user_repo
                    .search_by_keyword(
                        keyword,
                        req.reference_point.longitude,
                        req.reference_point.latitude,
                        radius,
                        limit,
                        cursor.clone(),
                    )
                    .await
                    .map_err(|err| {
                        tracing::error!(
                            viewer_id,
                            query_mode = "keyword",
                            keyword = %keyword,
                            radius,
                            limit,
                            has_cursor,
                            error = ?err,
                            "users.search query failed"
                        );
                        err
                    })
            } else {
                Err(ApiError::BadRequest(
                    "至少需要提供 keyword、tag、radius 中的一个搜索条件".to_string(),
                ))
            }
        };
        let (rows, next_cursor) = tokio::time::timeout(SEARCH_QUERY_TIMEOUT, query_future)
            .await
            .map_err(|_| {
                tracing::error!(
                    viewer_id,
                    query_mode,
                    radius,
                    limit,
                    has_cursor,
                    timeout_ms = SEARCH_QUERY_TIMEOUT.as_millis() as u64,
                    "users.search query timeout"
                );
                ApiError::ServiceUnavailable
            })??;
        let rows_count = rows.len();
        let has_next_cursor = next_cursor.is_some();
        tracing::info!(
            viewer_id,
            rows = rows_count,
            has_next_cursor,
            elapsed_ms = query_started_at.elapsed().as_millis(),
            "users.search query completed"
        );

        let build_started_at = std::time::Instant::now();
        let data = self
            .build_user_search_list(rows, next_cursor, viewer_id)
            .await
            .map_err(|err| {
                tracing::error!(
                    viewer_id,
                    query_mode,
                    error = ?err,
                    "users.search response assemble failed"
                );
                err
            })?;
        tracing::info!(
            viewer_id,
            items = data.items.len(),
            has_next_cursor = data.next_cursor.is_some(),
            elapsed_ms = build_started_at.elapsed().as_millis(),
            "users.search response assembled"
        );
        Ok(data)
    }

    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &UserOpenTheDoorRequest,
    ) -> Result<UserItem> {
        let user_id = Validator::parse_user_id(&req.user_id)?;
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| ApiError::UserNotFound("用户不存在".to_string()))?;

        if let Some(viewer_id) = viewer_id {
            if self.blocks_repo.is_blocked(user_id, viewer_id).await? {
                return Err(ApiError::BlockedByUser("privacy_restricted".to_string()));
            }
        }

        let tags = self
            .tag_repo
            .get_user_tags(user.user_id)
            .await
            .unwrap_or_default();
        let (is_online, last_online) = self
            .presence_service
            .get_online_status_with_privacy(user.user_id)
            .await;

        let location_visible = self.location_visible_for_user(user.user_id).await;
        let location = if location_visible {
            self.build_location_for_user(user.user_id).await
        } else {
            None
        };

        let stats = if viewer_id.is_some() {
            Some(self.build_user_card_stats(user.user_id).await?)
        } else {
            None
        };

        let relation = if let Some(viewer_id) = viewer_id {
            Some(
                self.get_relationship(
                    viewer_id,
                    &GetRelationshipRequest {
                        user_id: user.user_id.to_string(),
                    },
                )
                .await?,
            )
        } else {
            None
        };

        Ok(UserItem {
            user: UserInfo::new(
                user.user_id,
                user.nickname.unwrap_or_default(),
                self.media_service
                    .sign_url_if_path(user.avatar_url.as_deref())
                    .await,
            ),
            bio: user.bio,
            tags: Some(tags),
            presence_status: is_online.map(|online| {
                if online {
                    UserPresenceStatus::Online
                } else {
                    UserPresenceStatus::Offline
                }
            }),
            last_online,
            location,
            stats,
            relation,
            social_context: None,
        })
    }

    pub async fn get_me(&self, user_id: i64) -> Result<MyProfile> {
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| ApiError::UserNotFound("用户不存在".to_string()))?;

        let tags = self
            .tag_repo
            .get_user_tags(user_id)
            .await
            .unwrap_or_default();
        let stats = self.get_stats(user_id).await?;

        let (is_online, last_online) = self.presence_service.get_online_status(user_id).await;
        let last_online = last_online
            .or_else(|| user.last_login_at.map(|t| t.to_rfc3339()))
            .unwrap_or_else(|| user.created_at.to_rfc3339());

        let relative_location = self
            .build_location_for_user(user_id)
            .await
            .unwrap_or_else(|| RelativeLocation::without_distance(0.0, 0.0, ""));

        let location = Location {
            point: relative_location.point,
            location_name: relative_location.location_name.clone(),
        };

        let presence_status = if is_online {
            UserPresenceStatus::Online
        } else {
            UserPresenceStatus::Offline
        };

        Ok(MyProfile {
            user_id: user.user_id,
            nickname: user.nickname.unwrap_or_default(),
            avatar_url: self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
            user_type: user.user_type,
            bio: user.bio,
            tags: Some(tags),
            gender: Some(user.gender),
            email: user.email,
            phone: user.phone,
            location,
            presence_status,
            last_online,
            created_at: user.created_at.to_rfc3339(),
            stats,
        })
    }

    pub async fn update_profile(&self, user_id: i64, req: &UpdateProfileRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            user_id,
            has_nickname = req.nickname.is_some(),
            has_bio = req.bio.is_some(),
            has_avatar_file = req.avatar_file_id.is_some(),
            has_tags = req.tags.is_some(),
            has_gender = req.gender.is_some(),
            "user.update_profile started"
        );
        if let Some(nickname) = &req.nickname {
            Validator::validate_nickname(nickname)?;
        }
        if let Some(bio) = &req.bio {
            Validator::validate_length(bio, "bio", 0, 500)?;
        }

        let avatar_url = if let Some(fid) = req.avatar_file_id.as_deref() {
            let fid_uuid = Validator::parse_uuid(fid, "文件ID")?;
            let file = self
                .media_service
                .validate_file(fid_uuid, user_id, "avatar")
                .await?;
            Some(file)
        } else {
            None
        };

        self.user_repo
            .update_profile(
                user_id,
                req.nickname.as_deref(),
                avatar_url.as_deref(),
                req.bio.as_deref(),
                req.gender,
                None,
            )
            .await?;

        if let Some(tags) = &req.tags {
            if tags.len() > 10 {
                tracing::warn!(
                    user_id,
                    tags_count = tags.len(),
                    "user.update_profile rejected: too many tags"
                );
                return Err(ApiError::BadRequest("tags 最多 10 个".to_string()));
            }
            for tag in tags {
                Validator::validate_length(tag, "tag", 1, 30)?;
            }
            self.tag_repo.set_user_tags(user_id, tags).await?;
        }

        let _ = self
            .cache_manager
            .delete(
                crate::utils::cache::cache_keys::USER_INFO,
                &user_id.to_string(),
            )
            .await;

        self.event_bus
            .publish(crate::events::event_types::AppEvent::UserProfileUpdated {
                user_id,
                nickname: req.nickname.clone(),
                avatar_url: avatar_url.map(|s| s.to_string()),
                bio: req.bio.clone(),
            });
        tracing::info!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.update_profile completed"
        );

        Ok(())
    }

    pub async fn get_relationship(
        &self,
        user_id: i64,
        req: &GetRelationshipRequest,
    ) -> Result<Relationship> {
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let is_following = self
            .follows_repo
            .is_following(user_id, target_user_id)
            .await?;
        let is_followed_by = self
            .follows_repo
            .is_following(target_user_id, user_id)
            .await?;
        let is_blocked = self.blocks_repo.is_blocked(user_id, target_user_id).await?;

        Ok(Relationship {
            is_following,
            is_followed_by,
            is_blocked,
        })
    }

    pub async fn get_stats(&self, user_id: i64) -> Result<UserStats> {
        let stats = self.user_repo.get_stats(user_id).await?;
        Ok(UserStats {
            moments_count: stats.moments_count,
            friends_count: stats.following_count,
            followers_count: stats.followers_count,
            following_count: stats.following_count,
            groups_count: stats.groups_count,
        })
    }

    pub async fn get_settings(&self, user_id: i64) -> Result<UserSettingsData> {
        let settings = self.user_settings_repo.find_by_user_id(user_id).await?;
        Ok(self.map_settings(settings.as_ref()))
    }

    pub async fn update_settings(&self, user_id: i64, req: &UpdateSettingsRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let s = &req.settings;
        let changed_fields = [
            s.privacy_level.is_some(),
            s.location_visible.is_some(),
            s.presence_visible.is_some(),
            s.social_lists_visible.is_some(),
            s.groups_visible.is_some(),
            s.liked_moments_visible.is_some(),
            s.stats_visible.is_some(),
            s.allow_follow.is_some(),
            s.allow_private_chat.is_some(),
            s.searchable.is_some(),
            s.phone_searchable.is_some(),
            s.nickname_searchable.is_some(),
            s.location_searchable.is_some(),
        ]
        .into_iter()
        .filter(|changed| *changed)
        .count();
        tracing::info!(user_id, changed_fields, "user.update_settings started");
        let params = UpdateSettingsParams {
            privacy_level: s.privacy_level,
            location_visible: s.location_visible,
            presence_visible: s.presence_visible,
            social_lists_visible: s.social_lists_visible,
            groups_visible: s.groups_visible,
            liked_moments_visible: s.liked_moments_visible,
            stats_visible: s.stats_visible,
            allow_follow: s.allow_follow,
            allow_private_chat: s.allow_private_chat,
            searchable: s.searchable,
            phone_searchable: s.phone_searchable,
            nickname_searchable: s.nickname_searchable,
            location_searchable: s.location_searchable,
        };

        self.user_settings_repo.update(user_id, &params).await?;
        tracing::info!(
            user_id,
            changed_fields,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.update_settings completed"
        );
        Ok(())
    }

    pub async fn get_friends_list(
        &self,
        target_user_id: i64,
        viewer_id: Option<i64>,
        req: &ListFriendsRequest,
    ) -> Result<PaginatedData<FriendItem>> {
        if viewer_id != Some(target_user_id) {
            let social_visible = self.social_lists_visible(target_user_id).await;
            if !social_visible {
                return Err(ApiError::Forbidden("对方的好友列表不可见".to_string()));
            }
        }
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_friends(
                target_user_id,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let friend_ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&friend_ids).await?;
        let profile_map: HashMap<i64, UserRow> =
            profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let is_own_list = viewer_id == Some(target_user_id);
        let remark_map = if is_own_list {
            self.user_friend_remarks_repo
                .get_batch(target_user_id, &friend_ids)
                .await
                .unwrap_or_default()
        } else {
            HashMap::new()
        };
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&friend_ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let (is_online, last_online) = online_map
                    .get(&row.user_id)
                    .cloned()
                    .unwrap_or((None, None));
                let presence_status = is_online.map(|online| {
                    if online {
                        UserPresenceStatus::Online
                    } else {
                        UserPresenceStatus::Offline
                    }
                });
                items.push(FriendItem {
                    user: self.build_user_info(profile).await,
                    presence_status,
                    last_online,
                    remark: if is_own_list {
                        remark_map.get(&row.user_id).cloned()
                    } else {
                        None
                    },
                    followed_at: row.followed_at.to_rfc3339(),
                    followed_back_at: row.followed_back_at.to_rfc3339(),
                });
            }
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_following_list(
        &self,
        target_user_id: i64,
        viewer_id: Option<i64>,
        req: &ListFollowingRequest,
    ) -> Result<PaginatedData<FollowingItem>> {
        if viewer_id != Some(target_user_id) {
            let social_visible = self.social_lists_visible(target_user_id).await;
            if !social_visible {
                return Err(ApiError::Forbidden("对方的关注列表不可见".to_string()));
            }
        }
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_following(
                target_user_id,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> =
            profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let mutual_ids = self
            .follows_repo
            .batch_check_followed_by(target_user_id, &ids)
            .await
            .unwrap_or_default();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let is_mutual = mutual_ids.contains(&row.user_id);
                let (is_online, last_online) = online_map
                    .get(&row.user_id)
                    .cloned()
                    .unwrap_or((None, None));
                let presence_status = is_online.map(|online| {
                    if online {
                        UserPresenceStatus::Online
                    } else {
                        UserPresenceStatus::Offline
                    }
                });
                items.push(FollowingItem {
                    user: self.build_user_info(profile).await,
                    followed_at: row.created_at.to_rfc3339(),
                    is_mutual,
                    presence_status,
                    last_online,
                });
            }
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_followers_list(
        &self,
        target_user_id: i64,
        viewer_id: Option<i64>,
        req: &ListFollowersRequest,
    ) -> Result<PaginatedData<FollowerItem>> {
        if viewer_id != Some(target_user_id) {
            let social_visible = self.social_lists_visible(target_user_id).await;
            if !social_visible {
                return Err(ApiError::Forbidden("对方的粉丝列表不可见".to_string()));
            }
        }
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_followers(
                target_user_id,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> =
            profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let is_following_back = self
                    .follows_repo
                    .is_following(target_user_id, row.user_id)
                    .await
                    .unwrap_or(false);
                let (is_online, last_online) = online_map
                    .get(&row.user_id)
                    .cloned()
                    .unwrap_or((None, None));
                let presence_status = is_online.map(|online| {
                    if online {
                        UserPresenceStatus::Online
                    } else {
                        UserPresenceStatus::Offline
                    }
                });
                items.push(FollowerItem {
                    user: self.build_user_info(profile).await,
                    followed_at: row.created_at.to_rfc3339(),
                    is_following_back,
                    presence_status,
                    last_online,
                });
            }
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_user_groups_list(
        &self,
        req: &UserGroupsRequest,
        viewer_id: Option<i64>,
    ) -> Result<PaginatedData<GroupItem>> {
        let user_id = Validator::parse_user_id(&req.user_id)?;
        // 隐私检查：查看其他用户群组列表时，受 groups_visible 设置控制
        if viewer_id != Some(user_id) {
            let settings = self
                .user_settings_repo
                .find_by_user_id(user_id)
                .await
                .ok()
                .flatten();
            let visible = self.privacy_bool(settings.as_ref(), "groups_visible", true);
            if !visible {
                return Err(ApiError::Forbidden("对方的群组列表不可见".to_string()));
            }
        }
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .group_member_repo
            .list_user_groups(
                user_id,
                limit as usize,
                pagination.and_then(|p| p.cursor.as_deref()),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            items.push(GroupItem {
                group: GroupInfo {
                    group_id: row.group_id.to_string(),
                    name: row.name.clone(),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(row.avatar_url.as_deref())
                        .await,
                    members_count: row.members_count,
                },
                description: row.description,
                tags: None,
                has_password: row.has_password,
                location: RelativeLocation::without_distance(0.0, 0.0, ""),
                last_activity_at: None,
                created_at: None,
                owner_id: None,
                my_state: None,
                friends_in_group: None,
                join_method: None,
                is_searchable: None,
                max_members: None,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_user_liked_moments_list(
        &self,
        req: &UserLikedMomentsRequest,
        viewer_id: Option<i64>,
    ) -> Result<PaginatedData<MomentItem>> {
        let user_id = Validator::parse_user_id(&req.user_id)?;
        // 隐私检查：查看其他用户点赞列表时，受 liked_moments_visible 设置控制
        if viewer_id != Some(user_id) {
            let settings = self
                .user_settings_repo
                .find_by_user_id(user_id)
                .await
                .ok()
                .flatten();
            let visible = self.privacy_bool(settings.as_ref(), "liked_moments_visible", true);
            if !visible {
                return Err(ApiError::Forbidden("对方的点赞列表不可见".to_string()));
            }
        }
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_user_liked_moments(user_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        self.build_user_moment_items(rows, next_cursor).await
    }

	pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
		let started_at = std::time::Instant::now();
		tracing::info!(
			user_id,
			latitude = req.location.point.latitude,
			longitude = req.location.point.longitude,
			location_name_len = req.location.location_name.chars().count(),
			"user.update_location started"
		);
		let lat = req.location.point.latitude;
		let lon = req.location.point.longitude;
		Validator::validate_coordinates(lat, lon)?;
		let region_id = self
			.region_boundary_service
			.assign_region_for_point(lat, lon)
			.await;
		self
			.user_repo
			.update_location_with_region(user_id, lon, lat, region_id)
			.await?;
		let _ = self
			.user_location_history_repo
			.record(user_id, lon, lat)
			.await;
		let _ = self
			.presence_service
			.update_location_cache(user_id, lat, lon)
			.await;
        tracing::info!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.update_location completed"
        );
        Ok(())
    }

    pub async fn upgrade_account(&self, user_id: i64, req: &UpgradeAccountRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            user_id,
            login_id_len = req.login_id.chars().count(),
            "user.upgrade_account started"
        );
        Validator::validate_login_id(&req.login_id)?;
        Validator::validate_password(&req.password, "password")?;

        if let Some(existing) = self.user_repo.find_by_identifier(&req.login_id).await? {
            if existing.user_id != user_id {
                tracing::warn!(
                    user_id,
                    existing_user_id = existing.user_id,
                    "user.upgrade_account rejected: login_id already occupied"
                );
                return Err(ApiError::AlreadyExists("该用户已存在".to_string()));
            }
        }

        let password_hash = PasswordUtil::hash_password(&req.password)?;
        self.user_repo
            .update_password(user_id, &password_hash)
            .await?;
        self.user_repo
            .update_login_id(user_id, &req.login_id)
            .await?;
        self.user_repo
            .update_user_type(user_id, UserType::Normal)
            .await?;
        tracing::info!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.upgrade_account completed"
        );
        Ok(())
    }

    pub async fn delete_account(&self, user_id: i64, req: &DeleteAccountRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(user_id, "user.delete_account started");
        if req.confirmation != "DELETE_MY_ACCOUNT" {
            tracing::warn!(
                user_id,
                confirmation = %req.confirmation,
                "user.delete_account rejected: invalid confirmation token"
            );
            return Err(ApiError::BadRequest(
                "confirmation 必须为 DELETE_MY_ACCOUNT".to_string(),
            ));
        }
        self.user_repo.mark_deleted(user_id).await?;
        let _ = self.session_repo.revoke_all_by_user(user_id).await;
        tracing::info!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.delete_account completed"
        );
        Ok(())
    }

    pub async fn follow_user(&self, user_id: i64, req: &FollowRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        tracing::info!(
            user_id,
            target_user_id,
            has_source = req.source.is_some(),
            has_source_moment = req.source_moment_id.is_some(),
            "user.follow started"
        );
        if user_id == target_user_id {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.follow rejected: cannot follow self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }

        if !self.user_repo.exists(target_user_id).await? {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.follow rejected: target user not found"
            );
            return Err(ApiError::UserNotFound("用户不存在".to_string()));
        }

        if self.blocks_repo.is_blocked(target_user_id, user_id).await? {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.follow rejected: blocked by target user"
            );
            return Err(ApiError::BlockedByUser("privacy_restricted".to_string()));
        }

        let allow_follow = self
            .bool_from_settings(target_user_id, "allow_follow", true)
            .await;
        if !allow_follow {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.follow rejected: target disallows follow"
            );
            return Err(ApiError::FollowNotAllowed("对方不允许被关注".to_string()));
        }

        if let Some(ref source) = req.source {
            tracing::debug!(
                follow_source = ?source,
                source_moment_id = ?req.source_moment_id,
                "Follow source recorded"
            );
        }

        self.follows_repo.follow(user_id, target_user_id).await?;
        let is_mutual = self
            .follows_repo
            .is_following(target_user_id, user_id)
            .await?;

        self.event_bus.publish(AppEvent::NewFollower {
            follower_id: user_id,
            user_id: target_user_id,
            is_mutual,
            source: req.source,
            source_moment_id: req.source_moment_id.clone(),
        });
        tracing::info!(
            user_id,
            target_user_id,
            is_mutual,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.follow completed"
        );

        Ok(())
    }

    pub async fn unfollow_user(&self, user_id: i64, req: &UnfollowRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        tracing::info!(user_id, target_user_id, "user.unfollow started");
        if user_id == target_user_id {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.unfollow rejected: cannot unfollow self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }

        self.follows_repo.unfollow(user_id, target_user_id).await?;
        tracing::info!(
            user_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.unfollow completed"
        );
        Ok(())
    }

    pub async fn block_user(&self, user_id: i64, req: &BlockRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        tracing::info!(
            user_id,
            target_user_id,
            has_reason = req.reason.is_some(),
            "user.block started"
        );
        if user_id == target_user_id {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.block rejected: cannot block self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }
        if !self.user_repo.exists(target_user_id).await? {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.block rejected: target user not found"
            );
            return Err(ApiError::UserNotFound("用户不存在".to_string()));
        }
        if let Some(ref reason) = req.reason {
            tracing::debug!(block_reason = %reason, "Block reason recorded");
        }
        self.blocks_repo.block(user_id, target_user_id).await?;
        tracing::info!(
            user_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.block completed"
        );
        Ok(())
    }

    pub async fn unblock_user(&self, user_id: i64, req: &UnblockRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        tracing::info!(user_id, target_user_id, "user.unblock started");
        if user_id == target_user_id {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.unblock rejected: cannot unblock self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }
        self.blocks_repo.unblock(user_id, target_user_id).await?;
        tracing::info!(
            user_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.unblock completed"
        );
        Ok(())
    }

    pub async fn get_blocked_users_list(
        &self,
        user_id: i64,
        req: &MyBlockedRequest,
    ) -> Result<PaginatedData<BlockedUserSearchItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .blocks_repo
            .list_blocked(user_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> =
            profiles.into_iter().map(|p| (p.user_id, p)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                items.push(BlockedUserSearchItem {
                    user: self.build_user_info(profile).await,
                    blocked_at: row.created_at.to_rfc3339(),
                    reason: None,
                });
            }
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn set_friend_remark(&self, user_id: i64, req: &SetRemarkRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id = Validator::parse_user_id(&req.user_id)?;
        let trimmed = req.remark.trim();
        Validator::validate_remark(trimmed)?;
        tracing::info!(
            user_id,
            target_user_id,
            remark_len = trimmed.chars().count(),
            "user.set_friend_remark started"
        );
        if user_id == target_user_id {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.set_friend_remark rejected: cannot remark self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }

        if !self
            .follows_repo
            .is_following(user_id, target_user_id)
            .await?
        {
            tracing::warn!(
                user_id,
                target_user_id,
                "user.set_friend_remark rejected: target is not followed friend"
            );
            return Err(ApiError::NotFriends);
        }

        if trimmed.is_empty() {
            self.user_friend_remarks_repo
                .delete(user_id, target_user_id)
                .await?;
            tracing::info!(
                user_id,
                target_user_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "user.set_friend_remark completed (deleted)"
            );
        } else {
            self.user_friend_remarks_repo
                .upsert(user_id, target_user_id, trimmed.to_string())
                .await?;
            tracing::info!(
                user_id,
                target_user_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "user.set_friend_remark completed (upserted)"
            );
        }

        Ok(())
    }

    pub async fn admin_search_users(
        &self,
        req: &AdminUserSearchRequest,
    ) -> Result<PaginatedData<AdminUserSearchItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let user_id_filter = req
            .user_id
            .as_ref()
            .map(|id| Validator::parse_user_id(id))
            .transpose()?;
        let status_filter = req.status.map(|s| match s {
            crate::types::enums::UserAccountStatus::Normal => {
                crate::types::enums::UserStatus::Normal
            }
            crate::types::enums::UserAccountStatus::Banned => {
                crate::types::enums::UserStatus::Banned
            }
        });
        let (rows, next_cursor) = self
            .user_repo
            .admin_search(
                user_id_filter,
                req.nickname.as_deref(),
                req.login_id.as_deref(),
                status_filter,
                req.user_type,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let presence_status = if row.is_online {
                UserPresenceStatus::Online
            } else {
                UserPresenceStatus::Offline
            };
            items.push(AdminUserSearchItem {
                user_id: row.user_id,
                nickname: row.nickname.unwrap_or_default(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(row.avatar_url.as_deref())
                    .await,
                user_type: row.user_type,
                status: row.status.into(),
                created_at: row.created_at.to_rfc3339(),
                last_active_at: row.last_active_at.map(|t| t.to_rfc3339()),
                login_id: row.login_id,
                presence_status,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    /// 管理员封禁用户（包含业务规则校验）
    ///
    /// - 不能封禁自己
    /// - 非超管不能封禁管理员
    pub async fn admin_ban_user(
        &self,
        operator_id: i64,
        target_user_id: i64,
        is_target_admin: bool,
        is_operator_super: bool,
        reason: Option<String>,
        duration_days: Option<i32>,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id,
            target_user_id,
            is_target_admin,
            is_operator_super,
            has_reason = reason.is_some(),
            duration_days = duration_days.unwrap_or(0),
            "user.admin_ban started"
        );
        if operator_id == target_user_id {
            tracing::warn!(
                operator_id,
                target_user_id,
                "user.admin_ban rejected: cannot ban self"
            );
            return Err(ApiError::CannotOperateOnSelf);
        }
        if is_target_admin && !is_operator_super {
            tracing::warn!(
                operator_id,
                target_user_id,
                "user.admin_ban rejected: non-super admin cannot ban admin"
            );
            return Err(ApiError::CannotOperateAdmin(
                "无法操作管理员账户".to_string(),
            ));
        }

        self.user_repo
            .update_status(target_user_id, UserStatus::Banned)
            .await?;

        self.event_bus
            .publish(crate::events::event_types::AppEvent::AccountBanned {
                user_id: target_user_id,
                reason: reason.unwrap_or_else(|| "违规行为".to_string()),
                duration_days: duration_days.unwrap_or(0),
            });
        tracing::info!(
            operator_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.admin_ban completed"
        );

        Ok(())
    }

    pub async fn admin_unban_user(&self, target_user_id: i64) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(target_user_id, "user.admin_unban started");
        self.user_repo
            .update_status(target_user_id, UserStatus::Normal)
            .await?;
        tracing::info!(
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "user.admin_unban completed"
        );
        Ok(())
    }

    pub async fn admin_user_detail(&self, user_id: i64) -> Result<AdminUserDetail> {
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| ApiError::UserNotFound("用户不存在".to_string()))?;

        let tags = self
            .tag_repo
            .get_user_tags(user_id)
            .await
            .unwrap_or_default();
        let stats = self.user_repo.get_stats(user_id).await?;
        let (is_online, last_online) = self.presence_service.get_online_status(user_id).await;
        let location = self.build_location_for_user(user_id).await;
        let presence_status = if is_online {
            UserPresenceStatus::Online
        } else {
            UserPresenceStatus::Offline
        };

        Ok(AdminUserDetail {
            user_id: user.user_id,
            nickname: user.nickname.unwrap_or_default(),
            avatar_url: self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
            user_type: user.user_type,
            status: user.status.into(),
            created_at: user.created_at.to_rfc3339(),
            last_active_at: last_online,
            login_id: user.username,
            email: user.email,
            phone: user.phone,
            bio: user.bio,
            tags: Some(tags),
            location,
            presence_status,
            stats: UserStats {
                moments_count: stats.moments_count,
                friends_count: stats.following_count,
                followers_count: stats.followers_count,
                following_count: stats.following_count,
                groups_count: stats.groups_count,
            },
            ban_history: None,
        })
    }

    async fn build_user_search_list(
        &self,
        rows: Vec<crate::repositories::user_repo::UserSearchRow>,
        next_cursor: Option<String>,
        viewer_id: Option<i64>,
    ) -> Result<PaginatedData<UserItem>> {
        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let location_visible_map = self.presence_service.batch_get_location_visible(&ids).await;
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let (stats_map, relation_map, mutual_map) = if let Some(uid) = viewer_id {
            let (stats_map, relation_map, mutual_map) = tokio::try_join!(
                self.user_repo.batch_get_stats(&ids),
                self.batch_get_relationships(uid, &ids),
                self.follows_repo.count_mutual_friends_batch(uid, &ids),
            )?;
            (Some(stats_map), Some(relation_map), Some(mutual_map))
        } else {
            (None, None, None)
        };

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let (is_online, last_online) = online_map
                .get(&row.user_id)
                .cloned()
                .unwrap_or((None, None));
            let location = if location_visible_map
                .get(&row.user_id)
                .copied()
                .unwrap_or(true)
            {
                Some(RelativeLocation::with_distance(
                    0.0,
                    0.0,
                    "",
                    row.distance.unwrap_or(0.0),
                    row.bearing_degrees.unwrap_or(0.0),
                ))
            } else {
                None
            };

            let (stats, relation, social_context) =
                if let (Some(ref sm), Some(ref rm), Some(ref mm)) =
                    (&stats_map, &relation_map, &mutual_map)
                {
                    let stats = Some(sm.get(&row.user_id).map_or(
                        UserStats {
                            moments_count: 0,
                            friends_count: 0,
                            followers_count: 0,
                            following_count: 0,
                            groups_count: 0,
                        },
                        |s| UserStats {
                            moments_count: s.moments_count,
                            friends_count: s.following_count,
                            followers_count: s.followers_count,
                            following_count: s.following_count,
                            groups_count: s.groups_count,
                        },
                    ));
                    let relation = rm.get(&row.user_id).cloned();
                    let mutual_friends_count = mm.get(&row.user_id).copied().unwrap_or(0);
                    let social_context = Some(SocialContext {
                        mutual_friends_count,
                        mutual_friends_preview: None,
                        common_groups_count: None,
                        common_tags: None,
                    });
                    (stats, relation, social_context)
                } else {
                    (None, None, None)
                };

            items.push(UserItem {
                user: UserInfo::new(
                    row.user_id,
                    row.nickname.unwrap_or_default(),
                    self.media_service
                        .sign_url_if_path(row.avatar_url.as_deref())
                        .await,
                ),
                bio: row.bio,
                tags: None,
                presence_status: is_online.map(|online| {
                    if online {
                        UserPresenceStatus::Online
                    } else {
                        UserPresenceStatus::Offline
                    }
                }),
                last_online,
                location,
                stats,
                relation,
                social_context,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    async fn build_user_info(&self, user: &UserRow) -> UserInfo {
        UserInfo::new(
            user.user_id,
            user.nickname.clone().unwrap_or_default(),
            self.media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
        )
    }

    async fn build_user_card_stats(&self, user_id: i64) -> Result<UserStats> {
        let stats = self.user_repo.get_stats(user_id).await?;
        Ok(UserStats {
            moments_count: stats.moments_count,
            friends_count: stats.following_count,
            followers_count: stats.followers_count,
            following_count: stats.following_count,
            groups_count: stats.groups_count,
        })
    }

    /// 批量获取关系（is_following, is_followed_by, is_blocked）
    async fn batch_get_relationships(
        &self,
        current_user_id: i64,
        target_ids: &[i64],
    ) -> Result<HashMap<i64, Relationship>> {
        if target_ids.is_empty() {
            return Ok(HashMap::new());
        }
        let (following_set, followed_by_set, blocked_set) = tokio::try_join!(
            self.follows_repo
                .batch_check_following(current_user_id, target_ids),
            self.follows_repo
                .batch_check_followed_by(current_user_id, target_ids),
            self.blocks_repo
                .batch_check_blocked(current_user_id, target_ids),
        )?;

        let mut map = HashMap::with_capacity(target_ids.len());
        for &target_id in target_ids {
            map.insert(
                target_id,
                Relationship {
                    is_following: following_set.contains(&target_id),
                    is_followed_by: followed_by_set.contains(&target_id),
                    is_blocked: blocked_set.contains(&target_id),
                },
            );
        }
        Ok(map)
    }

    async fn build_location_for_user(&self, user_id: i64) -> Option<RelativeLocation> {
        if let Ok(Some((lat, lng))) = self.user_repo.get_current_location(user_id).await {
            Some(RelativeLocation::without_distance(lat, lng, ""))
        } else {
            None
        }
    }

    fn map_settings(&self, settings: Option<&UserSettingsRow>) -> UserSettingsData {
        UserSettingsData {
            privacy_level: self.privacy_level(settings, "privacy_level", PrivacyLevel::Public),
            location_visible: self.privacy_bool(settings, "location_visible", true),
            presence_visible: self.presence_visible(settings),
            social_lists_visible: self.social_lists_visible_from_settings(settings),
            groups_visible: self.privacy_bool(settings, "groups_visible", true),
            liked_moments_visible: self.privacy_bool(settings, "liked_moments_visible", true),
            stats_visible: self.privacy_bool(settings, "stats_visible", true),
            allow_follow: self.privacy_bool(settings, "allow_follow", true),
            allow_private_chat: self.privacy_chat_permission(
                settings,
                "allow_private_chat",
                PrivateChatPermission::Everyone,
            ),
            searchable: self.privacy_bool(settings, "searchable", true),
            phone_searchable: self.privacy_bool(settings, "phone_searchable", true),
            nickname_searchable: self.privacy_bool(settings, "nickname_searchable", true),
            location_searchable: self.privacy_bool(settings, "location_searchable", true),
        }
    }

    fn privacy_bool(&self, settings: Option<&UserSettingsRow>, key: &str, default: bool) -> bool {
        settings
            .and_then(|s| s.privacy_settings.get(key))
            .and_then(|v| v.as_bool())
            .unwrap_or(default)
    }

    /// 在线状态和最后在线时间的统一可见性控制
    fn presence_visible(&self, settings: Option<&UserSettingsRow>) -> bool {
        self.privacy_bool(settings, "presence_visible", true)
    }

    fn privacy_level(
        &self,
        settings: Option<&UserSettingsRow>,
        key: &str,
        default: PrivacyLevel,
    ) -> PrivacyLevel {
        settings
            .and_then(|s| s.privacy_settings.get(key))
            .and_then(|v| v.as_str())
            .and_then(|v| PrivacyLevel::from_str(v).ok())
            .unwrap_or(default)
    }

    fn privacy_chat_permission(
        &self,
        settings: Option<&UserSettingsRow>,
        key: &str,
        default: PrivateChatPermission,
    ) -> PrivateChatPermission {
        settings
            .and_then(|s| s.privacy_settings.get(key))
            .and_then(|v| v.as_str())
            .and_then(|v| PrivateChatPermission::from_str(v).ok())
            .unwrap_or(default)
    }

    async fn bool_from_settings(&self, user_id: i64, key: &str, default: bool) -> bool {
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();
        self.privacy_bool(settings.as_ref(), key, default)
    }

    async fn location_visible_for_user(&self, user_id: i64) -> bool {
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();
        self.privacy_bool(settings.as_ref(), "location_visible", true)
    }

    fn social_lists_visible_from_settings(&self, settings: Option<&UserSettingsRow>) -> bool {
        let s = settings;
        self.privacy_bool(s, "friends_visible", true)
            || self.privacy_bool(s, "following_visible", true)
            || self.privacy_bool(s, "followers_visible", true)
    }

    /// 社交列表是否可见：friends_visible / following_visible / followers_visible 中任一为 true 即可见
    async fn social_lists_visible(&self, user_id: i64) -> bool {
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();
        self.social_lists_visible_from_settings(settings.as_ref())
    }

    async fn build_user_moment_items(
        &self,
        rows: Vec<MomentDetailRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_ids: Vec<Uuid> = rows.iter().map(|r| r.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;
        let images_map = self.moment_repo.list_images_batch(&moment_ids).await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let tags = tags_map.get(&row.moment_id).cloned().unwrap_or_default();
            let images = images_map.get(&row.moment_id);
            let image_list = match images {
                Some(rows) => self.sign_images(rows).await,
                None => None,
            };

            let user_info = UserInfo::new(
                row.user_id,
                row.user_nickname.clone().unwrap_or_default(),
                self.media_service
                    .sign_url_if_path(row.user_avatar_url.as_deref())
                    .await,
            );
            items.push(MomentItem {
                moment_id: row.moment_id.to_string(),
                description: row.description.unwrap_or_default(),
                images: image_list,
                tags,
                mentions: row.mentions.clone(),
                created_at: row.created_at.to_rfc3339(),
                likes_count: row.like_count,
                comments_count: row.comments_count,
                shares_count: 0,
                user: user_info,
                location: RelativeLocation::without_distance(
                    row.latitude.unwrap_or(0.0),
                    row.longitude.unwrap_or(0.0),
                    row.location_name.unwrap_or_default(),
                ),
                privacy_level: None,
                my_state: None,
                recommendation: None,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    async fn sign_images(
        &self,
        rows: &[MomentImageRow],
    ) -> Option<Vec<crate::dto::response::MomentImage>> {
        if rows.is_empty() {
            return None;
        }

        let mut images = Vec::with_capacity(rows.len());
        for row in rows {
            let signed = self.media_service.sign_url_if_path(Some(&row.url)).await;
            images.push(crate::dto::response::MomentImage {
                url: signed.unwrap_or_else(|| row.url.clone()),
                thumbnail_url: None,
                width: row.width,
                height: row.height,
                mime_type: None,
            });
        }
        Some(images)
    }
}
