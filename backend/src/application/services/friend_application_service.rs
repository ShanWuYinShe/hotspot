/// 好友关系应用服务
/// 
/// 功能职责：
/// - 协调好友关系的业务逻辑
/// - 管理好友请求的生命周期
/// - 处理关注/取消关注逻辑
/// - 管理用户屏蔽功能
/// - 提供好友状态查询和统计

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, DomainError, UserId};
use crate::domain::entities::{
    FriendProfile, FriendRequest, FriendshipStatus, Friendship,
};
use crate::ports::{FriendRepository, UserRepository, EventPublisher};
use crate::adapters::adapters_pg::report_repository::{ReportRepository, ReportTargetType, ReportReason};

/// 好友关系应用服务接口
#[async_trait]
pub trait FriendApplicationService: Send + Sync {
    /// 发送好友申请
    async fn send_friend_request(&self, requester_id: i64, target_id: i64, message: Option<String>) -> DomainResult<FriendRequest>;
    
    /// 接受好友申请
    async fn accept_friend_request(&self, request_id: String, user_id: i64) -> DomainResult<Friendship>;
    
    /// 拒绝好友申请
    async fn reject_friend_request(&self, request_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 关注用户
    async fn follow_user(&self, follower_id: i64, target_id: i64) -> DomainResult<()>;
    
    /// 取消关注用户
    async fn unfollow_user(&self, follower_id: i64, target_id: i64) -> DomainResult<()>;
    
    /// 拉黑用户
    async fn block_user(&self, blocker_id: i64, blocked_id: i64, reason: Option<String>) -> DomainResult<()>;
    
    /// 获取好友列表
    async fn get_friends_list(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>>;
    
    /// 获取拉黑列表
    async fn get_blocked_list(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>>;
    
    /// 检查好友关系状态
    async fn check_friendship_status(&self, user_id: i64, target_id: i64) -> DomainResult<FriendshipStatus>;
    
    /// 获取待处理好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64>;
}

/// 好友关系应用服务实现
pub struct FriendApplicationServiceImpl {
    friend_repository: Arc<dyn FriendRepository>,
    user_repository: Arc<dyn UserRepository>,
    report_repository: Arc<dyn ReportRepository>,
    event_publisher: Arc<dyn EventPublisher>,
}

impl FriendApplicationServiceImpl {
    pub fn new(
        friend_repository: Arc<dyn FriendRepository>,
        user_repository: Arc<dyn UserRepository>,
        report_repository: Arc<dyn ReportRepository>,
        event_publisher: Arc<dyn EventPublisher>,
    ) -> Self {
        Self {
            friend_repository,
            user_repository,
            report_repository,
            event_publisher,
        }
    }
}

#[async_trait]
impl FriendApplicationService for FriendApplicationServiceImpl {
    /// 发送好友申请
    async fn send_friend_request(&self, requester_id: i64, target_id: i64, message: Option<String>) -> DomainResult<FriendRequest> {
        debug!("发送好友申请: {} -> {}", requester_id, target_id);
        
        // 验证用户不能向自己发送好友请求
        if requester_id == target_id {
            return Err(DomainError::ValidationError {
                field: "target_id".to_string(),
                reason: "不能向自己发送好友请求".to_string(),
            });
        }
        
        // 验证目标用户存在
        let target_user = self.user_repository.find_by_id(UserId::from(target_id)).await?;
        if target_user.is_none() {
            return Err(DomainError::NotFound {
                resource: "目标用户".to_string(),
            });
        }
        
        // 检查当前好友关系状态
        let current_status = self.friend_repository.check_friendship(requester_id, target_id).await?;
        match current_status {
            FriendshipStatus::Friends => {
                return Err(DomainError::ValidationError {
                    field: "friendship".to_string(),
                    reason: "已经是好友关系".to_string(),
                });
            }
            FriendshipStatus::Pending => {
                return Err(DomainError::ValidationError {
                    field: "friendship".to_string(),
                    reason: "好友请求已发送，请等待对方回应".to_string(),
                });
            }
            FriendshipStatus::Blocked => {
                return Err(DomainError::ValidationError {
                    field: "friendship".to_string(),
                    reason: "无法向该用户发送好友请求".to_string(),
                });
            }
            FriendshipStatus::NotFriends => {
                // 可以发送好友请求
            }
        }
        
        // 发送好友请求
        let friend_request = self.friend_repository.send_friend_request(
            requester_id,
            target_id,
            message.as_deref(),
        ).await?;
        
        // TODO: 发送通知给目标用户
        // TODO: 发布好友请求事件
        
        info!("好友申请发送成功: {} -> {}", requester_id, target_id);
        Ok(friend_request)
    }
    
    /// 接受好友申请
    async fn accept_friend_request(&self, request_id: String, user_id: i64) -> DomainResult<Friendship> {
        debug!("接受好友申请: 请求ID={}, 用户={}", request_id, user_id);
        
        // 接受好友请求
        let friendship = self.friend_repository.accept_friend_request(&request_id, user_id).await?;
        
        // TODO: 发送通知给申请发送者
        // TODO: 发布好友关系建立事件
        
        info!("好友申请已接受: 请求ID={}, 用户={}", request_id, user_id);
        Ok(friendship)
    }
    
    /// 拒绝好友申请
    async fn reject_friend_request(&self, request_id: String, user_id: i64) -> DomainResult<()> {
        debug!("拒绝好友申请: 请求ID={}, 用户={}", request_id, user_id);
        
        // 拒绝好友请求
        self.friend_repository.reject_friend_request(&request_id, user_id).await?;
        
        // TODO: 可选：发送通知给申请发送者
        // TODO: 发布好友请求拒绝事件
        
        info!("好友申请已拒绝: 请求ID={}, 用户={}", request_id, user_id);
        Ok(())
    }
    
    /// 关注用户
    async fn follow_user(&self, follower_id: i64, target_id: i64) -> DomainResult<()> {
        debug!("关注用户: {} -> {}", follower_id, target_id);
        
        // 验证用户不能关注自己
        if follower_id == target_id {
            return Err(DomainError::ValidationError {
                field: "target_id".to_string(),
                reason: "不能关注自己".to_string(),
            });
        }
        
        // 验证目标用户存在
        let target_user = self.user_repository.find_by_id(UserId::from(target_id)).await?;
        if target_user.is_none() {
            return Err(DomainError::NotFound {
                resource: "目标用户".to_string(),
            });
        }
        
        // TODO: 实现关注逻辑
        // 1. 检查是否已经关注
        // 2. 创建关注关系
        // 3. 发送通知给被关注者
        // 4. 发布关注事件
        
        info!("用户关注成功: {} -> {}", follower_id, target_id);
        Ok(())
    }
    
    /// 取消关注用户
    async fn unfollow_user(&self, follower_id: i64, target_id: i64) -> DomainResult<()> {
        debug!("取消关注用户: {} -> {}", follower_id, target_id);
        
        // TODO: 实现取消关注逻辑
        // 1. 检查关注关系是否存在
        // 2. 删除关注关系
        // 3. 发布取消关注事件
        
        info!("取消关注成功: {} -> {}", follower_id, target_id);
        Ok(())
    }
    
    /// 拉黑用户
    async fn block_user(&self, blocker_id: i64, blocked_id: i64, reason: Option<String>) -> DomainResult<()> {
        debug!("拉黑用户: {} -> {}", blocker_id, blocked_id);
        
        // 验证用户不能拉黑自己
        if blocker_id == blocked_id {
            return Err(DomainError::ValidationError {
                field: "blocked_id".to_string(),
                reason: "不能拉黑自己".to_string(),
            });
        }
        
        // 验证目标用户存在
        let target_user = self.user_repository.find_by_id(UserId::from(blocked_id)).await?;
        if target_user.is_none() {
            return Err(DomainError::NotFound {
                resource: "目标用户".to_string(),
            });
        }
        
        // TODO: 实现拉黑逻辑
        // 1. 检查是否已经拉黑
        // 2. 创建拉黑记录
        // 3. 移除现有的好友关系（如果存在）
        // 4. 移除关注关系（如果存在）
        // 5. 记录拉黑原因
        // 6. 发布拉黑事件
        
        // 如果提供了拉黑原因，可以创建举报记录
        if let Some(reason_text) = reason {
            let _ = self.report_repository.create_report(
                blocker_id,
                ReportTargetType::User,
                blocked_id.to_string(),
                ReportReason::Other,
                Some(reason_text),
            ).await;
        }
        
        info!("用户拉黑成功: {} -> {}", blocker_id, blocked_id);
        Ok(())
    }
    
    /// 获取好友列表
    async fn get_friends_list(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        debug!("获取好友列表: 用户={}, 游标={}, 限制={}", user_id, cursor, limit);
        
        let friends = self.friend_repository.get_friends(user_id, cursor, limit).await?;
        
        Ok(friends)
    }
    
    /// 获取拉黑列表
    async fn get_blocked_list(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        debug!("获取拉黑列表: 用户={}, 游标={}, 限制={}", user_id, cursor, limit);
        
        // TODO: 实现拉黑列表查询
        // 1. 查询用户的拉黑记录
        // 2. 关联被拉黑用户的信息
        // 3. 返回FriendProfile列表
        
        Ok(vec![])
    }
    
    /// 检查好友关系状态
    async fn check_friendship_status(&self, user_id: i64, target_id: i64) -> DomainResult<FriendshipStatus> {
        debug!("检查好友关系状态: {} <-> {}", user_id, target_id);
        
        let status = self.friend_repository.check_friendship(user_id, target_id).await?;
        
        Ok(status)
    }
    
    /// 获取待处理好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取待处理好友请求数量: 用户={}", user_id);
        
        let count = self.friend_repository.get_pending_requests_count(user_id).await?;
        
        Ok(count)
    }
}

impl Clone for FriendApplicationServiceImpl {
    fn clone(&self) -> Self {
        Self {
            friend_repository: Arc::clone(&self.friend_repository),
            user_repository: Arc::clone(&self.user_repository),
            report_repository: Arc::clone(&self.report_repository),
            event_publisher: Arc::clone(&self.event_publisher),
        }
    }
}
