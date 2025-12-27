use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::collections::HashMap;
use std::sync::Arc;
use uuid::Uuid;

use chrono::{Duration, Utc};

use crate::{
    constants::{DEFAULT_LOCATION_NAME, LOCATION_CHANGE_THRESHOLD},
    dto::{
        request::{
            Bounds,
            BlockRequest, DeleteAccountRequest, FollowRequest, MyBlockedRequest,
            MyFollowersRequest, MyFollowingRequest, MyFriendsRequest, SearchUsersByKeywordRequest,
            SearchUsersByLocationRequest, SearchUsersByTagRequest, SetRemarkRequest,
            UnblockRequest, UnfollowRequest, UpdateLocationRequest, UpdateProfileRequest,
            UpdateSettingsRequest, UpgradeAccountRequest, UserFollowersRequest,
            UserFollowingRequest, UserFriendsRequest, UserGroupsRequest, UserLikedMomentsRequest,
            UserOpenTheDoorRequest, UserRelationshipRequest, UserStatsRequest,
        },
        response::{
            AuthData, BlockedUserItem, FollowerItem, FollowingItem, FriendListItem, GeoData,
            GroupInfo, MomentSearchItem, MyProfileData, PaginatedData, PaginationMeta, UserInfo,
            UserRelationship, UserSearchItem, UserSettingsData, UserStatsData,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        device_token_repo::DeviceTokenRepository, file_repo::FileRepository,
        session_repo::SessionRepository, tag_repo::TagRepository,
        user_blocks_repo::UserBlocksRepository, user_follows_repo::UserFollowsRepository,
        user_friend_remarks_repo::UserFriendRemarksRepository,
        user_location_history_repo::UserLocationHistoryRepository,
        user_online_status_repo::UserOnlineStatusRepository, user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    utils::{jwt::JwtUtil, password::PasswordUtil, token::TokenUtil, validation::Validator},
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
    remarks_repo: UserFriendRemarksRepository,
    online_status_repo: UserOnlineStatusRepository,
    device_token_repo: DeviceTokenRepository,
    session_repo: SessionRepository,
    tag_repo: TagRepository,
    file_repo: FileRepository,
    redis: ConnectionManager,
}

impl UserService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self {
            event_bus,
            user_repo: UserRepository::new(pool.clone()),
            user_settings_repo: UserSettingsRepository::new(pool.clone()),
            user_location_history_repo: UserLocationHistoryRepository::new(pool.clone()),
            follows_repo: UserFollowsRepository::new(pool.clone()),
            blocks_repo: UserBlocksRepository::new(pool.clone()),
            remarks_repo: UserFriendRemarksRepository::new(pool.clone()),
            online_status_repo: UserOnlineStatusRepository::new(pool.clone()),
            device_token_repo: DeviceTokenRepository::new(pool.clone()),
            session_repo: SessionRepository::new(pool.clone()),
            tag_repo: TagRepository::new(pool.clone()),
            file_repo: FileRepository::new(pool.clone()),
            redis: redis.clone(),
        }
    }

    /// 透传 cursor 字符串
    fn parse_cursor(cursor: Option<&str>) -> Result<Option<String>> {
        Ok(cursor.map(|s| s.to_string()))
    }

    /// 按关键字搜索用户（全局，距离由 PostGIS 计算）
    pub async fn search_by_keyword(
        &self,
        req: &SearchUsersByKeywordRequest,
    ) -> Result<PaginatedData<UserSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let (users, next_cursor) = self
            .user_repo
            .search_by_keyword(&req.kw, req.latitude, req.longitude, limit as i64, cursor)
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 按标签搜索用户（全局，距离由 PostGIS 计算）
    pub async fn search_by_tag(
        &self,
        req: &SearchUsersByTagRequest,
    ) -> Result<PaginatedData<UserSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        // 从标签库获取用户ID列表
        let (user_ids, next_cursor) = self
            .tag_repo
            .search_users_by_tag(&req.tag, limit as i64, cursor)
            .await?;

        // 根据ID获取用户信息并计算距离
        let users = self
            .user_repo
            .find_by_ids_with_distance(&user_ids, req.latitude, req.longitude)
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 按位置搜索用户（范围过滤，距离由 PostGIS 计算）
    pub async fn search_by_location(
        &self,
        req: &SearchUsersByLocationRequest,
    ) -> Result<PaginatedData<UserSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let (users, next_cursor) = self
            .user_repo
            .search_by_location(
                req.kw.as_deref(),
                req.latitude,
                req.longitude,
                req.radius,
                limit as i64,
                cursor,
            )
            .await?;

        self.build_user_search_results(users, next_cursor).await
    }

    /// 按视口 bounds 获取用户（用于地图：替代 search_by_location）
    pub async fn search_by_bounds(
        &self,
        bounds: Bounds,
        latitude: f64,
        longitude: f64,
        limit: Option<i32>,
    ) -> Result<Vec<UserSearchItem>> {
        Validator::validate_coordinates(latitude, longitude)?;
        Validator::validate_coordinates(bounds.north, bounds.east)?;
        Validator::validate_coordinates(bounds.south, bounds.west)?;

        let limit = Validator::validate_limit(limit, 200)?;

        let users = self
            .user_repo
            .search_by_bounds(
                bounds.west,
                bounds.south,
                bounds.east,
                bounds.north,
                latitude,
                longitude,
                limit as i64,
            )
            .await?;

        let data = self.build_user_search_results(users, None).await?;
        Ok(data.items)
    }

    /// 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态，受隐私设置控制）
    pub async fn open_the_door(&self, req: &UserOpenTheDoorRequest) -> Result<UserSearchItem> {
        let user_id = req.id;

        let lat = req.latitude;
        let lng = req.longitude;
        Validator::validate_coordinates(lat, lng)?;

        // 使用 find_by_ids_with_distance 获取用户信息和距离
        let users = self
            .user_repo
            .find_by_ids_with_distance(&[user_id], lat, lng)
            .await?;

        let user = users
            .first()
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 获取用户标签：find_by_ids_with_distance 不包含 tags，因此这里单独查询
        let tag_repo = TagRepository::new(self.user_repo.pool().clone());
        let tags = match tag_repo.get_user_tags(user_id).await {
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
        let (is_online, last_online) = self.get_online_status_with_privacy(user_id).await;

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

        let geo = if location_visible {
            Some(GeoData {
                latitude: user.latitude,
                longitude: user.longitude,
                location_name: user.location_name.clone(),
                distance: user.distance as i32,
                degrees: user.degrees,
            })
        } else {
            None
        };

        Ok(UserSearchItem {
            user_id: user.user_id,
            nickname: user.nickname.clone(),
            avatar_url: user.avatar_url.clone(),
            bio: user.bio.clone(),
            tags,
            is_online,
            last_online,
            user_type: user.user_type,
            geo,
        })
    }

    /// Helper: 构建用户搜索结果（距离和方位角由 PostGIS 计算，包含在线状态，受隐私设置控制）
    async fn build_user_search_results(
        &self,
        users: Vec<crate::repositories::user_repo::UserSearchRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserSearchItem>> {
        // 批量获取标签
        let user_ids: Vec<i64> = users.iter().map(|u| u.user_id).collect();
        let tags_map = self.tag_repo.get_user_tags_batch(&user_ids).await?;

        // 批量获取在线状态（受隐私设置控制）
        let online_status_map = self.batch_get_online_status_with_privacy(&user_ids).await;

        // 批量获取位置可见性设置
        let location_visible_map = self.batch_get_location_visible(&user_ids).await;

        let items: Vec<UserSearchItem> = users
            .iter()
            .map(|user| {
                let (is_online, last_online) = online_status_map
                    .get(&user.user_id)
                    .cloned()
                    .unwrap_or((None, None));

                // 检查位置是否可见
                let location_visible = location_visible_map
                    .get(&user.user_id)
                    .copied()
                    .unwrap_or(true);

                let geo = if location_visible {
                    Some(GeoData {
                        latitude: user.latitude,
                        longitude: user.longitude,
                        location_name: user.location_name.clone(),
                        distance: user.distance as i32,
                        degrees: user.degrees,
                    })
                } else {
                    None
                };

                UserSearchItem {
                    user_id: user.user_id,
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                    bio: user.bio.clone(),
                    tags: tags_map.get(&user.user_id).cloned(),
                    is_online,
                    last_online,
                    user_type: user.user_type,
                    geo,
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

    /// 获取与指定用户的关系
    pub async fn get_relationship(
        &self,
        current_user_id: i64,
        req: &UserRelationshipRequest,
    ) -> Result<UserRelationship> {
        let target_user_id = req.id;

        let is_following = self
            .follows_repo
            .exists(current_user_id, target_user_id)
            .await?;
        let is_followed_by = self
            .follows_repo
            .exists(target_user_id, current_user_id)
            .await?;
        let is_blocking = self
            .blocks_repo
            .is_blocked(current_user_id, target_user_id)
            .await?;
        let is_blocked = self
            .blocks_repo
            .is_blocked(target_user_id, current_user_id)
            .await?;

        Ok(UserRelationship {
            is_mutual_follow: is_following && is_followed_by,
            is_following,
            is_followed_by,
            is_blocked,
            is_blocking,
        })
    }

    /// 更新用户位置
    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        Validator::validate_coordinates(req.latitude, req.longitude)?;

        // 使用 PostGIS 计算与最新位置的距离
        let distance = self
            .user_location_history_repo
            .get_distance_from_latest(user_id, req.latitude, req.longitude)
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
                    req.latitude,
                    req.longitude,
                    req.location_name.clone(),
                )
                .await?;
        }

        // 同时更新 users 表的当前位置
        self.user_repo
            .update_location(
                user_id,
                req.latitude,
                req.longitude,
                req.location_name.as_deref(),
            )
            .await?;

        // 发布位置更新事件
        let location_name = req
            .location_name
            .clone()
            .unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());
        self.event_bus.publish(AppEvent::UserLocationUpdated {
            user_id,
            latitude: req.latitude,
            longitude: req.longitude,
            location_name,
        });

        Ok(())
    }

    /// 更新用户资料
    pub async fn update_profile(&self, user_id: i64, req: &UpdateProfileRequest) -> Result<()> {
        if let Some(nickname) = &req.nickname {
            Validator::validate_not_empty(nickname, "昵称")?;
            Validator::validate_length(nickname, "昵称", 1, 100)?;
        }

        // 解析 avatar_file_id 为 URL
        let avatar_url = match &req.avatar_file_id {
            Some(file_id) => {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的文件ID格式".to_string()))?;
                let url = self
                    .file_repo
                    .validate_file(uuid, user_id, "avatar")
                    .await?;
                self.file_repo.mark_as_used(uuid).await?;
                Some(url)
            }
            None => None,
        };

        self.user_repo
            .update_profile(
                user_id,
                req.nickname.as_deref(),
                avatar_url.as_deref(),
                req.bio.as_deref(),
            )
            .await?;

        // 更新用户标签
        if let Some(tags) = &req.tags {
            self.tag_repo.set_user_tags(user_id, tags).await?;
        }

        Ok(())
    }

    // 好友 = 互相关注，无独立好友申请流程

    /// 关注用户
    pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
        let to_user_id = req.id;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能关注自己".to_string()));
        }

        // 检查目标用户的 allow_follow 设置
        let target_settings = self.user_settings_repo.find_by_user_id(to_user_id).await?;
        if let Some(settings) = target_settings {
            if !settings.allow_follow {
                return Err(AppError::FollowNotAllowed("对方不允许被关注".to_string()));
            }
        }

        // 强约束：已关注则返回冲突错误（避免“重复操作成功”掩盖状态问题）
        let inserted = self.follows_repo.create(from_user_id, to_user_id).await?;
        if !inserted {
            return Err(AppError::FollowRequestAlreadyExists(
                "已经关注过该用户".to_string(),
            ));
        }

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
        let to_user_id = req.id;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能取消关注自己".to_string()));
        }

        // 强约束：未关注则返回冲突错误
        if !self.follows_repo.exists(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("尚未关注该用户".to_string()));
        }

        // 先计算是否为互相关注（用于推送“好友关系解除”）
        let was_mutual = self
            .follows_repo
            .are_mutual_follows(from_user_id, to_user_id)
            .await?;

        self.follows_repo.delete(from_user_id, to_user_id).await?;

        self.event_bus.publish(AppEvent::Unfollowed {
            follower_id: from_user_id,
            user_id: to_user_id,
            was_mutual,
        });
        Ok(())
    }

    /// 拉黑用户
    pub async fn block_user(&self, from_user_id: i64, req: &BlockRequest) -> Result<()> {
        let to_user_id = req.id;

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
        let to_user_id = req.id;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能解除拉黑自己".to_string()));
        }

        // 强约束：未拉黑则返回冲突错误
        if !self.blocks_repo.exists(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("尚未拉黑该用户".to_string()));
        }
        self.blocks_repo.delete(from_user_id, to_user_id).await?;
        Ok(())
    }

    /// 设置好友备注
    pub async fn set_friend_remark(&self, user_id: i64, req: &SetRemarkRequest) -> Result<()> {
        let friend_id = req.id;

        // 好友 = 互相关注
        if !self
            .follows_repo
            .are_mutual_follows(user_id, friend_id)
            .await?
        {
            return Err(AppError::BadRequest(
                "不是好友关系（未互相关注）".to_string(),
            ));
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

    /// 获取好友列表（好友 = 互相关注）
    pub async fn get_friends_list(
        &self,
        user_id: i64,
        req: &MyFriendsRequest,
    ) -> Result<PaginatedData<FriendListItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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

        // 批量获取用户信息
        let users = self.user_repo.find_profiles(&friend_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取备注信息
        let remarks = self.remarks_repo.get_batch(user_id, &friend_ids).await?;

        // 批量获取在线状态（Redis 优先，数据库 Fallback）
        let online_status_map = self.batch_get_online_status(&friend_ids).await;

        let items: Vec<FriendListItem> = mutual_follows
            .iter()
            .filter_map(|r| {
                let user = user_map.get(&r.user_id)?;
                Some(FriendListItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    remark: remarks.get(&r.user_id).cloned(),
                    is_online: online_status_map.get(&r.user_id).copied(),
                    followed_at: r.followed_at.to_rfc3339(),
                    followed_back_at: r.followed_back_at.to_rfc3339(),
                })
            })
            .collect();

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
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = follow_rows
            .iter()
            .filter_map(|r| {
                let user = user_map.get(&r.following_id)?;
                Some(FollowingItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    followed_at: r.created_at.to_rfc3339(),
                    is_mutual: r.is_mutual,
                })
            })
            .collect();

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
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = follower_rows
            .iter()
            .filter_map(|r| {
                let user = user_map.get(&r.follower_id)?;
                Some(FollowerItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    followed_at: r.created_at.to_rfc3339(),
                    is_following_back: r.is_following_back,
                })
            })
            .collect();

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
        let target_user_id = req.id;

        // 检查 friends_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let friends_visible = settings
            .as_ref()
            .map(|s| s.friends_visible.as_str())
            .unwrap_or("public");

        // 公开 API 只能访问 friends_visible = 'public' 的用户
        if friends_visible != "public" {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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
        let items = users
            .into_iter()
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: u.avatar_url,
                user_type: u.user_type,
            })
            .collect();

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
        let target_user_id = req.id;

        // 检查 following_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let following_visible = settings
            .as_ref()
            .map(|s| s.following_visible.as_str())
            .unwrap_or("public");

        // 公开 API 只能访问 following_visible = 'public' 的用户
        if following_visible != "public" {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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
        let items = users
            .into_iter()
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: u.avatar_url,
                user_type: u.user_type,
            })
            .collect();

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
        let target_user_id = req.id;

        // 检查 followers_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let followers_visible = settings
            .as_ref()
            .map(|s| s.followers_visible.as_str())
            .unwrap_or("public");

        // 公开 API 只能访问 followers_visible = 'public' 的用户
        if followers_visible != "public" {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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
        let items = users
            .into_iter()
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: u.avatar_url,
                user_type: u.user_type,
            })
            .collect();

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
        let cursor = Self::parse_cursor(req.cursor.as_deref())?;

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

        let items = block_rows
            .iter()
            .filter_map(|r| {
                let user = user_map.get(&r.blocked_id)?;
                Some(BlockedUserItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    blocked_at: r.created_at.to_rfc3339(),
                    reason: r.reason.clone(),
                })
            })
            .collect();

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
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

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
        let (is_online, last_online) = self.get_online_status(user_id).await;

        // 5. 获取用户标签
        let tags = self.tag_repo.get_user_tags(user_id).await?;
        let tags = if tags.is_empty() { None } else { Some(tags) };

        // 6. 构造profile
        let profile = MyProfileData {
            user_id: user.user_id,
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            tags,
            email,
            phone,
            location_name,
            is_online,
            last_online,
            created_at: user.created_at.to_rfc3339(),
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
        let target_user_id = req.id;

        // 检查 stats_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let stats_visible = settings
            .as_ref()
            .map(|s| s.stats_visible.as_str())
            .unwrap_or("public");

        // 公开 API 只能访问 stats_visible = 'public' 的用户
        if stats_visible != "public" {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.get_stats(target_user_id).await
    }

    /// 获取当前用户统计数据（/me/ API）
    pub async fn get_my_stats(&self, user_id: i64) -> Result<UserStatsData> {
        self.get_stats(user_id).await
    }

    async fn get_stats(&self, user_id: i64) -> Result<UserStatsData> {
        let (moments_count, friends_count, followers_count, following_count, groups_count) =
            self.user_repo.get_stats_counts(user_id).await?;

        Ok(UserStatsData {
            moments_count: moments_count as i32,
            friends_count: friends_count as i32,
            followers_count: followers_count as i32,
            following_count: following_count as i32,
            groups_count: groups_count as i32,
        })
    }

    /// 获取用户设置
    pub async fn get_settings(&self, user_id: i64) -> Result<UserSettingsData> {
        // 从数据库获取设置
        let settings = self.user_settings_repo.find_by_user_id(user_id).await?;

        // 如果没有设置记录，返回默认值
        if let Some(s) = settings {
            Ok(UserSettingsData {
                privacy_level: s.privacy_level.to_string(),
                location_visible: s.location_visible,
                online_visible: s.online_visible,
                last_online_visible: s.last_online_visible,
                friends_visible: s.friends_visible.to_string(),
                following_visible: s.following_visible.to_string(),
                followers_visible: s.followers_visible.to_string(),
                moments_visible: s.moments_visible.to_string(),
                stats_visible: s.stats_visible.to_string(),
                allow_follow: s.allow_follow,
                allow_private_chat: s.allow_private_chat.to_string(),
                searchable: s.searchable,
                phone_searchable: s.phone_searchable,
                nickname_searchable: s.nickname_searchable,
                location_searchable: s.location_searchable,
            })
        } else {
            // 返回默认设置
            Ok(UserSettingsData {
                privacy_level: "public".to_string(),
                location_visible: true,
                online_visible: true,
                last_online_visible: true,
                friends_visible: "public".to_string(),
                following_visible: "public".to_string(),
                followers_visible: "public".to_string(),
                moments_visible: "public".to_string(),
                stats_visible: "public".to_string(),
                allow_follow: true,
                allow_private_chat: "everyone".to_string(),
                searchable: true,
                phone_searchable: true,
                nickname_searchable: true,
                location_searchable: true,
            })
        }
    }

    /// 更新用户设置
    pub async fn update_settings(&self, user_id: i64, req: &UpdateSettingsRequest) -> Result<()> {
        use crate::repositories::user_settings_repo::UpdateSettingsParams;
        use crate::types::enums::{ChatPermission, PrivacyLevel};
        use std::str::FromStr;

        // 确保设置记录存在
        self.user_settings_repo.create(user_id).await?;

        // 构建更新参数
        let params = UpdateSettingsParams {
            privacy_level: req
                .privacy_level
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            location_visible: req.location_visible,
            online_visible: req.online_visible,
            last_online_visible: req.last_online_visible,
            friends_visible: req
                .friends_visible
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            following_visible: req
                .following_visible
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            followers_visible: req
                .followers_visible
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            moments_visible: req
                .moments_visible
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            stats_visible: req
                .stats_visible
                .as_deref()
                .map(PrivacyLevel::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            allow_follow: req.allow_follow,
            allow_private_chat: req
                .allow_private_chat
                .as_deref()
                .map(ChatPermission::from_str)
                .transpose()
                .map_err(AppError::BadRequest)?,
            searchable: req.searchable,
            phone_searchable: req.phone_searchable,
            nickname_searchable: req.nickname_searchable,
            location_searchable: req.location_searchable,
        };

        self.user_settings_repo.update(user_id, &params).await?;

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
            return Err(AppError::Conflict("该登录ID已被注册".to_string()));
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
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

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

        Ok(AuthData {
            user_id,
            nickname: user.nickname,
            access_token,
            refresh_token: Some(refresh_token),
            access_token_expires_at: (Utc::now() + Duration::seconds(access_token_expiration))
                .to_rfc3339(),
            refresh_token_expires_at: (Utc::now() + Duration::seconds(refresh_token_expiration))
                .to_rfc3339(),
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

    /// 获取用户在线状态（受隐私设置控制，返回 Option 值）
    async fn get_online_status_with_privacy(&self, user_id: i64) -> (Option<bool>, Option<String>) {
        // 获取用户隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();

        // 默认值：true（如果没有设置记录）
        let online_visible = settings.as_ref().map(|s| s.online_visible).unwrap_or(true);
        let last_online_visible = settings
            .as_ref()
            .map(|s| s.last_online_visible)
            .unwrap_or(true);

        // 如果用户不允许显示在线状态，返回 None
        if !online_visible {
            return (None, None);
        }

        // 获取在线状态
        let (is_online, last_online) = self.get_online_status(user_id).await;

        (
            Some(is_online),
            // 如果用户不允许显示最后在线时间，不返回该字段
            if last_online_visible {
                last_online
            } else {
                None
            },
        )
    }

    /// 批量获取用户在线状态（受隐私设置控制）
    async fn batch_get_online_status_with_privacy(
        &self,
        user_ids: &[i64],
    ) -> HashMap<i64, (Option<bool>, Option<String>)> {
        if user_ids.is_empty() {
            return HashMap::new();
        }

        // 1) 批量拿隐私设置（避免 N+1）
        let mut settings_map: HashMap<i64, crate::repositories::user_settings_repo::UserSettings> =
            HashMap::new();
        if let Ok(settings) = self.user_settings_repo.get_batch(user_ids).await {
            settings_map = settings.into_iter().map(|s| (s.user_id, s)).collect();
        }

        // 2) 批量拿在线状态（Redis pipeline + DB ANY）
        let online_status_map = self.batch_get_online_status(user_ids).await;

        // 3) 预生成需要 last_online 的用户列表（离线 + 允许显示）
        let mut need_last_online_ids: Vec<i64> = Vec::new();
        for &user_id in user_ids {
            let settings = settings_map.get(&user_id);
            let online_visible = settings.map(|s| s.online_visible).unwrap_or(true);
            let last_online_visible = settings.map(|s| s.last_online_visible).unwrap_or(true);
            if !online_visible {
                continue;
            }
            let is_online = online_status_map.get(&user_id).copied().unwrap_or(false);
            if !is_online && last_online_visible {
                need_last_online_ids.push(user_id);
            }
        }

        // 4) Redis 批量 MGET last_online
        let mut last_online_map: HashMap<i64, Option<String>> = HashMap::new();
        if !need_last_online_ids.is_empty() {
            let keys: Vec<String> = need_last_online_ids
                .iter()
                .map(|id| format!("presence:last_online:{}", id))
                .collect();

            let mut conn = self.redis.clone();
            let values: redis::RedisResult<Vec<Option<String>>> =
                redis::cmd("MGET").arg(keys).query_async(&mut conn).await;

            if let Ok(vs) = values {
                for (idx, v) in vs.into_iter().enumerate() {
                    last_online_map.insert(need_last_online_ids[idx], v);
                }
            }
        }

        // 5) DB 批量补齐 last_seen_at（仅对 redis miss 的用户）
        let mut miss_ids: Vec<i64> = Vec::new();
        for &user_id in &need_last_online_ids {
            if !last_online_map.contains_key(&user_id)
                || last_online_map
                    .get(&user_id)
                    .and_then(|v| v.as_ref())
                    .is_none()
            {
                miss_ids.push(user_id);
            }
        }
        let mut db_last_seen_map: HashMap<i64, Option<String>> = HashMap::new();
        if !miss_ids.is_empty() {
            if let Ok(rows) = self.online_status_repo.get_batch_status(&miss_ids).await {
                for r in rows {
                    db_last_seen_map.insert(r.user_id, r.last_seen_at.map(|t| t.to_rfc3339()));
                }
            }
        }

        // 6) 组装结果
        let mut result: HashMap<i64, (Option<bool>, Option<String>)> = HashMap::new();
        for &user_id in user_ids {
            let settings = settings_map.get(&user_id);
            let online_visible = settings.map(|s| s.online_visible).unwrap_or(true);
            let last_online_visible = settings.map(|s| s.last_online_visible).unwrap_or(true);

            if !online_visible {
                result.insert(user_id, (None, None));
                continue;
            }

            let is_online = online_status_map.get(&user_id).copied().unwrap_or(false);
            let last_online = if !is_online && last_online_visible {
                last_online_map
                    .get(&user_id)
                    .and_then(|v| v.clone())
                    .or_else(|| db_last_seen_map.get(&user_id).and_then(|v| v.clone()))
            } else {
                None
            };

            result.insert(user_id, (Some(is_online), last_online));
        }

        result
    }

    /// 获取用户在线状态（内部方法，不检查隐私设置）
    async fn get_online_status(&self, user_id: i64) -> (bool, Option<String>) {
        // 1. 从 Redis 查询实时在线状态
        let redis_online: redis::RedisResult<Option<f64>> = redis::cmd("ZSCORE")
            .arg(crate::constants::REDIS_KEY_PRESENCE_ONLINE)
            .arg(user_id)
            .query_async(&mut self.redis.clone())
            .await;

        if redis_online.ok().flatten().is_some() {
            return (true, None);
        }

        // 2. 从 Redis 查询最后上线时间
        let key = format!("presence:last_online:{}", user_id);
        let redis_last_online: redis::RedisResult<Option<String>> = redis::cmd("GET")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await;

        if let Some(last_online) = redis_last_online.ok().flatten() {
            return (false, Some(last_online));
        }

        // 3. 从数据库查询
        match self.online_status_repo.get_status(user_id).await {
            Ok(Some(status)) => {
                let last_online = if !status.is_online {
                    status.last_seen_at.map(|t| t.to_rfc3339())
                } else {
                    None
                };
                (status.is_online, last_online)
            }
            _ => (false, None),
        }
    }

    /// 批量获取用户在线状态
    async fn batch_get_online_status(&self, user_ids: &[i64]) -> HashMap<i64, bool> {
        if user_ids.is_empty() {
            return HashMap::new();
        }

        let mut result: HashMap<i64, bool> = HashMap::new();

        // 1) Redis：使用 pipeline 批量 ZSCORE（减少 RTT）
        let mut conn = self.redis.clone();
        let mut pipe = redis::pipe();
        for &user_id in user_ids {
            pipe.cmd("ZSCORE")
                .arg(crate::constants::REDIS_KEY_PRESENCE_ONLINE)
                .arg(user_id);
        }
        let redis_scores: redis::RedisResult<Vec<Option<f64>>> = pipe.query_async(&mut conn).await;
        if let Ok(scores) = redis_scores {
            for (idx, score) in scores.into_iter().enumerate() {
                if score.is_some() {
                    result.insert(user_ids[idx], true);
                }
            }
        }

        // 2. 从数据库查询 Redis 中不在线的用户
        let not_online_ids: Vec<i64> = user_ids
            .iter()
            .filter(|id| !result.contains_key(id))
            .copied()
            .collect();

        if !not_online_ids.is_empty() {
            if let Ok(db_statuses) = self
                .online_status_repo
                .get_batch_status(&not_online_ids)
                .await
            {
                result.extend(db_statuses.into_iter().map(|s| (s.user_id, s.is_online)));
            }
        }

        // 3. 默认离线
        for &user_id in user_ids {
            result.entry(user_id).or_insert(false);
        }

        result
    }

    /// 批量获取用户位置可见性设置
    async fn batch_get_location_visible(&self, user_ids: &[i64]) -> HashMap<i64, bool> {
        // 默认值：true（如果没有 settings 记录则视为可见，符合“有默认值”的语义）
        let mut result: HashMap<i64, bool> =
            user_ids.iter().copied().map(|id| (id, true)).collect();

        // 真正的批量查询：避免 N+1
        if let Ok(settings) = self.user_settings_repo.get_batch(user_ids).await {
            for s in settings {
                result.insert(s.user_id, s.location_visible);
            }
        }

        result
    }

    /// 获取指定用户加入的公开群组
    pub async fn get_user_groups_list(
        &self,
        req: &UserGroupsRequest,
    ) -> Result<PaginatedData<GroupInfo>> {
        let target_user_id = req.id;

        // 验证用户存在
        if !self.user_repo.exists(target_user_id).await? {
            return Err(AppError::NotFound("用户不存在".to_string()));
        }

        let cursor = Self::parse_cursor(req.cursor.as_deref())?;
        let limit = Validator::validate_limit(req.limit, 100)? as i32;

        let group_member_repo = crate::repositories::group_member_repo::GroupMemberRepository::new(
            self.user_repo.pool().clone(),
        );

        // 单条 JOIN SQL：直接返回 group_info（避免 N+1 与错误的“成员列表limit=1000计数”）
        let (rows, next_cursor) = group_member_repo
            .list_public_group_infos_by_user(target_user_id, limit, cursor)
            .await?;

        let items = rows
            .into_iter()
            .map(|r| GroupInfo {
                id: r.group_id.to_string(),
                name: r.name,
                avatar_url: r.avatar_url,
                members_count: r.members_count,
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

    /// 获取指定用户点赞的瞬间
    pub async fn get_user_liked_moments_list(
        &self,
        req: &UserLikedMomentsRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let target_user_id = req.id;

        // 验证用户存在
        if !self.user_repo.exists(target_user_id).await? {
            return Err(AppError::NotFound("用户不存在".to_string()));
        }

        // 检查 moments_visible 隐私设置（点赞列表也受此控制）
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let moments_visible = settings
            .as_ref()
            .map(|s| s.moments_visible.as_str())
            .unwrap_or("public");

        // 公开 API 只能访问 moments_visible = 'public' 的用户
        if moments_visible != "public" {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        // 验证坐标
        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let cursor = Self::parse_cursor(req.cursor.as_deref())?;
        let limit = Validator::validate_limit(req.limit, 100)? as i32;

        // 获取用户点赞的瞬间
        let moment_repo =
            crate::repositories::moment_repo::MomentRepository::new(self.user_repo.pool().clone());

        // 这里的 get_user_liked_moments 之前已经被重构为返回 (Vec<MomentLikeRow>, Option<String>)
        let (moment_likes, next_cursor) = moment_repo
            .get_user_liked_moments(target_user_id, limit as i64, cursor)
            .await?;

        let moment_uuids: Vec<Uuid> = moment_likes.iter().map(|m| m.moment_id).collect();

        // 批量获取瞬间信息（带距离计算）
        let moments = moment_repo
            .find_by_ids_with_distance(&moment_uuids, req.latitude, req.longitude)
            .await?;

        // 获取用户信息
        let mut user_ids: Vec<i64> = moments.iter().map(|m| m.user_id).collect();
        user_ids.sort_unstable();
        user_ids.dedup();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 获取统计数据
        let stats = moment_repo.get_stats_batch(&moment_uuids).await?;

        // 获取标签
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_uuids).await?;

        let items: Vec<MomentSearchItem> = moments
            .into_iter()
            .filter_map(|m| {
                let user = user_map.get(&m.user_id)?;
                let (likes_count, comments_count) =
                    stats.get(&m.moment_id).copied().unwrap_or((0, 0));
                let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();

                Some(MomentSearchItem {
                    id: m.moment_id.to_string(),
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    description: m.description,
                    images: Some(m.images),
                    tags,
                    created_at: m.created_at.to_rfc3339(),
                    likes_count: likes_count as i32,
                    comments_count: comments_count as i32,
                    geo: GeoData {
                        latitude: m.latitude,
                        longitude: m.longitude,
                        location_name: m.location_name,
                        distance: m.distance as i32,
                        degrees: m.degrees,
                    },
                })
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
}
