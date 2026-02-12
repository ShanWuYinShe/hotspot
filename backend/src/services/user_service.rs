//! 用户服务层实现
//!
//! 核心业务逻辑：资料、关系链、搜索、设置

use std::collections::HashMap;
use std::str::FromStr;
use std::sync::Arc;

use chrono::{Duration, Utc};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use uuid::Uuid;

use crate::{
    constants::REDIS_KEY_SESSION,
    dto::{
        request::{
            AdminBanUserRequest, AdminUnbanUserRequest, AdminUserDetailRequest,
            AdminUserSearchRequest, BlockRequest, DeleteAccountRequest, FollowRequest,
            GetRelationshipRequest, HandleFollowRequestRequest, MyBlockedRequest,
            MyFollowRequestsRequest, SearchUsersByKeywordRequest, SearchUsersByLocationRequest,
            SearchUsersByTagRequest, SetRemarkRequest, UnblockRequest, UnfollowRequest,
            UpdateLocationRequest, UpdateProfileRequest, UpdateSettingsRequest,
            UpgradeAccountRequest, UserFollowersRequest, UserFollowingRequest, UserFriendsRequest,
            UserGroupsRequest, UserLikedMomentsRequest, UserOpenTheDoorRequest,
        },
        response::{
            AdminUserDetail, AdminUserSearchItem, AuthData, BlockedUserSearchItem, FollowRequestItem,
            FollowerItem, FollowingItem, FriendItem, GroupSearchItem, MyProfile, PaginatedData,
            PaginationMeta, RelativeLocation, UserCardStats, UserInfo, UserMomentListItem,
            UserRelation, UserSearchItem, UserSettingsData, UserStatsData,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        follow_request_repo::FollowRequestRepository,
        group_member_repo::GroupMemberRepository,
        moment_repo::{MomentDetailRow, MomentImageRow, MomentRepository},
        session_repo::SessionRepository,
        tag_repo::TagRepository,
        user_blocks_repo::UserBlocksRepository,
        user_follows_repo::UserFollowsRepository,
        user_friend_remarks_repo::UserFriendRemarksRepository,
        user_location_history_repo::UserLocationHistoryRepository,
        user_repo::{UserRepository, UserRow},
        user_settings_repo::{UpdateSettingsParams, UserSettingsRepository, UserSettingsRow},
    },
    types::enums::{PrivacyLevel, PrivateChatPermission, UserStatus, UserType},
    utils::{
        bloom_filter::ResourceExistenceChecker,
        cache::CacheManager,
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
    redis: ConnectionManager,
    access_token_expiration: i64,
    refresh_token_expiration: i64,
    user_repo: UserRepository,
    user_settings_repo: UserSettingsRepository,
    user_location_history_repo: UserLocationHistoryRepository,
    follows_repo: UserFollowsRepository,
    follow_request_repo: FollowRequestRepository,
    blocks_repo: UserBlocksRepository,
    user_friend_remarks_repo: UserFriendRemarksRepository,
    device_token_repo: crate::repositories::device_token_repo::DeviceTokenRepository,
    session_repo: SessionRepository,
    tag_repo: TagRepository,
    group_member_repo: GroupMemberRepository,
    moment_repo: MomentRepository,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    cache_manager: CacheManager,
    _resource_checker: ResourceExistenceChecker,
}

impl UserService {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        event_bus: Arc<EventBus>,
        redis: ConnectionManager,
        access_token_expiration: i64,
        refresh_token_expiration: i64,
        user_repo: UserRepository,
        user_settings_repo: UserSettingsRepository,
        user_location_history_repo: UserLocationHistoryRepository,
        follows_repo: UserFollowsRepository,
        follow_request_repo: FollowRequestRepository,
        blocks_repo: UserBlocksRepository,
        user_friend_remarks_repo: UserFriendRemarksRepository,
        device_token_repo: crate::repositories::device_token_repo::DeviceTokenRepository,
        session_repo: SessionRepository,
        tag_repo: TagRepository,
        group_member_repo: GroupMemberRepository,
        moment_repo: MomentRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
        resource_checker: ResourceExistenceChecker,
    ) -> Self {
        Self {
            event_bus,
            redis,
            access_token_expiration,
            refresh_token_expiration,
            user_repo,
            user_settings_repo,
            user_location_history_repo,
            follows_repo,
            follow_request_repo,
            blocks_repo,
            user_friend_remarks_repo,
            device_token_repo,
            session_repo,
            tag_repo,
            group_member_repo,
            moment_repo,
            presence_service,
            media_service,
            cache_manager,
            _resource_checker: resource_checker,
        }
    }

    pub async fn search_by_keyword(
        &self,
        req: &SearchUsersByKeywordRequest,
    ) -> Result<PaginatedData<UserSearchItem>> {
        Validator::validate_not_empty(&req.kw, "kw")?;
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .user_repo
            .search_by_keyword(
                &req.kw,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_user_search_list(rows, next_cursor).await
    }

    pub async fn search_by_tag(&self, req: &SearchUsersByTagRequest) -> Result<PaginatedData<UserSearchItem>> {
        Validator::validate_not_empty(&req.tag, "tag")?;
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .user_repo
            .search_by_tag(
                &req.tag,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_user_search_list(rows, next_cursor).await
    }

    pub async fn search_by_location(
        &self,
        req: &SearchUsersByLocationRequest,
    ) -> Result<PaginatedData<UserSearchItem>> {
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let radius = Validator::validate_radius_required(req.radius, 50000.0)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .user_repo
            .search_nearby(
                req.reference_point.longitude,
                req.reference_point.latitude,
                radius,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_user_search_list(rows, next_cursor).await
    }

    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &UserOpenTheDoorRequest,
    ) -> Result<UserSearchItem> {
        let user = self
            .user_repo
            .find_by_id(req.user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        if let Some(viewer_id) = viewer_id {
            if self.blocks_repo.is_blocked(req.user_id, viewer_id).await? {
                return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
            }
        }

        let tags = self.tag_repo.get_user_tags(user.user_id).await.unwrap_or_default();
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
            Some(self.get_relationship(viewer_id, &GetRelationshipRequest { user_id: user.user_id }).await?)
        } else {
            None
        };

        Ok(UserSearchItem {
            user_id: user.user_id,
            nickname: user.nickname.unwrap_or_default(),
            avatar_url: self.media_service.sign_url_if_path(user.avatar_url.as_deref()).await,
            user_type: user.user_type,
            bio: user.bio,
            tags: Some(tags),
            is_online,
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
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let tags = self.tag_repo.get_user_tags(user_id).await.unwrap_or_default();
        let stats = self.get_stats(user_id).await?;

        let (is_online, last_online) = self.presence_service.get_online_status(user_id).await;
        let last_online = last_online
            .or_else(|| user.last_login_at.map(|t| t.to_rfc3339()))
            .unwrap_or_else(|| user.created_at.to_rfc3339());

        let location = self
            .build_location_for_user(user_id)
            .await
            .unwrap_or(RelativeLocation {
                point: crate::dto::response::GeoPoint { latitude: 0.0, longitude: 0.0 },
                location_name: "".to_string(),
                distance: None,
                bearing_degrees: None,
            });

        Ok(MyProfile {
            user_id: user.user_id,
            nickname: user.nickname.unwrap_or_default(),
            avatar_url: self.media_service.sign_url_if_path(user.avatar_url.as_deref()).await,
            bio: user.bio,
            tags: Some(tags),
            gender: Some(user.gender),
            email: user.email,
            phone: user.phone,
            location,
            is_online,
            last_online,
            created_at: user.created_at.to_rfc3339(),
            stats,
        })
    }

    pub async fn update_profile(&self, user_id: i64, req: &UpdateProfileRequest) -> Result<()> {
        if let Some(nickname) = &req.nickname {
            Validator::validate_length(nickname, "昵称", 1, 50)?;
        }

        let avatar_url = if let Some(fid) = req.avatar_file_id.as_deref() {
            let fid_uuid = Uuid::parse_str(fid)
                .map_err(|_| AppError::InvalidFormat("无效文件ID".to_string()))?;
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
                None,
                None,
            )
            .await?;

        if let Some(tags) = &req.tags {
            self.tag_repo.set_user_tags(user_id, tags).await?;
        }

        let _ = self
            .cache_manager
            .delete(crate::utils::cache::cache_keys::USER_INFO, &user_id.to_string())
            .await;

        self.event_bus.publish(crate::events::event_types::AppEvent::UserProfileUpdated {
            user_id,
            nickname: req.nickname.clone(),
            avatar_url: avatar_url.map(|s| s.to_string()),
            bio: req.bio.clone(),
        });

        Ok(())
    }

    pub async fn get_relationship(
        &self,
        user_id: i64,
        req: &GetRelationshipRequest,
    ) -> Result<UserRelation> {
        let is_following = self.follows_repo.is_following(user_id, req.user_id).await?;
        let is_followed_by = self.follows_repo.is_following(req.user_id, user_id).await?;
        let is_blocked = self.blocks_repo.is_blocked(user_id, req.user_id).await?;
        let has_pending_follow_request =
            self.follow_request_repo.exists(user_id, req.user_id).await?;

        Ok(UserRelation {
            is_mutual_follow: is_following && is_followed_by,
            is_following,
            is_followed_by,
            is_blocked,
            has_pending_follow_request,
        })
    }

    pub async fn get_stats(&self, user_id: i64) -> Result<UserStatsData> {
        let stats = self.user_repo.get_stats(user_id).await?;
        Ok(UserStatsData {
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

        self.user_settings_repo.update(user_id, &params).await
    }

    pub async fn get_friends_list(
        &self,
        user_id: i64,
        req: &UserFriendsRequest,
    ) -> Result<PaginatedData<FriendItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_friends(user_id, limit, req.cursor.clone())
            .await?;

        let friend_ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&friend_ids).await?;
        let profile_map: HashMap<i64, UserRow> = profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let remark_map = self
            .user_friend_remarks_repo
            .get_batch(user_id, &friend_ids)
            .await
            .unwrap_or_default();
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&friend_ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let (is_online, _last_online) = online_map.get(&row.user_id).cloned().unwrap_or((None, None));
                items.push(FriendItem {
                    user: self.build_user_info(profile).await,
                    is_online,
                    remark: remark_map.get(&row.user_id).cloned(),
                    followed_at: row.followed_at.to_rfc3339(),
                    followed_back_at: row.followed_back_at.to_rfc3339(),
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

    pub async fn get_following_list(
        &self,
        user_id: i64,
        req: &UserFollowingRequest,
    ) -> Result<PaginatedData<FollowingItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_following(user_id, limit, req.cursor.clone())
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> = profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let is_mutual = self.follows_repo.is_following(row.user_id, user_id).await.unwrap_or(false);
                let (is_online, _last_online) = online_map.get(&row.user_id).cloned().unwrap_or((None, None));
                items.push(FollowingItem {
                    user: self.build_user_info(profile).await,
                    followed_at: row.created_at.to_rfc3339(),
                    is_mutual,
                    is_online,
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

    pub async fn get_followers_list(
        &self,
        user_id: i64,
        req: &UserFollowersRequest,
    ) -> Result<PaginatedData<FollowerItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .follows_repo
            .list_followers(user_id, limit, req.cursor.clone())
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> = profiles.into_iter().map(|p| (p.user_id, p)).collect();
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.user_id) {
                let is_following_back = self.follows_repo.is_following(user_id, row.user_id).await.unwrap_or(false);
                let (is_online, _last_online) = online_map.get(&row.user_id).cloned().unwrap_or((None, None));
                items.push(FollowerItem {
                    user: self.build_user_info(profile).await,
                    followed_at: row.created_at.to_rfc3339(),
                    is_following_back,
                    is_online,
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

    pub async fn get_user_groups_list(
        &self,
        req: &UserGroupsRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .group_member_repo
            .list_user_groups(req.user_id, limit as usize, req.cursor.as_deref())
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            items.push(GroupSearchItem {
                group_chat_id: row.group_id.to_string(),
                name: row.name,
                description: row.description,
                avatar_url: self.media_service.sign_url_if_path(row.avatar_url.as_deref()).await,
                tags: None,
                members_count: row.members_count as i32,
                has_password: row.has_password,
                location: RelativeLocation {
                    point: crate::dto::response::GeoPoint { latitude: 0.0, longitude: 0.0 },
                    location_name: "".to_string(),
                    distance: None,
                    bearing_degrees: None,
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

    pub async fn get_user_liked_moments_list(
        &self,
        req: &UserLikedMomentsRequest,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_user_liked_moments(req.user_id, limit, req.cursor.clone())
            .await?;

        self.build_user_moment_items(rows, next_cursor).await
    }

    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        Validator::validate_coordinates(req.location.point.latitude, req.location.point.longitude)?;
        self.user_repo
            .update_location(user_id, req.location.point.longitude, req.location.point.latitude)
            .await?;
        let _ = self
            .user_location_history_repo
            .record(user_id, req.location.point.longitude, req.location.point.latitude)
            .await;
        let _ = self
            .presence_service
            .update_location_cache(user_id, req.location.point.latitude, req.location.point.longitude)
            .await;
        Ok(())
    }

    pub async fn upgrade_account(
        &self,
        user_id: i64,
        req: &UpgradeAccountRequest,
        jwt_secret: &str,
    ) -> Result<AuthData> {
        Validator::validate_not_empty(&req.login_id, "login_id")?;
        Validator::validate_length(&req.password, "password", 6, 64)?;

        if let Some(existing) = self.user_repo.find_by_identifier(&req.login_id).await? {
            if existing.user_id != user_id {
                return Err(AppError::AlreadyExists("该用户已存在".to_string()));
            }
        }

        let password_hash = PasswordUtil::hash_password(&req.password)?;
        self.user_repo.update_password(user_id, &password_hash).await?;
        self.user_repo.update_login_id(user_id, &req.login_id).await?;
        self.user_repo.update_user_type(user_id, UserType::Normal).await?;

        let session_id = Uuid::now_v7();
        let access_token = JwtUtil::generate_token(
            user_id,
            session_id,
            self.access_token_expiration,
            jwt_secret,
        )?;
        let refresh_token_raw = TokenUtil::generate_refresh_token();
        let access_token_hash = TokenUtil::hash_token(&access_token, jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token_raw, jwt_secret);
        let refresh_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);

        self.session_repo
            .create(
                user_id,
                session_id,
                None,
                None,
                None,
                None,
                None,
                access_token_hash,
                refresh_token_hash,
                refresh_expires_at,
            )
            .await?;

        let key = format!("{}:{}", REDIS_KEY_SESSION, session_id);
        let mut conn = self.redis.clone();
        let _: () = conn
            .set_ex(key, user_id.to_string(), self.refresh_token_expiration as u64)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        Ok(AuthData {
            user_id,
            nickname: req.login_id.clone(),
            access_token,
            refresh_token: Some(refresh_token_raw.clone()),
            access_token_expires_at:
                (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
            refresh_token_expires_at: Some(refresh_expires_at.to_rfc3339()),
        })
    }

    pub async fn delete_account(&self, user_id: i64, req: &DeleteAccountRequest) -> Result<()> {
        if req.confirmation != "DELETE_MY_ACCOUNT" {
            return Err(AppError::Conflict("确认字符串不正确".to_string()));
        }
        self.user_repo.update_status(user_id, UserStatus::Deleted).await?;
        let _ = self.session_repo.revoke_all_by_user(user_id).await;
        Ok(())
    }

    pub async fn follow_user(&self, user_id: i64, req: &FollowRequest) -> Result<()> {
        if user_id == req.user_id {
            return Err(AppError::CannotOperateOnSelf);
        }

        if !self.user_repo.exists(req.user_id).await? {
            return Err(AppError::UserNotFound("用户不存在".to_string()));
        }

        if self.blocks_repo.is_blocked(req.user_id, user_id).await? {
            return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
        }

        let allow_follow = self.bool_from_settings(req.user_id, "allow_follow", true).await;
        if !allow_follow {
            return Err(AppError::FollowNotAllowed("对方不允许被关注".to_string()));
        }

        self.follows_repo.follow(user_id, req.user_id).await?;
        let is_mutual = self.follows_repo.is_following(req.user_id, user_id).await?;

        self.event_bus.publish(AppEvent::NewFollower {
            follower_id: user_id,
            user_id: req.user_id,
            is_mutual,
        });

        Ok(())
    }

    pub async fn unfollow_user(&self, user_id: i64, req: &UnfollowRequest) -> Result<()> {
        if user_id == req.user_id {
            return Err(AppError::CannotOperateOnSelf);
        }

        self.follows_repo.unfollow(user_id, req.user_id).await?;
        Ok(())
    }

    pub async fn get_follow_requests(
        &self,
        user_id: i64,
        req: &MyFollowRequestsRequest,
    ) -> Result<PaginatedData<FollowRequestItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as usize;
        let (rows, next_cursor) = self
            .follow_request_repo
            .list_received(user_id, None, limit, req.cursor.clone())
            .await?;

        let requester_ids: Vec<i64> = rows.iter().map(|r| r.requester_id).collect();
        let profiles = self.user_repo.find_profiles(&requester_ids).await?;
        let profile_map: HashMap<i64, UserRow> = profiles.into_iter().map(|p| (p.user_id, p)).collect();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            if let Some(profile) = profile_map.get(&row.requester_id) {
                items.push(FollowRequestItem {
                    request_id: row.request_id.to_string(),
                    requester: self.build_user_info(profile).await,
                    message: None,
                    requested_at: row.created_at.to_rfc3339(),
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

    pub async fn handle_follow_request(
        &self,
        user_id: i64,
        req: &HandleFollowRequestRequest,
    ) -> Result<()> {
        let request_id = Uuid::parse_str(&req.request_id)
            .map_err(|_| AppError::InvalidFormat("无效的 request_id".to_string()))?;
        let record = self
            .follow_request_repo
            .find_by_id(request_id)
            .await?
            .ok_or_else(|| AppError::NotFound("关注请求不存在".to_string()))?;

        if record.target_id != user_id {
            return Err(AppError::Forbidden("无权处理该请求".to_string()));
        }

        match req.action {
            crate::types::enums::FollowRequestAction::Accept => {
                self.follow_request_repo.accept(request_id).await?;
                self.follows_repo.follow(record.requester_id, record.target_id).await?;
            }
            crate::types::enums::FollowRequestAction::Reject => {
                self.follow_request_repo.reject(request_id).await?;
            }
        }

        Ok(())
    }

    pub async fn block_user(&self, user_id: i64, req: &BlockRequest) -> Result<()> {
        if user_id == req.user_id {
            return Err(AppError::CannotOperateOnSelf);
        }
        if !self.user_repo.exists(req.user_id).await? {
            return Err(AppError::UserNotFound("用户不存在".to_string()));
        }
        self.blocks_repo.block(user_id, req.user_id).await?;
        Ok(())
    }

    pub async fn unblock_user(&self, user_id: i64, req: &UnblockRequest) -> Result<()> {
        if user_id == req.user_id {
            return Err(AppError::CannotOperateOnSelf);
        }
        self.blocks_repo.unblock(user_id, req.user_id).await?;
        Ok(())
    }

    pub async fn get_blocked_users_list(
        &self,
        user_id: i64,
        req: &MyBlockedRequest,
    ) -> Result<PaginatedData<BlockedUserSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .blocks_repo
            .list_blocked(user_id, limit, req.cursor.clone())
            .await?;

        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let profiles = self.user_repo.find_profiles(&ids).await?;
        let profile_map: HashMap<i64, UserRow> = profiles.into_iter().map(|p| (p.user_id, p)).collect();

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

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn set_friend_remark(&self, user_id: i64, req: &SetRemarkRequest) -> Result<()> {
        if user_id == req.user_id {
            return Err(AppError::CannotOperateOnSelf);
        }

        if !self.follows_repo.is_following(user_id, req.user_id).await? {
            return Err(AppError::NotFriends);
        }

        if let Some(remark) = &req.remark {
            if remark.trim().is_empty() {
                self.user_friend_remarks_repo.delete(user_id, req.user_id).await?;
            } else {
                self.user_friend_remarks_repo
                    .upsert(user_id, req.user_id, remark.trim().to_string())
                    .await?;
            }
        } else {
            self.user_friend_remarks_repo.delete(user_id, req.user_id).await?;
        }

        Ok(())
    }

    pub async fn admin_search_users(
        &self,
        req: &AdminUserSearchRequest,
    ) -> Result<PaginatedData<AdminUserSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .user_repo
            .admin_search(
                req.user_id,
                req.nickname.as_deref(),
                req.login_id.as_deref(),
                req.status,
                req.user_type,
                limit,
                req.cursor.clone(),
            )
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            items.push(AdminUserSearchItem {
                user_id: row.user_id,
                nickname: row.nickname.unwrap_or_default(),
                avatar_url: self.media_service.sign_url_if_path(row.avatar_url.as_deref()).await,
                user_type: row.user_type,
                status: row.status,
                created_at: row.created_at.to_rfc3339(),
                last_active_at: row.last_active_at.map(|t| t.to_rfc3339()),
                login_id: row.login_id,
                is_online: row.is_online,
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

    pub async fn admin_ban_user(&self, req: &AdminBanUserRequest) -> Result<()> {
        self.user_repo.update_status(req.user_id, UserStatus::Banned).await?;

        self.event_bus.publish(crate::events::event_types::AppEvent::AccountBanned {
            user_id: req.user_id,
            reason: req.reason.clone().unwrap_or_else(|| "违规行为".to_string()),
            duration_days: req.duration_days.unwrap_or(0),
        });

        Ok(())
    }

    pub async fn admin_unban_user(&self, req: &AdminUnbanUserRequest) -> Result<()> {
        self.user_repo.update_status(req.user_id, UserStatus::Active).await
    }

    pub async fn admin_user_detail(&self, req: &AdminUserDetailRequest) -> Result<AdminUserDetail> {
        let user = self
            .user_repo
            .find_by_id(req.user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let tags = self.tag_repo.get_user_tags(req.user_id).await.unwrap_or_default();
        let stats = self.user_repo.get_stats(req.user_id).await?;
        let (is_online, last_online) = self.presence_service.get_online_status(req.user_id).await;
        let location = self.build_location_for_user(req.user_id).await;

        Ok(AdminUserDetail {
            user_id: user.user_id,
            nickname: user.nickname.unwrap_or_default(),
            avatar_url: self.media_service.sign_url_if_path(user.avatar_url.as_deref()).await,
            user_type: user.user_type,
            status: user.status,
            created_at: user.created_at.to_rfc3339(),
            last_active_at: last_online,
            login_id: user.username,
            email: user.email,
            phone: user.phone,
            bio: user.bio,
            tags: Some(tags),
            location,
            is_online,
            stats: UserCardStats {
                moments_count: stats.moments_count,
                friends_count: stats.following_count,
                followers_count: stats.followers_count,
                following_count: stats.following_count,
            },
            ban_history: None,
        })
    }

    async fn build_user_search_list(
        &self,
        rows: Vec<crate::repositories::user_repo::UserSearchRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserSearchItem>> {
        let ids: Vec<i64> = rows.iter().map(|r| r.user_id).collect();
        let location_visible_map = self.presence_service.batch_get_location_visible(&ids).await;
        let online_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&ids)
            .await;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let (is_online, last_online) = online_map.get(&row.user_id).cloned().unwrap_or((None, None));
            let location = if location_visible_map.get(&row.user_id).copied().unwrap_or(true) {
                Some(RelativeLocation {
                    point: crate::dto::response::GeoPoint { latitude: 0.0, longitude: 0.0 },
                    location_name: "".to_string(),
                    distance: Some(row.distance),
                    bearing_degrees: Some(row.bearing_degrees),
                })
            } else {
                None
            };

            items.push(UserSearchItem {
                user_id: row.user_id,
                nickname: row.nickname.unwrap_or_default(),
                avatar_url: self.media_service.sign_url_if_path(row.avatar_url.as_deref()).await,
                user_type: UserType::Normal,
                bio: row.bio,
                tags: None,
                is_online,
                last_online,
                location,
                stats: None,
                relation: None,
                social_context: None,
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

    async fn build_user_info(&self, user: &UserRow) -> UserInfo {
        UserInfo {
            user_id: user.user_id,
            nickname: user.nickname.clone().unwrap_or_default(),
            avatar_url: self.media_service.sign_url_if_path(user.avatar_url.as_deref()).await,
            user_type: user.user_type,
        }
    }

    async fn build_user_card_stats(&self, user_id: i64) -> Result<UserCardStats> {
        let stats = self.user_repo.get_stats(user_id).await?;
        Ok(UserCardStats {
            moments_count: stats.moments_count,
            friends_count: stats.following_count,
            followers_count: stats.followers_count,
            following_count: stats.following_count,
        })
    }

    async fn build_location_for_user(&self, user_id: i64) -> Option<RelativeLocation> {
        if let Ok(Some((lat, lng))) = self.user_repo.get_current_location(user_id).await {
            Some(RelativeLocation {
                point: crate::dto::response::GeoPoint { latitude: lat, longitude: lng },
                location_name: "".to_string(),
                distance: None,
                bearing_degrees: None,
            })
        } else {
            None
        }
    }

    fn map_settings(&self, settings: Option<&UserSettingsRow>) -> UserSettingsData {
        UserSettingsData {
            privacy_level: self.privacy_level(settings, "privacy_level", PrivacyLevel::Public),
            location_visible: self.privacy_bool(settings, "location_visible", true),
            online_visible: self.privacy_bool(settings, "online_visible", true),
            last_online_visible: self.privacy_bool(settings, "last_online_visible", true),
            friends_visible: self.privacy_level(settings, "friends_visible", PrivacyLevel::Public),
            following_visible: self.privacy_level(settings, "following_visible", PrivacyLevel::Public),
            followers_visible: self.privacy_level(settings, "followers_visible", PrivacyLevel::Public),
            groups_visible: self.privacy_level(settings, "groups_visible", PrivacyLevel::Public),
            moments_visible: self.privacy_level(settings, "moments_visible", PrivacyLevel::Public),
            liked_moments_visible: self.privacy_level(settings, "liked_moments_visible", PrivacyLevel::Public),
            stats_visible: self.privacy_level(settings, "stats_visible", PrivacyLevel::Public),
            allow_follow: self.privacy_bool(settings, "allow_follow", true),
            allow_private_chat: self.privacy_chat_permission(settings, "allow_private_chat", PrivateChatPermission::Everyone),
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
        let settings = self.user_settings_repo.find_by_user_id(user_id).await.ok().flatten();
        self.privacy_bool(settings.as_ref(), key, default)
    }

    async fn is_visibility_allowed(&self, user_id: i64, key: &str) -> Result<bool> {
        let settings = self.user_settings_repo.find_by_user_id(user_id).await?;
        let level = self.privacy_level(settings.as_ref(), key, PrivacyLevel::Public);
        Ok(matches!(level, PrivacyLevel::Public))
    }

    async fn location_visible_for_user(&self, user_id: i64) -> bool {
        let settings = self.user_settings_repo.find_by_user_id(user_id).await.ok().flatten();
        self.privacy_bool(settings.as_ref(), "location_visible", true)
    }

    async fn build_user_moment_items(
        &self,
        rows: Vec<MomentDetailRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserMomentListItem>> {
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

            items.push(UserMomentListItem {
                moment_id: row.moment_id.to_string(),
                description: row.description.unwrap_or_default(),
                images: image_list,
                tags,
                mentions: row.mentions,
                created_at: row.created_at.to_rfc3339(),
                likes_count: row.like_count,
                comments_count: row.comments_count,
                location: RelativeLocation {
                    point: crate::dto::response::GeoPoint {
                        latitude: row.latitude.unwrap_or(0.0),
                        longitude: row.longitude.unwrap_or(0.0),
                    },
                    location_name: row.location_name.unwrap_or_default(),
                    distance: None,
                    bearing_degrees: None,
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

    async fn sign_images(&self, rows: &[MomentImageRow]) -> Option<Vec<crate::dto::response::MomentImage>> {
        if rows.is_empty() {
            return None;
        }

        let mut images = Vec::with_capacity(rows.len());
        for row in rows {
            let signed = self.media_service.sign_url_if_path(Some(&row.url)).await;
            images.push(crate::dto::response::MomentImage {
                url: signed.unwrap_or_else(|| row.url.clone()),
                width: row.width,
                height: row.height,
                content_type: None,
            });
        }
        Some(images)
    }
}
