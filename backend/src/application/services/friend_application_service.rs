/// 好友应用服务
/// 
/// 处理好友关系相关的业务逻辑

use std::sync::Arc;
use tracing::{debug, info};
use sqlx::PgPool;

use crate::domain::{DomainResult, DomainError};
use crate::ports::repository_ports::{FriendRepositoryPort, UserRepositoryPort, FriendshipInfo, FriendshipStatus};

/// 好友应用服务
#[derive(Clone)]
pub struct FriendApplicationService<F, U>
where
    F: FriendRepositoryPort,
    U: UserRepositoryPort,
{
    friend_repository: Arc<F>,
    user_repository: Arc<U>,
    pool: PgPool,
}

impl<F, U> FriendApplicationService<F, U>
where
    F: FriendRepositoryPort,
    U: UserRepositoryPort,
{
    /// 创建新的好友应用服务
    pub fn new(friend_repository: Arc<F>, user_repository: Arc<U>, pool: PgPool) -> Self {
        Self {
            friend_repository,
            user_repository,
            pool,
        }
    }

    /// 发送好友请求
    pub async fn send_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<FriendshipInfo> {
        info!("用户{}向用户{}发送好友请求", requester_id, addressee_id);

        // 检查目标用户是否存在
        if self.user_repository.find_by_id(addressee_id).await?.is_none() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "目标用户不存在".to_string(),
            });
        }

        // 检查是否向自己发送请求
        if requester_id == addressee_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: "不能向自己发送好友请求".to_string(),
            });
        }

        // 发送好友请求
        let friendship = self.friend_repository.send_friend_request(requester_id, addressee_id).await?;
        
        info!("好友请求发送成功: {}", friendship.friendship_id);
        Ok(friendship)
    }

    /// 接受好友请求
    pub async fn accept_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()> {
        info!("用户{}接受来自用户{}的好友请求", addressee_id, requester_id);

        self.friend_repository.accept_friend_request(requester_id, addressee_id).await?;
        
        info!("好友请求接受成功: {} <-> {}", requester_id, addressee_id);
        Ok(())
    }

    /// 拒绝好友请求
    pub async fn reject_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()> {
        info!("用户{}拒绝来自用户{}的好友请求", addressee_id, requester_id);

        self.friend_repository.reject_friend_request(requester_id, addressee_id).await?;
        
        info!("好友请求拒绝成功: {} -> {}", requester_id, addressee_id);
        Ok(())
    }

    /// 检查用户是否有位置信息
    async fn check_user_has_location(&self, user_id: i64) -> DomainResult<bool> {
        let result = sqlx::query!(
            "SELECT EXISTS(SELECT 1 FROM user_locations WHERE user_id = $1) as \"has_location!\"",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| crate::domain::DomainError::BusinessRuleViolation { 
            rule: format!("Failed to check user location: {}", e) 
        })?;
        
        Ok(result.has_location)
    }

    /// 删除好友关系
    pub async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()> {
        info!("删除好友关系: {} <-> {}", user_id, friend_id);

        self.friend_repository.remove_friendship(user_id, friend_id).await?;
        
        info!("好友关系删除成功: {} <-> {}", user_id, friend_id);
        Ok(())
    }

    /// 获取用户的好友列表
    pub async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<(Vec<crate::application::queries::UserSearchResult>, bool, Option<i64>)> {
        debug!("获取用户{}的好友列表", user_id);

        // 获取好友ID列表
        let friend_ids = self.friend_repository.get_friends(user_id, cursor, limit + 1).await?;
        
        let has_more = friend_ids.len() > limit as usize;
        let actual_friends = if has_more {
            friend_ids.into_iter().take(limit as usize).collect::<Vec<_>>()
        } else {
            friend_ids
        };

        let next_cursor = if has_more && !actual_friends.is_empty() {
            Some(*actual_friends.last().unwrap())
        } else {
            None
        };

        // 获取好友详细信息
        let mut friends = Vec::new();
        for friend_id in actual_friends {
            if let Ok(Some(user)) = self.user_repository.find_by_id(friend_id).await {
                // 查询用户是否有位置信息
                let has_location = self.check_user_has_location(friend_id).await.unwrap_or(false);
                
                friends.push(crate::application::queries::UserSearchResult {
                    user_id: user.user_id,
                    nickname: user.nickname,
                    user_type: user.user_type.as_str().to_string(),
                    has_location,
                    created_at: user.created_at,
                });
            }
        }

        debug!("用户{}的好友列表获取成功，共{}个好友", user_id, friends.len());
        Ok((friends, has_more, next_cursor))
    }

    /// 获取收到的好友请求
    pub async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<(Vec<FriendRequestInfo>, bool, Option<i64>)> {
        debug!("获取用户{}收到的好友请求", user_id);

        let requests = self.friend_repository.get_received_friend_requests(user_id, cursor, limit + 1).await?;
        
        let has_more = requests.len() > limit as usize;
        let actual_requests = if has_more {
            requests.into_iter().take(limit as usize).collect::<Vec<_>>()
        } else {
            requests
        };

        let next_cursor = if has_more && !actual_requests.is_empty() {
            // 这里需要使用请求的ID作为游标，但FriendshipInfo没有ID字段
            // 暂时使用requester_id作为游标
            Some(actual_requests.last().unwrap().requester_id)
        } else {
            None
        };

        // 转换为包含用户信息的请求列表
        let mut friend_requests = Vec::new();
        for request in actual_requests {
            if let Ok(Some(requester)) = self.user_repository.find_by_id(request.requester_id).await {
                friend_requests.push(FriendRequestInfo {
                    friendship_id: request.friendship_id,
                    requester_id: request.requester_id,
                    requester_nickname: requester.nickname,
                    requester_avatar_url: requester.avatar_url,
                    status: request.status,
                    created_at: request.created_at,
                });
            }
        }

        debug!("用户{}收到的好友请求获取成功，共{}个请求", user_id, friend_requests.len());
        Ok((friend_requests, has_more, next_cursor))
    }

    /// 获取发送的好友请求
    pub async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<(Vec<FriendRequestInfo>, bool, Option<i64>)> {
        debug!("获取用户{}发送的好友请求", user_id);

        let requests = self.friend_repository.get_sent_friend_requests(user_id, cursor, limit + 1).await?;
        
        let has_more = requests.len() > limit as usize;
        let actual_requests = if has_more {
            requests.into_iter().take(limit as usize).collect::<Vec<_>>()
        } else {
            requests
        };

        let next_cursor = if has_more && !actual_requests.is_empty() {
            Some(actual_requests.last().unwrap().addressee_id)
        } else {
            None
        };

        // 转换为包含用户信息的请求列表
        let mut friend_requests = Vec::new();
        for request in actual_requests {
            if let Ok(Some(addressee)) = self.user_repository.find_by_id(request.addressee_id).await {
                friend_requests.push(FriendRequestInfo {
                    friendship_id: request.friendship_id,
                    requester_id: request.addressee_id, // 这里显示接收者信息
                    requester_nickname: addressee.nickname,
                    requester_avatar_url: addressee.avatar_url,
                    status: request.status,
                    created_at: request.created_at,
                });
            }
        }

        debug!("用户{}发送的好友请求获取成功，共{}个请求", user_id, friend_requests.len());
        Ok((friend_requests, has_more, next_cursor))
    }

    /// 检查两个用户是否是好友
    pub async fn are_friends(&self, user_id1: i64, user_id2: i64) -> DomainResult<bool> {
        debug!("检查用户{}和{}是否是好友", user_id1, user_id2);

        let are_friends = self.friend_repository.are_friends(user_id1, user_id2).await?;
        
        debug!("用户{}和{}的好友关系: {}", user_id1, user_id2, are_friends);
        Ok(are_friends)
    }

    /// 获取待处理的好友请求数量
    pub async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取用户{}的待处理好友请求数量", user_id);

        let count = self.friend_repository.get_pending_requests_count(user_id).await?;
        
        debug!("用户{}有{}个待处理的好友请求", user_id, count);
        Ok(count)
    }
}

/// 好友请求信息（包含用户详细信息）
#[derive(Debug, Clone)]
pub struct FriendRequestInfo {
    pub friendship_id: String,
    pub requester_id: i64,
    pub requester_nickname: String,
    pub requester_avatar_url: Option<String>,
    pub status: FriendshipStatus,
    pub created_at: chrono::DateTime<chrono::Utc>,
}
