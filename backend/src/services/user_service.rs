use redis::aio::ConnectionManager;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            AddFriendRequest, BlockRequest, SearchUserByIdRequest, SearchUserByLocationRequest,
            SearchUserByNameRequest, SetFriendRemarkRequest, UpdateLocationRequest,
            UpdateProfileRequest, UpdateSettingsRequest, UpgradeAccountRequest,
        },
        response::{
            FriendResponse, NearbyUserResponse, PaginatedResponse, UserProfileResponse,
            UserRelationshipInfo, UserResponse, GetMeResponse, GetMeProfileResponse,
            GetMeStatsResponse, GetMeUnreadResponse, GetSettingsResponse, AuthResponse,
        },
    },
    error::{AppError, Result},
    repositories::{
        user_location_history_repo::UserLocationHistoryRepository,
        user_relationship_repo::UserRelationshipRepository, 
        user_repo::UserRepository,
        message_read_repo::MessageReadRepository,
        inbox_repo::InboxRepository,
    },
    types::enums::{RelationshipType, RelationshipStatus},
    utils::{geo::GeoUtil, validation::Validator},
    constants::LOCATION_CHANGE_THRESHOLD,
};

pub struct UserService {
    user_repo: UserRepository,
    user_location_history_repo: UserLocationHistoryRepository,
    relationship_repo: UserRelationshipRepository,
    message_read_repo: MessageReadRepository,
    inbox_repo: InboxRepository,
    redis: ConnectionManager,
}

impl UserService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            user_repo: UserRepository::new(pool.clone()),
            user_location_history_repo: UserLocationHistoryRepository::new(pool.clone()),
            relationship_repo: UserRelationshipRepository::new(pool.clone()),
            message_read_repo: MessageReadRepository::new(pool.clone()),
            inbox_repo: InboxRepository::new(pool.clone()),
            redis: redis.clone(),
        }
    }

    /// 按名称搜索用户
    pub async fn search_by_name(
        &self,
        name: &str,
        limit: Option<i32>,
        cursor: Option<String>,
    ) -> Result<PaginatedResponse<UserResponse>> {
        Validator::validate_not_empty(name, "搜索关键词")?;
        let limit = Validator::validate_pagination(limit, 100)?;

        let (users, next_cursor) = self.user_repo.search_by_name(name, limit, cursor).await?;

        let items = users
            .into_iter()
            .map(|u| UserResponse {
                user_id: u.user_id.to_string(),
                nickname: u.nickname,
                avatar_url: u.avatar_url,
            })
            .collect();

        Ok(PaginatedResponse {
            items,
            has_more: next_cursor.is_some(),
            next_cursor,
        })
    }

    /// 按ID搜索用户
    pub async fn search_by_id(
        &self,
        req: &SearchUserByIdRequest,
        current_user_id: Option<i64>,
    ) -> Result<UserProfileResponse> {
        let user_id = req
            .id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        // 构建关系信息（如果已登录）
        let relationship = if let Some(current_id) = current_user_id {
            if current_id != user_id {
                Some(self.get_relationship_info(current_id, user_id).await?)
            } else {
                None
            }
        } else {
            None
        };

        // 获取最新位置信息
        let location_name = self
            .user_location_history_repo
            .get_latest_by_user_id(user.user_id)
            .await?
            .map(|loc| loc.location_name);

        Ok(UserProfileResponse {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            location_name,
            is_online: None, // TODO: 从Redis获取在线状态
            last_seen: user.last_active_at,
            relationship,
        })
    }

    /// 按位置搜索用户
    pub async fn search_by_location(
        &self,
        req: &SearchUserByLocationRequest,
    ) -> Result<PaginatedResponse<NearbyUserResponse>> {
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;
        let radius_i32 = req.radius.map(|r| r as i32);
        let radius = Validator::validate_radius(radius_i32, 50000)?;
        let limit = Validator::validate_pagination(req.limit, 100)?;

        // 使用 user_location_history_repo 搜索附近用户
        let nearby_users = self
            .user_location_history_repo
            .search_nearby_users(req.latitude, req.longitude, radius as f64, limit as i64)
            .await?;

        let mut items = Vec::new();
        for (user_id, distance) in nearby_users {
            // 获取用户详情
            if let Some(user) = self.user_repo.find_by_id(user_id).await? {
                // 获取最新位置信息
                if let Some(location) = self
                    .user_location_history_repo
                    .get_latest_by_user_id(user_id)
                    .await?
                {
                    items.push(NearbyUserResponse {
                        user_id: user.user_id.to_string(),
                        nickname: user.nickname,
                        avatar_url: user.avatar_url,
                        location_name: Some(location.location_name),
                        distance,
                        is_online: None, // TODO: 从 Redis 获取在线状态
                    });
                }
            }
        }

        // 游标分页逻辑（简化版）
        let has_more = items.len() >= limit as usize;
        let next_cursor = if has_more {
            items.last().map(|item| item.user_id.clone())
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 更新用户位置
    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;

        // 获取用户最新位置
        let latest_location = self
            .user_location_history_repo
            .get_latest_by_user_id(user_id)
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
                .create(
                    user_id,
                    req.latitude,
                    req.longitude,
                    req.location_name.as_deref().unwrap_or("未知位置"),
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

    /// 添加好友
    pub async fn add_friend(&self, from_user_id: i64, req: &AddFriendRequest) -> Result<(String, String)> {
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

        // 检查是否已经是好友或有待处理的申请
        if let Some(existing) = self
            .relationship_repo
            .find_mutual_relationship(from_user_id, to_user_id, RelationshipType::Friend)
            .await?
        {
            if existing.status == RelationshipStatus::Accepted {
                return Err(AppError::Conflict("已经是好友".to_string()));
            } else if existing.status == RelationshipStatus::Pending {
                // 如果对方也向我发送了申请，自动接受
                if existing.to_user_id == from_user_id {
                    self.relationship_repo
                        .update_status(existing.relationship_id, RelationshipStatus::Accepted)
                        .await?;
                    return Ok((existing.relationship_id.to_string(), RelationshipStatus::Accepted.to_string()));
                }
                return Err(AppError::Conflict("已发送好友申请".to_string()));
            }
        }

        // 创建好友申请
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, RelationshipType::Friend, RelationshipStatus::Pending)
            .await?;

        Ok((relationship_id.to_string(), "pending".to_string()))
    }

    /// 接受好友申请
    pub async fn accept_friend(&self, user_id: i64, relationship_id: i64) -> Result<()> {
        // 查找关系
        let relationship = self
            .relationship_repo
            .find_relationship(relationship_id, user_id, RelationshipType::Friend)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        if relationship.to_user_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        if relationship.status != RelationshipStatus::Pending {
            return Err(AppError::BadRequest("该申请已处理".to_string()));
        }

        // 更新状态
        self.relationship_repo
            .update_status(relationship_id, RelationshipStatus::Accepted)
            .await?;

        Ok(())
    }

    /// 拒绝好友申请
    pub async fn reject_friend(&self, user_id: i64, relationship_id: i64) -> Result<()> {
        // 查找关系
        let relationship = self
            .relationship_repo
            .find_relationship(relationship_id, user_id, RelationshipType::Friend)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        if relationship.to_user_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        // 删除申请
        self.relationship_repo.delete(relationship_id).await?;

        Ok(())
    }

    /// 删除好友
    pub async fn remove_friend(&self, user_id: i64, friend_id: i64) -> Result<()> {
        // 删除双向关系
        self.relationship_repo
            .delete_by_users(user_id, friend_id, RelationshipType::Friend)
            .await?;
        self.relationship_repo
            .delete_by_users(friend_id, user_id, RelationshipType::Friend)
            .await?;

        Ok(())
    }

    /// 关注用户
    pub async fn follow(&self, from_user_id: i64, to_user_id: i64) -> Result<String> {
        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能关注自己".to_string()));
        }

        // 检查是否已关注
        if let Some(_) = self
            .relationship_repo
            .find_relationship(from_user_id, to_user_id, RelationshipType::Following)
            .await?
        {
            return Err(AppError::Conflict("已关注该用户".to_string()));
        }

        // 创建关注关系
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, RelationshipType::Following, RelationshipStatus::Accepted)
            .await?;

        Ok(relationship_id.to_string())
    }

    /// 取消关注
    pub async fn unfollow(&self, from_user_id: i64, to_user_id: i64) -> Result<()> {
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, RelationshipType::Following)
            .await?;

        Ok(())
    }

    /// 拉黑用户
    pub async fn block(&self, from_user_id: i64, req: &BlockRequest) -> Result<String> {
        let to_user_id = req
            .user_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        if from_user_id == to_user_id {
            return Err(AppError::BadRequest("不能拉黑自己".to_string()));
        }

        // 检查是否已拉黑
        if let Some(_) = self
            .relationship_repo
            .find_relationship(from_user_id, to_user_id, RelationshipType::Blocked)
            .await?
        {
            return Err(AppError::Conflict("已拉黑该用户".to_string()));
        }

        // 删除好友和关注关系
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, RelationshipType::Friend)
            .await
            .ok();
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, RelationshipType::Following)
            .await
            .ok();

        // 创建拉黑关系
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, RelationshipType::Blocked, RelationshipStatus::Accepted)
            .await?;

        Ok(relationship_id.to_string())
    }

    /// 解除拉黑
    pub async fn unblock(&self, from_user_id: i64, to_user_id: i64) -> Result<()> {
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, RelationshipType::Blocked)
            .await?;

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

        // 查找好友关系
        let relationship = self
            .relationship_repo
            .find_mutual_relationship(user_id, friend_id, RelationshipType::Friend)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        if relationship.status != RelationshipStatus::Accepted {
            return Err(AppError::BadRequest("不是好友关系".to_string()));
        }

        // 更新备注（只更新自己这一侧的）
        let my_relationship = if relationship.from_user_id == user_id {
            relationship
        } else {
            self.relationship_repo
                .find_relationship(user_id, friend_id, RelationshipType::Friend)
                .await?
                .ok_or(AppError::NotFound("关系不存在".to_string()))?
        };

        self.relationship_repo
            .set_remark(my_relationship.relationship_id, &req.remark)
            .await?;

        Ok(())
    }

    /// 获取好友列表
    pub async fn list_friends(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListFriendsRequest,
    ) -> Result<PaginatedResponse<crate::dto::response::FriendResponse>> {
        // TODO: 实现好友列表查询(游标分页)
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }

    /// 获取关注列表
    pub async fn list_following(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListFollowingRequest,
    ) -> Result<PaginatedResponse<UserResponse>> {
        // TODO: 实现关注列表查询(游标分页)
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }

    /// 获取粉丝列表
    pub async fn list_followers(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListFollowersRequest,
    ) -> Result<PaginatedResponse<UserResponse>> {
        // TODO: 实现粉丝列表查询(游标分页)
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }

    /// 获取拉黑列表
    pub async fn list_blocked(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListBlockedRequest,
    ) -> Result<PaginatedResponse<crate::dto::response::BlockedUserResponse>> {
        // TODO: 实现拉黑列表查询(游标分页)
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }

    /// 获取我的资料与聚合信息
    pub async fn get_me(&self, user_id: i64) -> Result<GetMeResponse> {
        // 1. 获取用户信息
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        // 2. 获取统计数据
        let (moments_count, friends_count, followers_count, following_count, groups_count) = 
            self.user_repo.get_user_stats(user_id).await?;
        
        let stats = GetMeStatsResponse {
            moments_count,
            friends_count,
            followers_count,
            following_count,
            groups_count,
        };

        // 3. 获取未读数
        let (messages_unread, _, _) = self.message_read_repo.get_unread_summary(user_id).await?;
        let (friend_requests, group_invites) = self.user_repo.get_unread_stats(user_id).await?;
        let notifications_unread = self.inbox_repo.get_unread_count(user_id).await? as i32;
        
        let unread = GetMeUnreadResponse {
            messages: messages_unread,
            friend_requests,
            group_invites,
            notifications: notifications_unread,
        };

        // 4. 获取设置
        let (privacy_level, location_sharing, allow_follow) = 
            self.user_repo.get_settings(user_id).await?;
        
        let settings = GetSettingsResponse {
            privacy_level,
            location_sharing,
            allow_follow,
        };

        // 5. 构造profile
        let profile = GetMeProfileResponse {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            email: None,    // TODO: 如果login_id是邮箱格式,解析出来
            phone: None,    // TODO: 如果login_id是手机号格式,解析出来
            created_at: user.created_at,
        };

        Ok(GetMeResponse {
            profile,
            stats,
            unread,
            settings,
        })
    }

    /// 获取用户设置
    pub async fn get_settings(&self, user_id: i64) -> Result<GetSettingsResponse> {
        let (privacy_level, location_sharing, allow_follow) = 
            self.user_repo.get_settings(user_id).await?;

        Ok(GetSettingsResponse {
            privacy_level,
            location_sharing,
            allow_follow,
        })
    }

    /// 更新用户设置
    pub async fn update_settings(
        &self,
        user_id: i64,
        req: &UpdateSettingsRequest,
    ) -> Result<crate::dto::response::UpdateSettingsResponse> {
        self.user_repo.update_settings(
            user_id,
            req.privacy_level.as_deref(),
            req.location_sharing,
            req.allow_follow,
        ).await?;

        Ok(crate::dto::response::UpdateSettingsResponse {
            success: true,
            message: "设置更新成功".to_string(),
        })
    }

    /// 升级账户(临时用户→正式用户)
    pub async fn upgrade_account(
        &self,
        user_id: i64,
        req: &UpgradeAccountRequest,
    ) -> Result<AuthResponse> {
        // 从配置中获取Token过期时间(TODO: 从AppState传递)
        let jwt_secret = "YOUR_SECRET_KEY"; // TODO: 从配置读取
        let access_token_expiration = 3600;
        let refresh_token_expiration = 2592000;
        // 1. 验证登录ID
        Validator::validate_not_empty(&req.login_id, "登录ID")?;
        
        // 2. 检查登录ID是否已被使用
        if self.user_repo.exists_by_login_id(&req.login_id).await? {
            return Err(AppError::Conflict("该登录ID已被注册".to_string()));
        }

        // 3. 对密码进行哈希
        let password_hash = crate::utils::password::PasswordUtil::hash_password(&req.password_hash)?;

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
            crate::utils::jwt::JwtUtil::generate_token(user_id, access_token_expiration, jwt_secret)?;
        let refresh_token =
            crate::utils::jwt::JwtUtil::generate_token(user_id, refresh_token_expiration, jwt_secret)?;

        // 7. 存储Refresh Token到Redis
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthResponse {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: chrono::Utc::now() + chrono::Duration::seconds(access_token_expiration),
        })
    }

    /// 删除账户
    pub async fn delete_account(&self, user_id: i64, req: &crate::dto::request::DeleteAccountRequest) -> Result<crate::dto::response::DeleteAccountResponse> {
        if req.confirmation != "DELETE_MY_ACCOUNT" {
            return Err(AppError::BadRequest("确认字符串不匹配".to_string()));
        }

        // 软删除用户
        self.user_repo.soft_delete(user_id).await?;

        // 删除Redis中的Token
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("DEL")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(crate::dto::response::DeleteAccountResponse {
            success: true,
            message: "账户已删除".to_string(),
        })
    }

    /// 获取用户关系信息
    async fn get_relationship_info(
        &self,
        from_user_id: i64,
        to_user_id: i64,
    ) -> Result<UserRelationshipInfo> {
        // 检查好友关系
        if let Some(friend_rel) = self
            .relationship_repo
            .find_mutual_relationship(from_user_id, to_user_id, RelationshipType::Friend)
            .await?
        {
            let status = if friend_rel.status == RelationshipStatus::Accepted {
                "friend"
            } else if friend_rel.from_user_id == from_user_id {
                "pending_sent"
            } else {
                "pending_received"
            };

            return Ok(UserRelationshipInfo {
                status: status.to_string(),
                can_message: friend_rel.status == RelationshipStatus::Accepted,
                can_view_moments: friend_rel.status == RelationshipStatus::Accepted,
            });
        }

        // 检查拉黑关系
        if self
            .relationship_repo
            .is_blocked(from_user_id, to_user_id)
            .await?
        {
            return Ok(UserRelationshipInfo {
                status: "blocked".to_string(),
                can_message: false,
                can_view_moments: false,
            });
        }

        if self
            .relationship_repo
            .is_blocked(to_user_id, from_user_id)
            .await?
        {
            return Ok(UserRelationshipInfo {
                status: "blocked_by".to_string(),
                can_message: false,
                can_view_moments: false,
            });
        }

        // 检查关注关系
        let i_follow = self
            .relationship_repo
            .find_relationship(from_user_id, to_user_id, RelationshipType::Following)
            .await?
            .is_some();

        let they_follow = self
            .relationship_repo
            .find_relationship(to_user_id, from_user_id, RelationshipType::Following)
            .await?
            .is_some();

        let status = match (i_follow, they_follow) {
            (true, true) => "following", // 互相关注
            (true, false) => "following",
            (false, true) => "follower",
            (false, false) => "none",
        };

        Ok(UserRelationshipInfo {
            status: status.to_string(),
            can_message: false,
            can_view_moments: i_follow,
        })
    }
}
