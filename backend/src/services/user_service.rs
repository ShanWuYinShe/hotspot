use redis::aio::ConnectionManager;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            AddFriendRequest, BlockRequest, SearchUserByIdRequest, SearchUserByLocationRequest,
            SearchUserByNameRequest, SetFriendRemarkRequest, UpdateLocationRequest,
            UpdateProfileRequest,
        },
        response::{
            FriendResponse, NearbyUserResponse, PaginatedResponse, UserProfileResponse,
            UserRelationshipInfo, UserResponse,
        },
    },
    error::{AppError, Result},
    repositories::{user_relationship_repo::UserRelationshipRepository, user_repo::UserRepository},
    utils::{geo::GeoUtil, validation::Validator},
};

pub struct UserService {
    user_repo: UserRepository,
    relationship_repo: UserRelationshipRepository,
    redis: ConnectionManager,
}

impl UserService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            user_repo: UserRepository::new(pool.clone()),
            relationship_repo: UserRelationshipRepository::new(pool.clone()),
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
        let cursor = cursor.and_then(|c| c.parse::<i64>().ok());

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
            .ok_or(AppError::NotFound)?;

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

        Ok(UserProfileResponse {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            avatar_url: user.avatar_url,
            bio: user.bio,
            location_name: user.location_name,
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
        let radius = Validator::validate_radius(req.radius, 50000)?;
        let limit = Validator::validate_pagination(req.limit, 100)?;
        let cursor = req.cursor.as_ref().and_then(|c| c.parse::<i64>().ok());

        let (users, next_cursor) = self
            .user_repo
            .search_by_location(req.latitude, req.longitude, radius, limit, cursor)
            .await?;

        let items = users
            .into_iter()
            .map(|u| {
                // 计算距离（从数据库查询结果中获取）
                let distance = if let (Some(lat), Some(lon)) = (u.latitude, u.longitude) {
                    // PostGIS 已经计算了距离，这里只是示例
                    // 实际应该从查询结果中获取
                    0.0
                } else {
                    0.0
                };

                NearbyUserResponse {
                    user_id: u.user_id.to_string(),
                    nickname: u.nickname,
                    avatar_url: u.avatar_url,
                    location_name: u.location_name,
                    distance,
                    is_online: None,
                }
            })
            .collect();

        Ok(PaginatedResponse {
            items,
            has_more: next_cursor.is_some(),
            next_cursor,
        })
    }

    /// 更新用户位置
    pub async fn update_location(&self, user_id: i64, req: &UpdateLocationRequest) -> Result<()> {
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;

        self.user_repo
            .update_location(
                user_id,
                req.latitude,
                req.longitude,
                req.location_name.clone(),
            )
            .await?;

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
            .ok_or(AppError::NotFound)?;

        // 检查是否已经是好友或有待处理的申请
        if let Some(existing) = self
            .relationship_repo
            .find_mutual_relationship(from_user_id, to_user_id, "friend")
            .await?
        {
            if existing.status == "accepted" {
                return Err(AppError::Conflict("已经是好友".to_string()));
            } else if existing.status == "pending" {
                // 如果对方也向我发送了申请，自动接受
                if existing.to_user_id == from_user_id {
                    self.relationship_repo
                        .update_status(existing.relationship_id, "accepted")
                        .await?;
                    return Ok((existing.relationship_id.to_string(), "accepted".to_string()));
                }
                return Err(AppError::Conflict("已发送好友申请".to_string()));
            }
        }

        // 创建好友申请
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, "friend", "pending")
            .await?;

        Ok((relationship_id.to_string(), "pending".to_string()))
    }

    /// 接受好友申请
    pub async fn accept_friend(&self, user_id: i64, relationship_id: i64) -> Result<()> {
        // 查找关系
        let relationship = self
            .relationship_repo
            .find_relationship(relationship_id, user_id, "friend")
            .await?
            .ok_or(AppError::NotFound)?;

        if relationship.to_user_id != user_id {
            return Err(AppError::Forbidden);
        }

        if relationship.status != "pending" {
            return Err(AppError::BadRequest("该申请已处理".to_string()));
        }

        // 更新状态
        self.relationship_repo
            .update_status(relationship_id, "accepted")
            .await?;

        Ok(())
    }

    /// 拒绝好友申请
    pub async fn reject_friend(&self, user_id: i64, relationship_id: i64) -> Result<()> {
        // 查找关系
        let relationship = self
            .relationship_repo
            .find_relationship(relationship_id, user_id, "friend")
            .await?
            .ok_or(AppError::NotFound)?;

        if relationship.to_user_id != user_id {
            return Err(AppError::Forbidden);
        }

        // 删除申请
        self.relationship_repo.delete(relationship_id).await?;

        Ok(())
    }

    /// 删除好友
    pub async fn remove_friend(&self, user_id: i64, friend_id: i64) -> Result<()> {
        // 删除双向关系
        self.relationship_repo
            .delete_by_users(user_id, friend_id, "friend")
            .await?;
        self.relationship_repo
            .delete_by_users(friend_id, user_id, "friend")
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
            .find_relationship(from_user_id, to_user_id, "following")
            .await?
        {
            return Err(AppError::Conflict("已关注该用户".to_string()));
        }

        // 创建关注关系
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, "following", "accepted")
            .await?;

        Ok(relationship_id.to_string())
    }

    /// 取消关注
    pub async fn unfollow(&self, from_user_id: i64, to_user_id: i64) -> Result<()> {
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, "following")
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
            .find_relationship(from_user_id, to_user_id, "blocked")
            .await?
        {
            return Err(AppError::Conflict("已拉黑该用户".to_string()));
        }

        // 删除好友和关注关系
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, "friend")
            .await
            .ok();
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, "following")
            .await
            .ok();

        // 创建拉黑关系
        let relationship_id = self
            .relationship_repo
            .create(from_user_id, to_user_id, "blocked", "accepted")
            .await?;

        Ok(relationship_id.to_string())
    }

    /// 解除拉黑
    pub async fn unblock(&self, from_user_id: i64, to_user_id: i64) -> Result<()> {
        self.relationship_repo
            .delete_by_users(from_user_id, to_user_id, "blocked")
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
            .find_mutual_relationship(user_id, friend_id, "friend")
            .await?
            .ok_or(AppError::NotFound)?;

        if relationship.status != "accepted" {
            return Err(AppError::BadRequest("不是好友关系".to_string()));
        }

        // 更新备注（只更新自己这一侧的）
        let my_relationship = if relationship.from_user_id == user_id {
            relationship
        } else {
            self.relationship_repo
                .find_relationship(user_id, friend_id, "friend")
                .await?
                .ok_or(AppError::NotFound)?
        };

        self.relationship_repo
            .set_remark(my_relationship.relationship_id, &req.remark)
            .await?;

        Ok(())
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
            .find_mutual_relationship(from_user_id, to_user_id, "friend")
            .await?
        {
            let status = if friend_rel.status == "accepted" {
                "friend"
            } else if friend_rel.from_user_id == from_user_id {
                "pending_sent"
            } else {
                "pending_received"
            };

            return Ok(UserRelationshipInfo {
                status: status.to_string(),
                can_message: friend_rel.status == "accepted",
                can_view_moments: friend_rel.status == "accepted",
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
            .find_relationship(from_user_id, to_user_id, "following")
            .await?
            .is_some();

        let they_follow = self
            .relationship_repo
            .find_relationship(to_user_id, from_user_id, "following")
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
