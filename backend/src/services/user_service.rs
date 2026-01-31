use std::collections::HashMap;
use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use chrono::{Duration, Utc};

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE, LOCATION_CHANGE_THRESHOLD},
    dto::{
        request::{
            AdminBanUserRequest, AdminUnbanUserRequest, AdminUserDetailRequest,
            AdminUserSearchRequest, BlockRequest, DeleteAccountRequest, FollowRequest,
            GetRelationshipRequest, HandleFollowRequestRequest, MyBlockedRequest,
            MyFollowRequestsRequest, MyFollowersRequest, MyFollowingRequest, MyFriendsRequest,
            SearchUsersByKeywordRequest, SearchUsersByLocationRequest, SearchUsersByTagRequest,
            SetRemarkRequest, UnblockRequest, UnfollowRequest, UpdateLocationRequest,
            UpdateProfileRequest, UpdateSettingsRequest, UpgradeAccountRequest,
            UserFollowersRequest, UserFollowingRequest, UserFriendsRequest, UserGroupsRequest,
            UserLikedMomentsRequest, UserOpenTheDoorRequest, UserStatsRequest,
        },
        response::{
            AdminUserDetail, AdminUserItem, AuthData, BlockedUserItem, FollowRequestItem,
            FollowerItem, FollowingItem, FriendItem, GroupItem, Location, MomentImage, MomentItem,
            MyProfileData, PaginatedData, PaginationMeta, RelativeLocation, UserCard,
            UserCardStats, UserInfo, UserItem, UserRelation, UserSettingsData, UserStatsData,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        device_token_repo::DeviceTokenRepository, follow_request_repo::FollowRequestRepository,
        session_repo::SessionRepository, tag_repo::TagRepository,
        user_blocks_repo::UserBlocksRepository, user_follows_repo::UserFollowsRepository,
        user_friend_remarks_repo::UserFriendRemarksRepository,
        user_location_history_repo::UserLocationHistoryRepository, user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    types::enums::{PrivacyLevel, PrivateChatPermission, UserAccountStatus},
    utils::{
        cache::{cache_keys, CacheManager},
        jwt::JwtUtil,
        password::PasswordUtil,
        token::TokenUtil,
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
    follow_request_repo: FollowRequestRepository,
    blocks_repo: UserBlocksRepository,
    remarks_repo: UserFriendRemarksRepository,
    device_token_repo: DeviceTokenRepository,
    session_repo: SessionRepository,
    tag_repo: TagRepository,
    group_member_repo: crate::repositories::group_member_repo::GroupMemberRepository,
    moment_repo: crate::repositories::moment_repo::MomentRepository,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    cache_manager: CacheManager,
}

impl UserService {
    pub fn new(
        event_bus: Arc<EventBus>,
        user_repo: UserRepository,
        user_settings_repo: UserSettingsRepository,
        user_location_history_repo: UserLocationHistoryRepository,
        follows_repo: UserFollowsRepository,
        follow_request_repo: FollowRequestRepository,
        blocks_repo: UserBlocksRepository,
        remarks_repo: UserFriendRemarksRepository,
        device_token_repo: DeviceTokenRepository,
        session_repo: SessionRepository,
        tag_repo: TagRepository,
        group_member_repo: crate::repositories::group_member_repo::GroupMemberRepository,
        moment_repo: crate::repositories::moment_repo::MomentRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            event_bus,
            user_repo,
            user_settings_repo,
            user_location_history_repo,
            follows_repo,
            follow_request_repo,
            blocks_repo,
            remarks_repo,
            device_token_repo,
            session_repo,
            tag_repo,
            group_member_repo,
            moment_repo,
            presence_service,
            media_service,
            cache_manager,
        }
    }

    /// 透传 cursor 字符串
    fn parse_cursor(cursor: Option<&str>) -> Option<String> {
        cursor.map(|s| s.to_string())
    }

    fn default_user_settings() -> UserSettingsData {
        UserSettingsData {
            privacy_level: PrivacyLevel::Public,
            location_visible: true,
            online_visible: true,
            last_online_visible: true,
            friends_visible: PrivacyLevel::Public,
            following_visible: PrivacyLevel::Public,
            followers_visible: PrivacyLevel::Public,
            groups_visible: PrivacyLevel::Public,
            moments_visible: PrivacyLevel::Public,
            liked_moments_visible: PrivacyLevel::Public,
            stats_visible: PrivacyLevel::Public,
            allow_follow: true,
            allow_private_chat: PrivateChatPermission::Everyone,
            searchable: true,
            phone_searchable: true,
            nickname_searchable: true,
            location_searchable: true,
        }
    }

    /// 按关键字搜索用户（全局，距离由 PostGIS 计算）
    pub async fn search_by_keyword(
        &self,
        req: &SearchUsersByKeywordRequest,
    ) -> Result<PaginatedData<UserItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;

        let (users, next_cursor) = self
            .user_repo
            .search_by_keyword(
                &req.kw,
                req.reference_point.latitude,
                req.reference_point.longitude,
                limit as i64,
                cursor,
            )
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 按标签搜索用户（全局，距离由 PostGIS 计算）
    pub async fn search_by_tag(
        &self,
        req: &SearchUsersByTagRequest,
    ) -> Result<PaginatedData<UserItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;

        // 从标签库获取用户ID列表
        let (user_ids, next_cursor) = self
            .tag_repo
            .search_users_by_tag(&req.tag, limit as i64, cursor)
            .await?;

        // 根据ID获取用户信息并计算距离
        let users = self
            .user_repo
            .find_by_ids_with_distance(
                &user_ids,
                req.reference_point.latitude,
                req.reference_point.longitude,
            )
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 按位置搜索用户（范围过滤，距离由 PostGIS 计算）
    pub async fn search_by_location(
        &self,
        req: &SearchUsersByLocationRequest,
    ) -> Result<PaginatedData<UserItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let (users, next_cursor) = self
            .user_repo
            .search_by_location(
                req.kw.as_deref(),
                req.reference_point.latitude,
                req.reference_point.longitude,
                req.radius,
                limit as i64,
                cursor,
            )
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 获取用户资料（通过 user_id 获取指定用户的资料，包含在线状态，受隐私设置控制）
    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &UserOpenTheDoorRequest,
    ) -> Result<UserCard> {
        let user_id = req.user_id;

        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        if let Some(viewer_id) = viewer_id {
            if viewer_id != user_id && self.blocks_repo.is_blocked(user_id, viewer_id).await? {
                return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
            }
        }

        // 获取用户标签
        let tags = match self.tag_repo.get_user_tags(user_id).await {
            Ok(tags) => {
                if tags.is_empty() {
                    None
                } else {
                    Some(tags)
                }
            }
            Err(e) => {
                tracing::warn!(user_id, error = %e, "get_user_tags failed");
                None
            }
        };

        // 获取在线状态（受隐私设置控制）
        let (is_online, last_online) = self
            .presence_service
            .get_online_status_with_privacy(user_id)
            .await;

        // 检查位置可见性设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();
        let location_visible = settings
            .as_ref()
            .map(|s| s.location_visible)
            .unwrap_or(true);
        let location = if location_visible {
            if let (Some(lat), Some(lng), Some(name)) = (
                user.current_latitude,
                user.current_longitude,
                user.current_location_name.clone(),
            ) {
                Some(Location {
                    latitude: lat,
                    longitude: lng,
                    location_name: name,
                })
            } else {
                None
            }
        } else {
            None
        };

        // 获取统计数据
        let stats_data = self.get_stats(user_id).await?;
        let stats = UserCardStats {
            moments_count: stats_data.moments_count,
            friends_count: stats_data.friends_count,
            followers_count: stats_data.followers_count,
            following_count: stats_data.following_count,
        };

        let relation = if let Some(current_user_id) = viewer_id {
            if current_user_id == user_id {
                None
            } else {
                Some(
                    self.get_relationship(current_user_id, &GetRelationshipRequest { user_id })
                        .await?,
                )
            }
        } else {
            None
        };

        Ok(UserCard {
            user_id: user.user_id,
            nickname: user.nickname.clone(),
            avatar_url: self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
            user_type: user.user_type,
            bio: if user.bio.is_empty() {
                None
            } else {
                Some(user.bio.clone())
            },
            tags,
            is_online,
            last_online,
            location,
            stats,
            relation,
        })
    }

    /// Helper: 构建用户搜索结果（距离和方位角由 PostGIS 计算，包含在线状态，受隐私设置控制）
    async fn build_user_search_results(
        &self,
        users: Vec<crate::repositories::user_repo::UserSearchRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserItem>> {
        // 批量获取标签
        let user_ids: Vec<i64> = users.iter().map(|u| u.user_id).collect();
        let tags_map = self.tag_repo.get_user_tags_batch(&user_ids).await?;

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
        for user in users {
            let (is_online, last_online) = online_status_map
                .get(&user.user_id)
                .cloned()
                .unwrap_or((None, None));

            // 检查位置是否可见
            let location_visible = location_visible_map
                .get(&user.user_id)
                .copied()
                .unwrap_or(true);

            let location = if location_visible {
                Some(RelativeLocation {
                    latitude: user.latitude,
                    longitude: user.longitude,
                    location_name: user.location_name.clone(),
                    distance: user.distance,
                    bearing_degrees: user.bearing_degrees,
                })
            } else {
                None
            };

            items.push(UserItem {
                user_id: user.user_id,
                nickname: user.nickname.clone(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(user.avatar_url.as_deref())
                    .await,
                bio: if user.bio.is_empty() {
                    None
                } else {
                    Some(user.bio.clone())
                },
                tags: tags_map.get(&user.user_id).cloned(),
                is_online,
                last_online,
                user_type: user.user_type,
                location,
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

    /// 获取与指定用户的关系
    pub async fn get_relationship(
        &self,
        current_user_id: i64,
        req: &GetRelationshipRequest,
    ) -> Result<UserRelation> {
        let target_user_id = req.user_id;

        let is_following = self
            .follows_repo
            .exists(current_user_id, target_user_id)
            .await?;
        let is_followed_by = self
            .follows_repo
            .exists(target_user_id, current_user_id)
            .await?;
        // is_blocked: 我是否拉黑了对方
        let is_blocked = self
            .blocks_repo
            .is_blocked(current_user_id, target_user_id)
            .await?;
        // 注意：不返回"对方是否拉黑了我"，这是对方的隐私
        let has_pending_follow_request = self
            .follow_request_repo
            .has_pending_request(current_user_id, target_user_id)
            .await?;

        Ok(UserRelation {
            is_mutual_follow: is_following && is_followed_by,
            is_following,
            is_followed_by,
            is_blocked,
            has_pending_follow_request,
        })
    }

    /// 更新用户位置
    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        Validator::validate_coordinates(req.location.latitude, req.location.longitude)?;

        // 使用 PostGIS 计算与最新位置的距离
        let distance = self
            .user_location_history_repo
            .get_distance_from_latest(user_id, req.location.latitude, req.location.longitude)
            .await?;

        // 检查是否需要插入新记录：没有历史位置或超过阈值
        let should_insert = match distance {
            Some(d) => d > LOCATION_CHANGE_THRESHOLD,
            None => true,
        };

        if should_insert {
            self.user_location_history_repo
                .record(
                    user_id,
                    req.location.latitude,
                    req.location.longitude,
                    Some(req.location.location_name.clone()),
                )
                .await?;
        }

        // 同时更新 users 表的当前位置
        self.user_repo
            .update_location(
                user_id,
                req.location.latitude,
                req.location.longitude,
                Some(req.location.location_name.as_str()),
            )
            .await?;

        // 更新 Redis 缓存
        self.presence_service
            .update_location_cache(user_id, req.location.latitude, req.location.longitude)
            .await
            .ok();

        // 发布位置更新事件
        self.event_bus.publish(AppEvent::UserLocationUpdated {
            user_id,
            latitude: req.location.latitude,
            longitude: req.location.longitude,
            location_name: req.location.location_name.clone(),
        });

        Ok(())
    }

    /// 更新用户资料
    pub async fn update_profile(&self, user_id: i64, req: &UpdateProfileRequest) -> Result<()> {
        if let Some(nickname) = &req.nickname {
            Validator::validate_not_empty(nickname, "昵称")?;
            Validator::validate_length(nickname, "昵称", 1, 100)?;
        }

        // 解析 avatar_file_id 为 (url, path)
        let (_avatar_url, avatar_storage_path) = match &req.avatar_file_id {
            Some(file_id) => {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的文件ID格式".to_string()))?;
                let file_data = self
                    .media_service
                    .validate_file(uuid, user_id, "avatar")
                    .await?;
                self.media_service.mark_as_used(uuid).await?;
                (Some(file_data.clone()), Some(file_data))
            }
            None => (None, None),
        };

        // Note: For user profile, we might still be storing URL in DB for now
        // But for consistency with S3 presign expiration, we should store storage_path.
        // However, the `user_repo.update_profile` expects `avatar_url`.
        // If we store `storage_path` in `avatar_url` column, we need to sign it on read.
        // Let's assume we store `storage_path` in `avatar_url` column for now,
        // as `avatar_url` in users table is just a string.

        self.user_repo
            .update_profile(
                user_id,
                req.nickname.as_deref(),
                avatar_storage_path.as_deref(), // Store storage_path
                req.bio.as_deref(),
            )
            .await?;

        // 更新用户标签
        if let Some(tags) = &req.tags {
            self.tag_repo.set_user_tags(user_id, tags).await?;
        }

        // 清除 Profile 缓存
        let cache_key = user_id.to_string();
        if let Err(e) = self
            .cache_manager
            .delete(cache_keys::USER_INFO, &cache_key)
            .await
        {
            tracing::warn!(user_id, error = %e, "delete user info cache failed");
        }

        Ok(())
    }

    // 好友 = 互相关注，无独立好友申请流程

    /// 关注用户
    pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
        let to_user_id = req.user_id;

        // 1. 自我操作检查
        if from_user_id == to_user_id {
            tracing::warn!(
                user_id = from_user_id,
                action = "follow_user",
                reason = "self_operation",
                error_code = 10411,
                "关注失败：不能关注自己"
            );
            return Err(AppError::BadRequest("不能关注自己".to_string()));
        }

        // 2. 检查目标用户存在
        let target_user = self
            .user_repo
            .find_by_id(to_user_id)
            .await?
            .ok_or_else(|| {
                tracing::warn!(
                    from_user_id,
                    to_user_id,
                    action = "follow_user",
                    reason = "target_not_found",
                    error_code = 10310,
                    "关注失败：目标用户不存在"
                );
                AppError::UserNotFound("用户不存在".to_string())
            })?;

        // 3. 系统用户检查
        if target_user.user_type == crate::types::enums::UserType::System {
            tracing::warn!(
                from_user_id,
                to_user_id,
                user_type = ?target_user.user_type,
                action = "follow_user",
                reason = "system_user",
                error_code = 10213,
                "关注失败：系统用户不允许被关注"
            );
            return Err(AppError::FollowNotAllowed("对方不允许被关注".to_string()));
        }

        // 4. 黑名单检查：对方拉黑我时禁止关注
        if self
            .blocks_repo
            .is_blocked(to_user_id, from_user_id)
            .await?
        {
            tracing::warn!(
                from_user_id,
                to_user_id,
                action = "follow_user",
                reason = "blocked_by_target",
                error_code = 10211,
                "关注失败：已被对方拉黑"
            );
            return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
        }

        // 已关注则视为幂等成功
        if self.follows_repo.exists(from_user_id, to_user_id).await? {
            if self
                .follows_repo
                .are_mutual_follows(from_user_id, to_user_id)
                .await?
            {
                let err = AppError::AlreadyFriends;
                tracing::debug!(code = err.error_code(), "already friends");
            }
            return Ok(());
        }

        // 检查目标用户的 allow_follow 设置
        let target_settings = self.user_settings_repo.find_by_user_id(to_user_id).await?;
        if let Some(settings) = target_settings {
            if !settings.allow_follow {
                // 需要审批：创建关注请求
                let request_id = self
                    .follow_request_repo
                    .create(from_user_id, to_user_id, None)
                    .await?;
                self.event_bus.publish(AppEvent::FollowRequestReceived {
                    request_id: request_id.to_string(),
                    requester_id: from_user_id,
                    target_id: to_user_id,
                });
                return Ok(());
            }
        }

        // 创建关注关系
        self.follows_repo.create(from_user_id, to_user_id).await?;

        // 关注成功后发布事件（通知通过 WS 推送；推送侧会再检查“偏好 + 订阅”）
        let is_mutual = self.follows_repo.exists(to_user_id, from_user_id).await?;
        self.event_bus.publish(AppEvent::NewFollower {
            follower_id: from_user_id,
            user_id: to_user_id,
            is_mutual,
        });
        if is_mutual {
            self.event_bus.publish(AppEvent::MutualFollow {
                user_id: from_user_id,
                mutual_user_id: to_user_id,
            });
        }
        Ok(())
    }

    /// 取消关注用户
    pub async fn unfollow_user(&self, from_user_id: i64, req: &UnfollowRequest) -> Result<()> {
        let to_user_id = req.user_id;

        if from_user_id == to_user_id {
            tracing::warn!(
                user_id = from_user_id,
                action = "unfollow_user",
                reason = "self_operation",
                error_code = 10411,
                "取消关注失败：不能取消关注自己"
            );
            return Err(AppError::BadRequest("不能取消关注自己".to_string()));
        }

        // 强约束：未关注则返回冲突错误
        if !self.follows_repo.exists(from_user_id, to_user_id).await? {
            tracing::warn!(
                from_user_id,
                to_user_id,
                action = "unfollow_user",
                reason = "not_following",
                error_code = 10410,
                "取消关注失败：尚未关注该用户"
            );
            return Err(AppError::Conflict("尚未关注该用户".to_string()));
        }

        self.follows_repo
            .delete(from_user_id, to_user_id)
            .await
            .map_err(|e| {
                tracing::error!(
                    from_user_id,
                    to_user_id,
                    action = "unfollow_user",
                    error = %e,
                    error_type = "database",
                    "取消关注失败"
                );
                e
            })?;

        tracing::info!(
            from_user_id,
            to_user_id,
            action = "unfollow_user",
            result = "success",
            "取消关注成功"
        );
        Ok(())
    }

    /// 拉黑用户
    pub async fn block_user(&self, from_user_id: i64, req: &BlockRequest) -> Result<()> {
        let to_user_id = req.user_id;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能拉黑自己".to_string()));
        }

        // 强约束：已拉黑则返回冲突错误
        if self.blocks_repo.exists(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("已拉黑该用户".to_string()));
        }

        // 拉黑时取消关注（双向）
        self.follows_repo
            .delete(from_user_id, to_user_id)
            .await
            .ok();
        self.follows_repo
            .delete(to_user_id, from_user_id)
            .await
            .ok();

        // 拉黑时清理待处理的关注请求（双向）
        self.follow_request_repo
            .delete(from_user_id, to_user_id)
            .await
            .ok();
        self.follow_request_repo
            .delete(to_user_id, from_user_id)
            .await
            .ok();

        let inserted = self
            .blocks_repo
            .create(from_user_id, to_user_id, req.reason.clone())
            .await?;
        if !inserted {
            // 极端并发：存在性检查后仍发生冲突
            return Err(AppError::Conflict("已拉黑该用户".to_string()));
        }
        Ok(())
    }

    /// 解除拉黑用户
    pub async fn unblock_user(&self, from_user_id: i64, req: &UnblockRequest) -> Result<()> {
        let to_user_id = req.user_id;

        if from_user_id == to_user_id {
            tracing::warn!(
                user_id = from_user_id,
                action = "unblock_user",
                reason = "self_operation",
                error_code = 10411,
                "解除拉黑失败：不能解除拉黑自己"
            );
            return Err(AppError::BadRequest("不能解除拉黑自己".to_string()));
        }

        // 强约束：未拉黑则返回冲突错误
        if !self.blocks_repo.exists(from_user_id, to_user_id).await? {
            tracing::warn!(
                from_user_id,
                to_user_id,
                action = "unblock_user",
                reason = "not_blocked",
                error_code = 10410,
                "解除拉黑失败：尚未拉黑该用户"
            );
            return Err(AppError::Conflict("尚未拉黑该用户".to_string()));
        }

        self.blocks_repo
            .delete(from_user_id, to_user_id)
            .await
            .map_err(|e| {
                tracing::error!(
                    from_user_id,
                    to_user_id,
                    action = "unblock_user",
                    error = %e,
                    error_type = "database",
                    "解除拉黑失败"
                );
                e
            })?;

        tracing::info!(
            from_user_id,
            to_user_id,
            action = "unblock_user",
            result = "success",
            "解除拉黑成功"
        );
        Ok(())
    }

    /// 设置好友备注
    pub async fn set_friend_remark(&self, user_id: i64, req: &SetRemarkRequest) -> Result<()> {
        let friend_id = req.user_id;

        // 好友 = 互相关注
        if !self
            .follows_repo
            .are_mutual_follows(user_id, friend_id)
            .await?
        {
            return Err(AppError::NotFriends);
        }

        if let Some(remark) = &req.remark {
            if !remark.is_empty() {
                self.remarks_repo
                    .upsert(user_id, friend_id, remark.clone())
                    .await?;
            } else {
                self.remarks_repo.delete(user_id, friend_id).await?;
            }
        } else {
            self.remarks_repo.delete(user_id, friend_id).await?;
        }

        Ok(())
    }

    /// 批量获取用户简介（优先查 Redis 缓存）
    async fn get_user_profiles_cached(
        &self,
        user_ids: &[i64],
    ) -> Result<Vec<crate::repositories::user_repo::UserProfileRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }

        let mut profiles = Vec::with_capacity(user_ids.len());
        let mut missing_ids = Vec::new();
        let cache_ids: Vec<String> = user_ids.iter().map(|id| id.to_string()).collect();

        // 1. 尝试从缓存获取（批量）
        let cached = match self
            .cache_manager
            .mget::<crate::repositories::user_repo::UserProfileRow>(
                cache_keys::USER_INFO,
                &cache_ids,
            )
            .await
        {
            Ok(v) => v,
            Err(e) => {
                tracing::warn!(error = %e, "mget user info cache failed");
                Vec::new()
            }
        };

        if !cached.is_empty() {
            for (id, value) in cached {
                if let Some(profile) = value {
                    profiles.push(profile);
                } else if let Ok(parsed_id) = id.parse::<i64>() {
                    missing_ids.push(parsed_id);
                }
            }
        } else {
            missing_ids.extend_from_slice(user_ids);
        }

        // 2. 如果有缺失，从 DB 获取
        if !missing_ids.is_empty() {
            let db_profiles = self.user_repo.find_profiles(&missing_ids).await?;
            let mut cache_items = Vec::with_capacity(db_profiles.len());
            for profile in db_profiles {
                // 3. 写入缓存 (TTL 1小时)
                cache_items.push((profile.user_id.to_string(), profile.clone()));
                profiles.push(profile);
            }
            if let Err(e) = self
                .cache_manager
                .mset(cache_keys::USER_INFO, &cache_items)
                .await
            {
                tracing::warn!(error = %e, "mset user info cache failed");
            }
        }

        // 保持输入顺序 (Optional, but good for UI stability)
        // 这里简单返回，调用方通常会构建 Map
        Ok(profiles)
    }

    /// 获取好友列表（好友 = 互相关注）
    pub async fn get_friends_list(
        &self,
        user_id: i64,
        req: &MyFriendsRequest,
    ) -> Result<PaginatedData<FriendItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        // 获取互相关注的用户列表
        let (mutual_follows, next_cursor) = self
            .follows_repo
            .list_mutual_follows(user_id, limit, cursor)
            .await?;

        if mutual_follows.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let friend_ids: Vec<i64> = mutual_follows.iter().map(|r| r.user_id).collect();

        // 批量获取用户信息 (Cached)
        let users = self.get_user_profiles_cached(&friend_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取备注信息
        let remarks = self.remarks_repo.get_batch(user_id, &friend_ids).await?;

        // 批量获取在线状态（Redis 优先，数据库 Fallback）
        let online_status_map = self
            .presence_service
            .batch_get_online_status_raw(&friend_ids)
            .await;

        let mut items = Vec::with_capacity(mutual_follows.len());
        for r in mutual_follows {
            if let Some(user) = user_map.get(&r.user_id) {
                items.push(FriendItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(user.avatar_url.as_deref())
                            .await,
                        user_type: user.user_type,
                    },
                    remark: remarks.get(&r.user_id).cloned(),
                    is_online: online_status_map.get(&r.user_id).copied(),
                    followed_at: r.followed_at.to_rfc3339(),
                    followed_back_at: r.followed_back_at.to_rfc3339(),
                });
            }
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取关注列表
    pub async fn get_following_list(
        &self,
        user_id: i64,
        req: &MyFollowingRequest,
    ) -> Result<PaginatedData<FollowingItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (follow_rows, next_cursor) = self
            .follows_repo
            .list_following_detail(user_id, limit, cursor)
            .await?;

        if follow_rows.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let user_ids: Vec<i64> = follow_rows.iter().map(|r| r.following_id).collect();
        // 批量获取用户信息 (Cached)
        let users = self.get_user_profiles_cached(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取在线状态
        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&user_ids)
            .await;

        let mut items = Vec::with_capacity(follow_rows.len());
        for r in follow_rows {
            if let Some(user) = user_map.get(&r.following_id) {
                let is_online = online_status_map
                    .get(&r.following_id)
                    .and_then(|(online, _)| *online);

                items.push(FollowingItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(user.avatar_url.as_deref())
                            .await,
                        user_type: user.user_type,
                    },
                    followed_at: r.created_at.to_rfc3339(),
                    is_mutual: r.is_mutual,
                    is_online,
                });
            }
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取粉丝列表
    pub async fn get_followers_list(
        &self,
        user_id: i64,
        req: &MyFollowersRequest,
    ) -> Result<PaginatedData<FollowerItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (follower_rows, next_cursor) = self
            .follows_repo
            .list_followers_detail(user_id, limit, cursor)
            .await?;

        if follower_rows.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let user_ids: Vec<i64> = follower_rows.iter().map(|r| r.follower_id).collect();
        // 批量获取用户信息 (Cached)
        let users = self.get_user_profiles_cached(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取在线状态
        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&user_ids)
            .await;

        let mut items = Vec::with_capacity(follower_rows.len());
        for r in follower_rows {
            if let Some(user) = user_map.get(&r.follower_id) {
                let is_online = online_status_map
                    .get(&r.follower_id)
                    .and_then(|(online, _)| *online);

                items.push(FollowerItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(user.avatar_url.as_deref())
                            .await,
                        user_type: user.user_type,
                    },
                    followed_at: r.created_at.to_rfc3339(),
                    is_following_back: r.is_following_back,
                    is_online,
                });
            }
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取指定用户好友列表（公开 API）
    /// 获取指定用户好友列表（公开 API，受 friends_visible 隐私设置控制）
    pub async fn get_user_friends_list(
        &self,
        req: &UserFriendsRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let target_user_id = req.user_id;

        // 检查 friends_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let friends_visible = settings
            .as_ref()
            .map(|s| s.friends_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 friends_visible = 'public' 的用户
        if friends_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (mutual_follows, next_cursor) = self
            .follows_repo
            .list_mutual_follows(target_user_id, limit, cursor)
            .await?;

        if mutual_follows.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let friend_ids: Vec<i64> = mutual_follows.iter().map(|r| r.user_id).collect();
        let users = self.user_repo.find_profiles(&friend_ids).await?;
        let mut items = Vec::with_capacity(users.len());
        for u in users {
            items.push(UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await,
                user_type: u.user_type,
            });
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取指定用户关注列表（公开 API）
    /// 获取指定用户关注列表（公开 API，受 following_visible 隐私设置控制）
    pub async fn get_user_following_list(
        &self,
        req: &UserFollowingRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let target_user_id = req.user_id;

        // 检查 following_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let following_visible = settings
            .as_ref()
            .map(|s| s.following_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 following_visible = 'public' 的用户
        if following_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (user_ids, next_cursor) = self
            .follows_repo
            .list_following(target_user_id, limit, cursor)
            .await?;

        if user_ids.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let users = self.user_repo.find_profiles(&user_ids).await?;
        let mut items = Vec::with_capacity(users.len());
        for u in users {
            items.push(UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await,
                user_type: u.user_type,
            });
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取指定用户粉丝列表（公开 API）
    /// 获取指定用户粉丝列表（公开 API，受 followers_visible 隐私设置控制）
    pub async fn get_user_followers_list(
        &self,
        req: &UserFollowersRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let target_user_id = req.user_id;

        // 检查 followers_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let followers_visible = settings
            .as_ref()
            .map(|s| s.followers_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 followers_visible = 'public' 的用户
        if followers_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (user_ids, next_cursor) = self
            .follows_repo
            .list_followers(target_user_id, limit, cursor)
            .await?;

        if user_ids.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let users = self.user_repo.find_profiles(&user_ids).await?;
        let mut items = Vec::with_capacity(users.len());
        for u in users {
            items.push(UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await,
                user_type: u.user_type,
            });
        }

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取拉黑列表
    pub async fn get_blocked_users_list(
        &self,
        user_id: i64,
        req: &MyBlockedRequest,
    ) -> Result<PaginatedData<BlockedUserItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (block_rows, next_cursor) = self
            .blocks_repo
            .list_blocked_detail(user_id, limit, cursor)
            .await?;

        if block_rows.is_empty() {
            return Ok(PaginatedData {
                items: vec![],
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        let user_ids: Vec<i64> = block_rows.iter().map(|r| r.blocked_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = {
            let mut items_vec = Vec::with_capacity(block_rows.len());
            for r in &block_rows {
                if let Some(user) = user_map.get(&r.blocked_id) {
                    items_vec.push(BlockedUserItem {
                        user: UserInfo {
                            user_id: user.user_id,
                            nickname: user.nickname.clone(),
                            avatar_url: self
                                .media_service
                                .sign_url_if_path(user.avatar_url.as_deref())
                                .await,
                            user_type: user.user_type,
                        },
                        blocked_at: r.created_at.to_rfc3339(),
                        reason: r.reason.clone(),
                    });
                }
            }
            items_vec
        };

        let next_cursor_str = next_cursor;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取我的资料
    pub async fn get_me(&self, user_id: i64) -> Result<MyProfileData> {
        // 1. 获取用户信息
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::UserNotFound("用户不存在".to_string()))?;

        // 2. 解析 login_id 获取邮箱和手机号
        let (email, phone) = user
            .login_id
            .as_ref()
            .map(|id| Self::parse_login_id(id))
            .unwrap_or((None, None));

        // 3. 获取最新位置名称
        let location_name = self
            .user_location_history_repo
            .get_latest(user_id)
            .await?
            .and_then(|loc| loc.location_name);

        // 4. 获取在线状态（Redis 优先，数据库 Fallback）
        let (is_online, last_online) = self.presence_service.get_online_status(user_id).await;

        // 5. 获取用户标签
        let tags = self.tag_repo.get_user_tags(user_id).await?;
        let tags = if tags.is_empty() { None } else { Some(tags) };

        // 6. 构造profile (location 契约：必须返回完整 Location，无数据时使用默认值)
        let location = if let Some(name) = location_name {
            if let Some(loc) = self.user_location_history_repo.get_latest(user_id).await? {
                Location {
                    latitude: loc.latitude,
                    longitude: loc.longitude,
                    location_name: name,
                }
            } else {
                // 有名称但无坐标（可能是默认位置），构造默认 Location
                Location {
                    latitude: 0.0,
                    longitude: 0.0,
                    location_name: name,
                }
            }
        } else {
            // 无位置信息时使用系统默认位置
            Location {
                latitude: 0.0,
                longitude: 0.0,
                location_name: "未知星球".to_string(),
            }
        };

        // 7. 获取用户统计数据
        let stats_data = self.get_stats(user_id).await?;

        let profile = MyProfileData {
            user_id: user.user_id,
            nickname: user.nickname,
            avatar_url: self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
            bio: if user.bio.is_empty() {
                None
            } else {
                Some(user.bio)
            },
            tags,
            gender: user.gender,
            email,
            phone,
            location,
            is_online,
            last_online: last_online.unwrap_or_default(),
            created_at: user.created_at.to_rfc3339(),
            stats: stats_data,
        };

        Ok(profile)
    }

    /// 解析 login_id 获取邮箱和手机号
    fn parse_login_id(login_id: &str) -> (Option<String>, Option<String>) {
        // 邮箱格式检测
        if login_id.contains('@') && login_id.contains('.') {
            return (Some(login_id.to_string()), None);
        }

        // 手机号格式检测（纯数字，长度11位）
        if login_id.len() == 11 && login_id.chars().all(|c| c.is_ascii_digit()) {
            return (None, Some(login_id.to_string()));
        }

        // 国际手机号格式（以+开头）
        if login_id.starts_with('+') && login_id[1..].chars().all(|c| c.is_ascii_digit()) {
            return (None, Some(login_id.to_string()));
        }

        (None, None)
    }

    /// 获取指定用户统计数据（公开 API）
    /// 获取指定用户统计数据（公开 API，受 stats_visible 隐私设置控制）
    /// - 如果 stats_visible != 'public'，返回权限不足错误
    pub async fn get_user_stats(&self, req: &UserStatsRequest) -> Result<UserStatsData> {
        let target_user_id = req.user_id;

        // 检查 stats_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let stats_visible = settings
            .as_ref()
            .map(|s| s.stats_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 stats_visible = 'public' 的用户
        if stats_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.get_stats(target_user_id).await
    }

    /// 获取当前用户统计数据（/me/ API）
    pub async fn get_my_stats(&self, user_id: i64) -> Result<UserStatsData> {
        self.get_stats(user_id).await
    }

    async fn get_stats(&self, user_id: i64) -> Result<UserStatsData> {
        let key = user_id.to_string();
        let cached = self
            .cache_manager
            .get_or_set_with_lock_with_ttl(cache_keys::USER_STATS, &key, 300, 60, || async {
                let (moments_count, friends_count, followers_count, following_count, groups_count) =
                    self.user_repo
                        .get_stats_counts(user_id)
                        .await
                        .map_err(|e| Box::new(e) as Box<dyn std::error::Error + Send + Sync>)?;
                Ok(Some(UserStatsData {
                    moments_count: moments_count as i32,
                    friends_count: friends_count as i32,
                    followers_count: followers_count as i32,
                    following_count: following_count as i32,
                    groups_count: groups_count as i32,
                }))
            })
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        if let Some(stats) = cached {
            Ok(stats)
        } else {
            Ok(UserStatsData {
                moments_count: 0,
                friends_count: 0,
                followers_count: 0,
                following_count: 0,
                groups_count: 0,
            })
        }
    }

    /// 获取用户设置
    pub async fn get_settings(&self, user_id: i64) -> Result<UserSettingsData> {
        let cache_key = user_id.to_string();

        // 1. 尝试从缓存获取
        match self
            .cache_manager
            .get::<UserSettingsData>(cache_keys::USER_SETTINGS, &cache_key)
            .await
        {
            Ok(Some(Some(settings))) => return Ok(settings),
            Ok(Some(None)) => return Ok(Self::default_user_settings()),
            Ok(None) => {}
            Err(e) => {
                tracing::warn!(user_id, error = %e, "get user settings cache failed");
            }
        }

        // 从数据库获取设置
        let settings = self.user_settings_repo.find_by_user_id(user_id).await?;

        // 如果有设置记录，直接返回
        if let Some(s) = settings {
            let data = UserSettingsData {
                privacy_level: s.privacy_level,
                location_visible: s.location_visible,
                online_visible: s.online_visible,
                last_online_visible: s.last_online_visible,
                friends_visible: s.friends_visible,
                following_visible: s.following_visible,
                followers_visible: s.followers_visible,
                groups_visible: s.groups_visible,
                moments_visible: s.moments_visible,
                liked_moments_visible: s.liked_moments_visible,
                stats_visible: s.stats_visible,
                allow_follow: s.allow_follow,
                allow_private_chat: s.allow_private_chat,
                searchable: s.searchable,
                phone_searchable: s.phone_searchable,
                nickname_searchable: s.nickname_searchable,
                location_searchable: s.location_searchable,
            };

            // 写入缓存 (1小时)
            if let Err(e) = self
                .cache_manager
                .set(cache_keys::USER_SETTINGS, &cache_key, &data)
                .await
            {
                tracing::warn!(user_id, error = %e, "set user settings cache failed");
            }

            return Ok(data);
        }

        // 返回默认设置
        if let Err(e) = self
            .cache_manager
            .set_null(cache_keys::USER_SETTINGS, &cache_key)
            .await
        {
            tracing::warn!(user_id, error = %e, "set user settings null cache failed");
        }

        Ok(Self::default_user_settings())
    }

    /// 更新用户设置
    pub async fn update_settings(&self, user_id: i64, req: &UpdateSettingsRequest) -> Result<()> {
        use crate::repositories::user_settings_repo::UpdateSettingsParams;

        // 确保设置记录存在
        self.user_settings_repo.create(user_id).await?;

        // 构建更新参数 - 请求已使用强类型枚举，直接使用
        let params = UpdateSettingsParams {
            privacy_level: req.privacy_level,
            location_visible: req.location_visible,
            online_visible: req.online_visible,
            last_online_visible: req.last_online_visible,
            friends_visible: req.friends_visible,
            following_visible: req.following_visible,
            followers_visible: req.followers_visible,
            groups_visible: req.groups_visible,
            moments_visible: req.moments_visible,
            liked_moments_visible: req.liked_moments_visible,
            stats_visible: req.stats_visible,
            allow_follow: req.allow_follow,
            allow_private_chat: req.allow_private_chat,
            searchable: req.searchable,
            phone_searchable: req.phone_searchable,
            nickname_searchable: req.nickname_searchable,
            location_searchable: req.location_searchable,
        };

        self.user_settings_repo.update(user_id, &params).await?;

        // 清除设置缓存
        let cache_key = user_id.to_string();
        if let Err(e) = self
            .cache_manager
            .delete(cache_keys::USER_SETTINGS, &cache_key)
            .await
        {
            tracing::warn!(user_id, error = %e, "delete user settings cache failed");
        }

        Ok(())
    }

    /// 升级账户(临时用户→正式用户)
    pub async fn upgrade_account(
        &self,
        user_id: i64,
        req: &UpgradeAccountRequest,
        jwt_secret: &str,
    ) -> Result<AuthData> {
        let access_token_expiration = 3600;
        let refresh_token_expiration = 2592000;
        // 1. 验证登录ID
        Validator::validate_not_empty(&req.login_id, "登录ID")?;

        // 2. 检查登录ID是否已被使用
        if self.user_repo.exists_by_login_id(&req.login_id).await? {
            return Err(AppError::AlreadyExists("该登录ID已被注册".to_string()));
        }

        // 3. 对密码进行服务端慢哈希（bcrypt）
        let password_hash = PasswordUtil::hash_password(&req.password)?;

        // 4. 更新用户信息
        self.user_repo
            .upgrade_account(user_id, &req.login_id, &password_hash)
            .await?;

        // 5. 获取更新后的用户信息
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::UserNotFound("用户不存在".to_string()))?;

        // 6. 生成新的Token
        let session_id = Uuid::new_v4();
        let access_token =
            JwtUtil::generate_token(user_id, session_id, access_token_expiration, jwt_secret)?;
        let refresh_token = TokenUtil::generate_refresh_token();

        // DB 仅存 hash
        let access_token_hash = TokenUtil::hash_token(&access_token, jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token, jwt_secret);

        // 7. 创建会话记录
        let session_expires_at = Utc::now() + Duration::seconds(refresh_token_expiration);
        self.session_repo
            .create(
                user_id,
                session_id,
                None, // device_type
                None, // device_name
                None, // device_fingerprint
                None, // ip_address
                None, // user_agent
                access_token_hash,
                refresh_token_hash,
                session_expires_at,
            )
            .await?;

        let refresh_token_expires_at =
            Some((Utc::now() + Duration::seconds(refresh_token_expiration)).to_rfc3339());

        Ok(AuthData {
            user_id,
            nickname: user.nickname,
            access_token,
            refresh_token: Some(refresh_token),
            access_token_expires_at: (Utc::now() + Duration::seconds(access_token_expiration))
                .to_rfc3339(),
            refresh_token_expires_at,
        })
    }

    /// 删除账户
    pub async fn delete_account(&self, user_id: i64, req: &DeleteAccountRequest) -> Result<()> {
        if req.confirmation != "DELETE_MY_ACCOUNT" {
            return Err(AppError::BadRequest("确认字符串不匹配".to_string()));
        }

        // 软删除用户
        self.user_repo.soft_delete(user_id).await?;

        // 注销所有设备 token
        self.device_token_repo.unregister_all(user_id).await?;

        // 撤销所有会话
        self.session_repo.revoke_all_by_user(user_id).await?;

        Ok(())
    }

    /// 获取指定用户加入的公开群组
    pub async fn get_user_groups_list(
        &self,
        req: &UserGroupsRequest,
    ) -> Result<PaginatedData<GroupItem>> {
        let target_user_id = req.user_id;

        // 验证用户存在
        if !self.user_repo.exists(target_user_id).await? {
            return Err(AppError::UserNotFound("用户不存在".to_string()));
        }

        // 检查 groups_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let groups_visible = settings
            .as_ref()
            .map(|s| s.groups_visible)
            .unwrap_or(PrivacyLevel::Public);
        if groups_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let cursor = Self::parse_cursor(req.cursor.as_deref());
        let limit = Validator::validate_limit(req.limit, 100)? as i32;

        // 单条 JOIN SQL：直接返回 group_info（避免 N+1 与错误的“成员列表limit=1000计数”）
        let (rows, next_cursor) = self
            .group_member_repo
            .list_public_group_infos_by_user(
                target_user_id,
                limit,
                cursor,
                DEFAULT_LATITUDE,
                DEFAULT_LONGITUDE,
            )
            .await?;

        let group_ids: Vec<Uuid> = rows.iter().map(|r| r.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let mut items = Vec::with_capacity(rows.len());
        for r in rows {
            let tags = tags_map.get(&r.group_id).cloned();
            let tags = tags.filter(|t| !t.is_empty());
            items.push(GroupItem {
                group_chat_id: r.group_id.to_string(),
                name: r.name,
                description: r.description,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(r.avatar_url.as_deref())
                    .await,
                tags,
                members_count: r.members_count,
                has_password: r.has_password,
                location: RelativeLocation {
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location_name: r.location_name,
                    distance: r.distance,
                    bearing_degrees: r.bearing_degrees,
                },
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

    /// 获取关注请求列表（待处理）
    pub async fn get_follow_requests(
        &self,
        user_id: i64,
        req: &MyFollowRequestsRequest,
    ) -> Result<PaginatedData<FollowRequestItem>> {
        let limit = Validator::validate_limit(req.limit, 50)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (rows, next_cursor) = self
            .follow_request_repo
            .list_received(user_id, limit, cursor)
            .await?;

        let requester_ids: Vec<i64> = rows.iter().map(|r| r.requester_id).collect();
        let profiles = self.user_repo.find_profiles(&requester_ids).await?;
        let profile_map: HashMap<i64, _> = profiles.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let requester = profile_map.get(&row.requester_id);
            items.push(FollowRequestItem {
                request_id: row.request_id.to_string(),
                requester: UserInfo {
                    user_id: row.requester_id,
                    nickname: requester
                        .map(|u| u.nickname.clone())
                        .unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(requester.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                    user_type: requester
                        .map(|u| u.user_type)
                        .unwrap_or(crate::types::enums::UserType::Normal),
                },
                message: row.message,
                requested_at: row.created_at.to_rfc3339(),
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

    /// 处理关注请求（接受/拒绝）
    pub async fn handle_follow_request(
        &self,
        user_id: i64,
        req: &HandleFollowRequestRequest,
    ) -> Result<()> {
        let request_id = Uuid::parse_str(&req.request_id)
            .map_err(|_| AppError::BadRequest("无效请求ID".to_string()))?;
        let accept = match req.action.as_str() {
            "accept" => true,
            "reject" => false,
            _ => {
                return Err(AppError::BadRequest(
                    "无效操作，必须为 accept 或 reject".to_string(),
                ))
            }
        };

        let row = self
            .follow_request_repo
            .handle(request_id, user_id, accept)
            .await?
            .ok_or_else(|| AppError::NotFound("关注请求不存在".to_string()))?;

        if accept {
            if !self
                .follows_repo
                .exists(row.requester_id, row.target_id)
                .await?
            {
                self.follows_repo
                    .create(row.requester_id, row.target_id)
                    .await?;
            }
            let is_mutual = self
                .follows_repo
                .exists(row.target_id, row.requester_id)
                .await?;
            self.event_bus.publish(AppEvent::NewFollower {
                follower_id: row.requester_id,
                user_id: row.target_id,
                is_mutual,
            });
            if is_mutual {
                self.event_bus.publish(AppEvent::MutualFollow {
                    user_id: row.requester_id,
                    mutual_user_id: row.target_id,
                });
            }
        }

        self.event_bus.publish(AppEvent::FollowRequestHandled {
            request_id: row.request_id.to_string(),
            requester_id: row.requester_id,
            target_id: row.target_id,
            accepted: accept,
        });

        Ok(())
    }

    /// 获取指定用户点赞的瞬间
    pub async fn get_user_liked_moments_list(
        &self,
        req: &UserLikedMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let target_user_id = req.user_id;

        // 验证用户存在并获取参考位置
        let target_user = self
            .user_repo
            .find_by_id(target_user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;
        let reference_lat = target_user.current_latitude.unwrap_or(DEFAULT_LATITUDE);
        let reference_lng = target_user.current_longitude.unwrap_or(DEFAULT_LONGITUDE);

        // 检查 liked_moments_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let liked_visible = settings
            .as_ref()
            .map(|s| s.liked_moments_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 moments_visible = 'public' 的用户
        if liked_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let cursor = Self::parse_cursor(req.cursor.as_deref());
        let limit = Validator::validate_limit(req.limit, 100)? as i32;

        // 获取用户点赞的瞬间
        // 这里的 get_user_liked_moments 之前已经被重构为返回 (Vec<MomentLikeRow>, Option<String>)
        let (moment_likes, next_cursor) = self
            .moment_repo
            .get_user_liked_moments(target_user_id, limit as i64, cursor)
            .await?;

        let moment_uuids: Vec<Uuid> = moment_likes.iter().map(|m| m.moment_id).collect();

        // 批量获取瞬间信息（带距离计算）
        let moments = self
            .moment_repo
            .find_by_ids_with_distance(&moment_uuids, reference_lat, reference_lng, None)
            .await?;

        // 获取用户信息
        let mut user_ids: Vec<i64> = moments.iter().map(|m| m.user_id).collect();
        user_ids.sort_unstable();
        user_ids.dedup();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 获取统计数据
        let stats = self.moment_repo.get_stats_batch(&moment_uuids).await?;

        // 获取标签
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_uuids).await?;

        let mut items = Vec::with_capacity(moments.len());
        for m in moments {
            if let Some(user) = user_map.get(&m.user_id) {
                let (likes_count, comments_count) =
                    stats.get(&m.moment_id).copied().unwrap_or((0, 0));
                let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();

                let mut images = Vec::with_capacity(m.images.len());
                for path in &m.images {
                    let url = match self.media_service.sign_url(path).await {
                        Ok(u) => u,
                        Err(e) => {
                            tracing::warn!("Failed to sign url for path {}: {}", path, e);
                            path.clone()
                        }
                    };
                    images.push(MomentImage {
                        url,
                        width: None,
                        height: None,
                        content_type: None,
                    });
                }

                items.push(MomentItem {
                    moment_id: m.moment_id.to_string(),
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(user.avatar_url.as_deref())
                            .await,
                        user_type: user.user_type,
                    },
                    description: m.description,
                    images: Some(images),
                    tags,
                    mentions: m.mentions.clone(),
                    created_at: m.created_at.to_rfc3339(),
                    likes_count: likes_count as i32,
                    comments_count: comments_count as i32,
                    location: RelativeLocation {
                        latitude: m.latitude,
                        longitude: m.longitude,
                        location_name: m.location_name,
                        distance: m.distance,
                        bearing_degrees: m.bearing_degrees,
                    },
                });
            }
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    // ==================== 管理员用户管理 (Admin) ====================

    pub async fn admin_search_users(
        &self,
        req: &AdminUserSearchRequest,
    ) -> Result<PaginatedData<AdminUserItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let cursor = Self::parse_cursor(req.cursor.as_deref());

        let (users, next_cursor) = self
            .user_repo
            .admin_search(
                req.user_id,
                req.nickname.as_deref(),
                req.login_id.as_deref(),
                req.status.as_ref().map(|s| s.as_str()),
                req.user_type,
                limit,
                cursor,
            )
            .await?;

        let user_ids: Vec<i64> = users.iter().map(|u| u.user_id).collect();
        let online_map = self
            .presence_service
            .batch_get_online_status_raw(&user_ids)
            .await;

        let mut items = Vec::with_capacity(users.len());
        for u in users {
            items.push(AdminUserItem {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await,
                user_type: u.user_type,
                status: UserAccountStatus::from_str(&u.status).unwrap_or(UserAccountStatus::Normal),
                created_at: u.created_at.to_rfc3339(),
                last_active_at: u.last_active_at.map(|t| t.to_rfc3339()),
                login_id: u.login_id.unwrap_or_default(),
                is_online: *online_map.get(&u.user_id).unwrap_or(&false),
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

    pub async fn admin_user_detail(&self, req: &AdminUserDetailRequest) -> Result<AdminUserDetail> {
        let user = self
            .user_repo
            .admin_detail(req.user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let tags = match self.tag_repo.get_user_tags(user.user_id).await {
            Ok(tags) => {
                if tags.is_empty() {
                    None
                } else {
                    Some(tags)
                }
            }
            Err(e) => {
                tracing::warn!(user_id = user.user_id, error = %e, "get_user_tags failed");
                None
            }
        };

        let location = if let (Some(lat), Some(lng), Some(name)) = (
            user.current_latitude,
            user.current_longitude,
            user.current_location_name.clone(),
        ) {
            Some(Location {
                latitude: lat,
                longitude: lng,
                location_name: name,
            })
        } else {
            None
        };

        let stats_data = self.get_stats(user.user_id).await?;
        let stats = UserCardStats {
            moments_count: stats_data.moments_count,
            friends_count: stats_data.friends_count,
            followers_count: stats_data.followers_count,
            following_count: stats_data.following_count,
        };

        Ok(AdminUserDetail {
            user_id: user.user_id,
            nickname: user.nickname,
            avatar_url: self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await,
            user_type: user.user_type,
            status: if user.is_banned {
                UserAccountStatus::Banned
            } else {
                UserAccountStatus::Normal
            },
            created_at: user.created_at.to_rfc3339(),
            last_active_at: user.last_active_at.map(|t| t.to_rfc3339()),
            login_id: user.login_id.unwrap_or_default(),
            email: user.email,
            phone: user.phone,
            bio: if user.bio.is_empty() {
                None
            } else {
                Some(user.bio)
            },
            tags,
            location,
            is_online: user.is_online,
            stats,
            ban_history: None,
        })
    }

    pub async fn admin_ban_user(&self, req: &AdminBanUserRequest) -> Result<()> {
        // 1. 验证用户存在
        if !self.user_repo.exists(req.user_id).await? {
            return Err(AppError::UserNotFound("用户不存在".to_string()));
        }
        // 2. 执行封禁
        self.user_repo
            .ban(req.user_id, req.reason.as_deref(), req.duration_days)
            .await?;
        // 3. 强制登出（撤销所有会话）
        self.session_repo.revoke_all_by_user(req.user_id).await?;
        Ok(())
    }

    pub async fn admin_unban_user(&self, req: &AdminUnbanUserRequest) -> Result<()> {
        // 1. 验证用户存在
        if !self.user_repo.exists(req.user_id).await? {
            return Err(AppError::UserNotFound("用户不存在".to_string()));
        }
        // 2. 执行解封
        self.user_repo
            .unban(req.user_id, req.reason.as_deref())
            .await?;
        Ok(())
    }
}
