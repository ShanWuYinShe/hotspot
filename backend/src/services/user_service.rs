use std::collections::HashMap;
use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            SendFriendRequestRequest, BlockUserRequest, OpenTheDoorRequest, DiscoverUsersRequest,
            SetFriendRemarkRequest, UpdateLocationRequest,
            UpdateProfileRequest, UpdateSettingsRequest, UpgradeAccountRequest,
            ListFriendsRequest, ListFollowingRequest, ListFollowersRequest, GetBlockedUsersRequest,
            GetFriendRequestsRequest, DeleteAccountRequest, GetStatsRequest,
        },
        response::{
            PaginatedData, PaginationMeta, UserDetailData,
            UserRelationship, DiscoveredUserItem, MyProfileData,
            UserStatsData, UserSettingsData, AuthData,
            FriendRequestItem, UserInfo, FriendListItem, SendFriendRequestData,
        },
    },
    error::{AppError, Result},
    repositories::{
        user_location_history_repo::UserLocationHistoryRepository,
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
        user_relationship_repo::UserRelationshipRepository,
        user_follows_repo::UserFollowsRepository,
        user_blocks_repo::UserBlocksRepository,
        user_friend_remarks_repo::UserFriendRemarksRepository,
    },
    utils::{geo::GeoUtil, validation::Validator, password::PasswordUtil, jwt::JwtUtil},
    constants::{LOCATION_CHANGE_THRESHOLD, DEFAULT_LOCATION_NAME},
};

pub struct UserService {
    user_repo: UserRepository,
    user_settings_repo: UserSettingsRepository,
    user_location_history_repo: UserLocationHistoryRepository,
    relationship_repo: UserRelationshipRepository,
    follows_repo: UserFollowsRepository,
    blocks_repo: UserBlocksRepository,
    remarks_repo: UserFriendRemarksRepository,
    redis: ConnectionManager,
}

impl UserService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            user_repo: UserRepository::new(pool.clone()),
            user_settings_repo: UserSettingsRepository::new(pool.clone()),
            user_location_history_repo: UserLocationHistoryRepository::new(pool.clone()),
            relationship_repo: UserRelationshipRepository::new(pool.clone()),
            follows_repo: UserFollowsRepository::new(pool.clone()),
            blocks_repo: UserBlocksRepository::new(pool.clone()),
            remarks_repo: UserFriendRemarksRepository::new(pool.clone()),
            redis: redis.clone(),
        }
    }

    /// 解析 UUID cursor
    fn parse_uuid_cursor(cursor: Option<&str>) -> Result<Option<Uuid>> {
        match cursor {
            Some(c) => {
                let uuid = Uuid::parse_str(c)
                    .map_err(|_| AppError::BadRequest("无效的 cursor 格式".to_string()))?;
                Ok(Some(uuid))
            }
            None => Ok(None),
        }
    }

    /// 发现用户 (合并 search_by_name 和 search_by_location)
    pub async fn discover(
        &self,
        req: &DiscoverUsersRequest,
    ) -> Result<PaginatedData<DiscoveredUserItem>> {
        let limit = Validator::validate_pagination(req.limit, 100)?;
        // 解析 cursor（UUID）
        let cursor = Self::parse_uuid_cursor(req.cursor.as_deref())?;

        // 如果提供了位置，使用位置搜索
        if let (Some(lat), Some(lng)) = (req.latitude, req.longitude) {
            GeoUtil::validate_coordinates(lat, lng)?;
            let radius = req.radius.unwrap_or(5000.0);
            let radius = Validator::validate_radius(Some(radius as i32), 50000)?;
            
            let nearby_users = self
                .user_repo
                .search_nearby_users(lat, lng, radius as f64, (limit + 1) as i64, cursor)
                .await?;

            let has_more = nearby_users.len() > limit as usize;
            let users_to_return: Vec<_> = nearby_users.into_iter().take(limit as usize).collect();

            if users_to_return.is_empty() {
                return Ok(PaginatedData {
                    items: vec![],
                    pagination: PaginationMeta {
                        has_more: false,
                        next_cursor: None,
                    },
                });
            }

            // 批量获取用户最新位置
            let user_ids: Vec<i64> = users_to_return.iter().map(|u| u.user_id).collect();
            let locations = self.user_location_history_repo.get_latest_batch(&user_ids).await?;

            let items: Vec<DiscoveredUserItem> = users_to_return.iter().filter_map(|user| {
                let location = locations.get(&user.user_id)?;
                let distance = GeoUtil::haversine_distance(lat, lng, location.latitude, location.longitude);
                Some(DiscoveredUserItem {
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                    location_name: location.location_name.clone().unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string()),
                    distance: distance as i32,
                })
            }).collect();

            let next_cursor = if has_more {
                users_to_return.last().map(|u| u.cursor_id.to_string())
            } else {
                None
            };

            return Ok(PaginatedData {
                items,
                pagination: PaginationMeta {
                    has_more,
                    next_cursor,
                },
            });
        } 
        
        // 如果没有位置，按关键词搜索 (Fallback)
        if let Some(keyword) = &req.keyword {
            let (users, next_cursor) = self.user_repo.search_by_name(keyword, limit as i64, cursor).await?;
            
            // 批量获取位置信息
            let user_ids: Vec<i64> = users.iter().map(|u| u.user_id).collect();
            let locations = self.user_location_history_repo.get_latest_batch(&user_ids).await?;
            
            let items = users.into_iter().map(|u| {
                let location_name = locations.get(&u.user_id)
                    .and_then(|loc| loc.location_name.clone())
                    .unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());
                DiscoveredUserItem {
                    user_id: u.user_id.to_string(),
                    nickname: u.nickname,
                    avatar_url: u.avatar_url,
                    location_name,
                    distance: 0,
                }
            }).collect();

            return Ok(PaginatedData {
                items,
                pagination: PaginationMeta {
                    has_more: next_cursor.is_some(),
                    next_cursor,
                },
            });
        }

        // 既没有位置也没有关键词，返回空
        Ok(PaginatedData {
            items: vec![],
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 按ID获取用户详情 (Open The Door)
    pub async fn open_the_door(
        &self,
        req: &OpenTheDoorRequest,
        current_user_id: Option<i64>,
    ) -> Result<UserDetailData> {
        let user_id = req
            .id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        // 构建关系信息
        let relationship = if let Some(current_id) = current_user_id {
            if current_id != user_id {
                self.get_relationship_info(current_id, user_id).await?
            } else {
                // 自己看自己
                UserRelationship {
                    is_friend: false,
                    is_following: false,
                    is_followed_by: false,
                    is_blocked: false,
                    is_blocking: false,
                    friend_request_status: None,
                }
            }
        } else {
            // 未登录
            UserRelationship {
                is_friend: false,
                is_following: false,
                is_followed_by: false,
                is_blocked: false,
                is_blocking: false,
                friend_request_status: None,
            }
        };

        // 获取最新位置信息
        let location_name = self
            .user_location_history_repo
            .get_latest(user.user_id)
            .await?
            .and_then(|loc| loc.location_name)
            .unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());

        // 获取在线状态
        let is_online = self.get_online_status(user.user_id).await;
        let last_seen = if !is_online {
            self.get_last_seen(user.user_id).await
        } else {
            None
        };

        Ok(UserDetailData {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            location_name: Some(location_name),
            is_online,
            last_seen,
            created_at: user.created_at.to_rfc3339(),
            relationship,
        })
    }

    /// 更新用户位置
    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;

        // 获取用户最新位置
        let latest_location = self
            .user_location_history_repo
            .get_latest(user_id)
            .await?;

        // 检查是否需要插入新记录
        let should_insert = if let Some(latest) = latest_location {
            // 计算与最新位置的距离
            let distance = GeoUtil::haversine_distance(
                latest.latitude,
                latest.longitude,
                req.latitude,
                req.longitude,
            );
            // 超过阈值才插入
            distance > LOCATION_CHANGE_THRESHOLD
        } else {
            // 没有历史位置，插入新记录
            true
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

        Ok(())
    }

    /// 更新用户资料
    pub async fn update_profile(&self, user_id: i64, req: &UpdateProfileRequest) -> Result<()> {
        if let Some(nickname) = &req.nickname {
            Validator::validate_not_empty(nickname, "昵称")?;
            Validator::validate_length(nickname, "昵称", 1, 100)?;
        }

        self.user_repo
            .update_profile(
                user_id,
                req.nickname.as_deref(),
                req.avatar_url.as_deref(),
                req.bio.as_deref(),
            )
            .await?;

        Ok(())
    }

    /// 发送好友申请
    pub async fn send_friend_request(&self, from_user_id: i64, req: &SendFriendRequestRequest) -> Result<SendFriendRequestData> {
        let to_user_id = req
            .to_user_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能添加自己为好友".to_string()));
        }

        // 检查对方是否存在
        self.user_repo
            .find_by_id(to_user_id)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        if self.relationship_repo.are_friends(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("已经是好友".to_string()));
        }

        if let Some(_) = self.relationship_repo.find_relationship(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("已发送好友申请".to_string()));
        }

        let request_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, req.message.clone())
            .await?;

        Ok(SendFriendRequestData {
            id: request_id.to_string()
        })
    }

    /// 接受好友申请
    pub async fn accept_friend_request(&self, user_id: i64, req: &crate::dto::request::AcceptFriendRequestRequest) -> Result<()> {
        let request_id = Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的请求ID".to_string()))?;

        let rel = self.relationship_repo.find_by_id(request_id).await?
            .ok_or(AppError::NotFound("好友申请不存在".to_string()))?;
        
        if rel.to_user_id != user_id {
            return Err(AppError::Forbidden("无权操作".to_string()));
        }
        
        if rel.status != "pending" {
             return Err(AppError::BadRequest("申请状态不正确".to_string()));
        }

        self.relationship_repo.update_status(request_id, "accepted").await?;

        Ok(())
    }

    /// 拒绝好友申请
    pub async fn reject_friend_request(&self, user_id: i64, req: &crate::dto::request::RejectFriendRequestRequest) -> Result<()> {
        let request_id = Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的请求ID".to_string()))?;

        let rel = self.relationship_repo.find_by_id(request_id).await?
            .ok_or(AppError::NotFound("好友申请不存在".to_string()))?;
        
        if rel.to_user_id != user_id {
            return Err(AppError::Forbidden("无权操作".to_string()));
        }

        self.relationship_repo.update_status(request_id, "rejected").await?;
        Ok(())
    }

    /// 取消好友申请
    pub async fn cancel_friend_request(&self, user_id: i64, req: &crate::dto::request::CancelFriendRequestRequest) -> Result<()> {
        let request_id = Uuid::parse_str(&req.id).map_err(|_| AppError::BadRequest("无效的请求ID".to_string()))?;

        let rel = self.relationship_repo.find_by_id(request_id).await?
            .ok_or(AppError::NotFound("好友申请不存在".to_string()))?;
        
        // 只有发送者可以取消申请
        if rel.from_user_id != user_id {
            return Err(AppError::Forbidden("无权操作".to_string()));
        }

        // 只能取消待处理的申请
        if rel.status != "pending" {
            return Err(AppError::Conflict("申请已处理".to_string()));
        }

        self.relationship_repo.delete(request_id).await?;
        Ok(())
    }

    /// 删除好友
    pub async fn remove_friend(&self, user_id: i64, req: &crate::dto::request::RemoveFriendRequest) -> Result<()> {
        let friend_id = req.user_id.parse::<i64>().map_err(|_| AppError::BadRequest("Invalid user_id".to_string()))?;
        
        self.relationship_repo.delete_friendship(user_id, friend_id).await?;

        Ok(())
    }

    /// 关注用户
    pub async fn follow(&self, from_user_id: i64, req: &crate::dto::request::FollowUserRequest) -> Result<()> {
        let to_user_id = req.user_id.parse::<i64>().map_err(|_| AppError::BadRequest("Invalid user_id".to_string()))?;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能关注自己".to_string()));
        }

        if self.follows_repo.exists(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("已关注该用户".to_string()));
        }

        self.follows_repo.create(from_user_id, to_user_id).await?;

        Ok(())
    }

    /// 取消关注
    pub async fn unfollow(&self, from_user_id: i64, req: &crate::dto::request::UnfollowUserRequest) -> Result<()> {
        let to_user_id = req.user_id.parse::<i64>().map_err(|_| AppError::BadRequest("Invalid user_id".to_string()))?;
        
        self.follows_repo.delete(from_user_id, to_user_id).await?;

        Ok(())
    }

    /// 拉黑用户
    pub async fn block_user(&self, from_user_id: i64, req: &BlockUserRequest) -> Result<()> {
        let to_user_id = req
            .user_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能拉黑自己".to_string()));
        }

        if self.blocks_repo.exists(from_user_id, to_user_id).await? {
            return Err(AppError::Conflict("已拉黑该用户".to_string()));
        }

        self.relationship_repo.delete_friendship(from_user_id, to_user_id).await.ok();
        self.follows_repo.delete(from_user_id, to_user_id).await.ok();

        self.blocks_repo.create(from_user_id, to_user_id, req.reason.clone()).await?;

        Ok(())
    }

    /// 解除拉黑
    pub async fn unblock_user(&self, from_user_id: i64, req: &crate::dto::request::UnblockUserRequest) -> Result<()> {
        let to_user_id = req.user_id.parse::<i64>().map_err(|_| AppError::BadRequest("Invalid user_id".to_string()))?;
        
        self.blocks_repo.delete(from_user_id, to_user_id).await?;

        Ok(())
    }

    /// 设置好友备注
    pub async fn set_friend_remark(
        &self,
        user_id: i64,
        req: &SetFriendRemarkRequest,
    ) -> Result<()> {
        let friend_id = req
            .user_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        if !self.relationship_repo.are_friends(user_id, friend_id).await? {
            return Err(AppError::BadRequest("不是好友关系".to_string()));
        }
        
        if let Some(remark) = &req.remark {
            if !remark.is_empty() {
                self.remarks_repo.upsert(user_id, friend_id, remark.clone()).await?;
            } else {
                self.remarks_repo.delete(user_id, friend_id).await.ok();
            }
        } else {
            self.remarks_repo.delete(user_id, friend_id).await.ok();
        }

        Ok(())
    }

    /// 获取好友列表
    pub async fn list_friends(
        &self,
        user_id: i64,
        req: &ListFriendsRequest,
    ) -> Result<PaginatedData<FriendListItem>> {
        let limit = Validator::validate_pagination(req.limit, 100)? as i32;
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        
        let status = req.status.as_deref();

        let (rels, next_cursor) = self.relationship_repo.list_from_user(
            user_id, 
            status, 
            limit, 
            cursor
        ).await?;

        if rels.is_empty() {
             return Ok(PaginatedData { items: vec![], pagination: PaginationMeta { has_more: false, next_cursor: None } });
        }

        let friend_ids: Vec<i64> = rels.iter().map(|r| {
            if r.from_user_id == user_id {
                r.to_user_id
            } else {
                r.from_user_id
            }
        }).collect();
        
        // 批量获取用户信息
        let users = self.user_repo.find_by_ids(&friend_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取备注信息
        let remarks = self.remarks_repo.get_batch(user_id, &friend_ids).await?;

        let items: Vec<FriendListItem> = rels.iter().filter_map(|r| {
            let friend_id = if r.from_user_id == user_id { r.to_user_id } else { r.from_user_id };
            let user = user_map.get(&friend_id)?;
            Some(FriendListItem {
                user: UserInfo {
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                },
                relation_id: r.request_id.to_string(),
                status: r.status.clone(),
                remark: remarks.get(&friend_id).cloned(),
                is_online: None,
            })
        }).collect();

        let next_cursor_str = next_cursor.map(|id| id.to_string());

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取关注列表
    pub async fn list_following(
        &self,
        user_id: i64,
        req: &ListFollowingRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let limit = Validator::validate_pagination(req.limit, 100)? as i32;
        let cursor = Self::parse_uuid_cursor(req.cursor.as_deref())?;
        
        let (user_ids, next_cursor) = self.follows_repo.list_following(user_id, limit, cursor).await?;

        if user_ids.is_empty() {
             return Ok(PaginatedData { items: vec![], pagination: PaginationMeta { has_more: false, next_cursor: None } });
        }

        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let items = users.into_iter().map(|u| UserInfo {
            user_id: u.user_id.to_string(),
            nickname: u.nickname,
            avatar_url: u.avatar_url,
        }).collect();

        let next_cursor_str = next_cursor.map(|id| id.to_string());

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取粉丝列表
    pub async fn list_followers(
        &self,
        user_id: i64,
        req: &ListFollowersRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let limit = Validator::validate_pagination(req.limit, 100)? as i32;
        let cursor = Self::parse_uuid_cursor(req.cursor.as_deref())?;
        
        let (user_ids, next_cursor) = self.follows_repo.list_followers(user_id, limit, cursor).await?;

        if user_ids.is_empty() {
             return Ok(PaginatedData { items: vec![], pagination: PaginationMeta { has_more: false, next_cursor: None } });
        }

        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let items = users.into_iter().map(|u| UserInfo {
            user_id: u.user_id.to_string(),
            nickname: u.nickname,
            avatar_url: u.avatar_url,
        }).collect();

        let next_cursor_str = next_cursor.map(|id| id.to_string());

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }

    /// 获取拉黑列表
    pub async fn get_blocked_users(
        &self,
        user_id: i64,
        req: &GetBlockedUsersRequest,
    ) -> Result<PaginatedData<UserInfo>> {
        let limit = Validator::validate_pagination(req.limit, 100)? as i32;
        let cursor = Self::parse_uuid_cursor(req.cursor.as_deref())?;
        
        let (user_ids, next_cursor) = self.blocks_repo.list_blocked(user_id, limit, cursor).await?;

        if user_ids.is_empty() {
             return Ok(PaginatedData { items: vec![], pagination: PaginationMeta { has_more: false, next_cursor: None } });
        }

        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let items = users.into_iter().map(|u| UserInfo {
            user_id: u.user_id.to_string(),
            nickname: u.nickname,
            avatar_url: u.avatar_url,
        }).collect();

        let next_cursor_str = next_cursor.map(|id| id.to_string());

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor_str.is_some(),
                next_cursor: next_cursor_str,
            },
        })
    }
    
    /// 获取好友申请列表 (Received)
    pub async fn get_friend_requests(
        &self,
        user_id: i64,
        req: &GetFriendRequestsRequest,
    ) -> Result<PaginatedData<FriendRequestItem>> {
        let limit = Validator::validate_pagination(req.limit, 100)? as i32;
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        
        let status = req.status.as_deref();
        let (rels, next_cursor) = match status {
            Some("sent") => self.relationship_repo.list_from_user(user_id, status, limit, cursor).await?,
            Some("received") | None => self.relationship_repo.list_to_user(user_id, status, limit, cursor).await?,
            Some("all") => {
                let (sent, _) = self.relationship_repo.list_from_user(user_id, None, limit, cursor).await?;
                let (received, cursor) = self.relationship_repo.list_to_user(user_id, None, limit, cursor).await?;
                let mut all = sent;
                all.extend(received);
                (all, cursor)
            },
            _ => return Err(AppError::BadRequest("无效的状态参数".to_string())),
        };
        
        if rels.is_empty() {
             return Ok(PaginatedData { items: vec![], pagination: PaginationMeta { has_more: false, next_cursor: None } });
        }

        let mut user_ids = Vec::new();
        for r in &rels {
            user_ids.push(r.from_user_id);
            user_ids.push(r.to_user_id);
        }
        user_ids.sort();
        user_ids.dedup();
        
        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = rels.into_iter().filter_map(|r| {
            let from_user = user_map.get(&r.from_user_id)?;
            let to_user = user_map.get(&r.to_user_id)?;
            Some(FriendRequestItem {
                id: r.request_id.to_string(),
                from_user: UserInfo {
                     user_id: from_user.user_id.to_string(),
                     nickname: from_user.nickname.clone(),
                     avatar_url: from_user.avatar_url.clone(),
                },
                to_user: UserInfo {
                     user_id: to_user.user_id.to_string(),
                     nickname: to_user.nickname.clone(),
                     avatar_url: to_user.avatar_url.clone(),
                },
                message: r.message.clone(),
                status: r.status.clone(),
                created_at: r.created_at.to_rfc3339(),
            })
        }).collect();

        let next_cursor_str = next_cursor.map(|id| id.to_string());

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
        let (email, phone) = user.login_id.as_ref()
            .map(|id| Self::parse_login_id(id))
            .unwrap_or((None, None));

        // 3. 获取最新位置名称
        let location_name = self.user_location_history_repo.get_latest(user_id).await?
            .and_then(|loc| loc.location_name);

        // 4. 获取在线状态
        let is_online = self.get_online_status(user_id).await;
        let last_seen = if !is_online {
            self.get_last_seen(user_id).await
        } else {
            None
        };

        // 5. 构造profile
        let profile = MyProfileData {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            email,
            phone,
            location_name,
            is_online,
            last_seen,
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

    /// 获取用户统计数据
    pub async fn get_stats(&self, user_id: i64, req: &GetStatsRequest) -> Result<UserStatsData> {
        // 如果请求中有id，则查询该用户，否则查询自己
        let target_user_id = if let Some(ref id) = req.id {
            id.parse::<i64>()
                .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?
        } else {
            user_id
        };

        // 获取各项统计数据
        let moments_count = self.user_repo.count_moments(target_user_id).await?;
        let friends_count = self.relationship_repo.count_friends(target_user_id).await?;
        let followers_count = self.follows_repo.count_followers(target_user_id).await?;
        let following_count = self.follows_repo.count_following(target_user_id).await?;
        let groups_count = self.user_repo.count_groups(target_user_id).await?;

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
        let (privacy_level, location_sharing, allow_follow, allow_private_chat) = 
            self.user_repo.get_settings(user_id).await?;

        Ok(UserSettingsData {
            privacy_level,
            is_location_sharing: location_sharing,
            can_be_followed: allow_follow,
            allow_private_chat,
        })
    }

    /// 更新用户设置
    pub async fn update_settings(
        &self,
        user_id: i64,
        req: &UpdateSettingsRequest,
    ) -> Result<()> {
        self.user_settings_repo.update(
            user_id,
            req.privacy_level.as_deref(),
            req.is_location_sharing,
            req.can_be_followed,
            req.allow_private_chat,
        ).await?;

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

        // 3. 对密码进行哈希
        let _password_hash = PasswordUtil::hash_password(&req.password_hash)?;

        // 4. 更新用户信息 (TODO: 需要在UserRepository添加upgrade_account方法)
        // self.user_repo.upgrade_account(user_id, &req.login_id, &password_hash).await?;

        // 5. 获取更新后的用户信息
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        // 6. 生成新的Token
        let access_token =
            JwtUtil::generate_token(user_id, access_token_expiration, jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, refresh_token_expiration, jwt_secret)?;

        // 7. 存储Refresh Token到Redis
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthData {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: (chrono::Utc::now() + chrono::Duration::seconds(access_token_expiration)).to_rfc3339(),
        })
    }

    /// 删除账户
    pub async fn delete_account(&self, user_id: i64, req: &DeleteAccountRequest) -> Result<()> {
        if req.confirmation != "DELETE_MY_ACCOUNT" {
            return Err(AppError::BadRequest("确认字符串不匹配".to_string()));
        }

        self.user_repo.soft_delete(user_id).await?;

        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("DEL")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(())
    }

    /// 获取用户关系信息 (内部辅助)
    async fn get_relationship_info(
        &self,
        from_user_id: i64,
        to_user_id: i64,
    ) -> Result<UserRelationship> {
        // 检查好友关系
        let is_friend = self.relationship_repo.are_friends(from_user_id, to_user_id).await?;
        
        // 检查待处理的好友申请
        let friend_request_status = if let Some(req) = self.relationship_repo.find_relationship(from_user_id, to_user_id).await? {
            if req.status == "pending" {
                Some("sent".to_string())
            } else {
                None
            }
        } else if let Some(req) = self.relationship_repo.find_relationship(to_user_id, from_user_id).await? {
            if req.status == "pending" {
                Some("received".to_string())
            } else {
                None
            }
        } else {
            None
        };

        // 检查拉黑关系 (我拉黑他)
        let is_blocking = self.blocks_repo.exists(from_user_id, to_user_id).await?;

        // 检查被拉黑关系 (他拉黑我)
        let is_blocked = self.blocks_repo.exists(to_user_id, from_user_id).await?;

        // 检查关注关系 (我关注他)
        let is_following = self.follows_repo.exists(from_user_id, to_user_id).await?;

        // 检查被关注关系 (他关注我)
        let is_followed_by = self.follows_repo.exists(to_user_id, from_user_id).await?;

        Ok(UserRelationship {
            is_friend,
            is_following,
            is_followed_by,
            is_blocked,
            is_blocking,
            friend_request_status,
        })
    }

    /// 从 Redis 获取用户在线状态
    async fn get_online_status(&self, user_id: i64) -> bool {
        let key = format!("{}:{}", crate::constants::REDIS_KEY_PRESENCE_ONLINE, user_id);
        let result: redis::RedisResult<Option<String>> = redis::cmd("GET")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await;
        result.ok().flatten().is_some()
    }

    /// 从 Redis 获取用户最后在线时间
    async fn get_last_seen(&self, user_id: i64) -> Option<String> {
        let key = format!("presence:last_seen:{}", user_id);
        let result: redis::RedisResult<Option<String>> = redis::cmd("GET")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await;
        result.ok().flatten()
    }
}
