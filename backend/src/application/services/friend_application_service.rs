// 好友应用服务
// 
// 功能职责：
// - 实现好友相关的业务逻辑
// - 协调好友仓储和其他服务
// - 处理好友请求的权限验证和业务规则
// - 提供好友关系管理功能
// 暂时禁用，等待重新设计
/*
use async_trait::async_trait;
use std::sync::Arc;

use crate::{
    domain::{DomainResult, DomainError, FriendRequest, Friendship, FriendProfile, FriendshipStatus},
    ports::repository_ports::{FriendRepository, UserRepository},
};
*/

// 暂时禁用，等待重新设计
/*
/// 好友应用服务接口
#[async_trait]
pub trait FriendApplicationService: Send + Sync {
    /// 发送好友请求
    async fn send_friend_request(
        &self,
        requester_id: i64,
        target_id: i64,
        message: Option<&str>,
    ) -> DomainResult<FriendRequest>;

    /// 接受好友请求
    async fn accept_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<Friendship>;

    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()>;

    /// 移除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()>;

    /// 获取用户的好友列表
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>>;

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>>;

    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>>;

    /// 检查好友关系状态
    async fn check_friendship(&self, user_id: i64, other_user_id: i64) -> DomainResult<FriendshipStatus>;

    /// 获取待处理的好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64>;
}

/// 好友应用服务实现
#[derive(Clone)]
pub struct FriendApplicationServiceImpl<F, U>
where
    F: FriendRepository,
    U: UserRepository + 'static,
{
    base: super::BaseApplicationService,
    friend_repository: Arc<F>,
    #[allow(dead_code)]
    user_repository: Arc<U>,
}

impl<F, U> FriendApplicationServiceImpl<F, U>
where
    F: FriendRepository,
    U: UserRepository + 'static,
{
    /// 创建新的好友应用服务实例
    pub fn new(
        friend_repository: Arc<F>, 
        user_repository: Arc<U>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        Self {
            base: super::BaseApplicationService::new("FriendApplicationService")
                .with_user_repository(user_repository.clone() as Arc<dyn UserRepository>)
                .with_event_publisher(event_publisher),
            friend_repository,
            user_repository,
        }
    }

    /// 验证用户是否存在（使用基类方法）
    async fn validate_user_exists(&self, user_id: i64) -> DomainResult<()> {
        self.base.validate_user_exists(user_id).await
    }

    /// 验证好友请求的业务规则
    async fn validate_friend_request(&self, requester_id: i64, target_id: i64) -> DomainResult<()> {
        // 不能向自己发送好友请求
        if requester_id == target_id {
            return Err(DomainError::BusinessRuleViolation { rule: "不能向自己发送好友请求".to_string() });
        }

        // 验证目标用户存在
        self.validate_user_exists(target_id).await?;

        // 检查当前好友关系状态
        let status = self.friend_repository.check_friendship(requester_id, target_id).await?;
        match status {
            FriendshipStatus::Friends => {
                return Err(DomainError::BusinessRuleViolation { rule: "已经是好友关系".to_string() });
            }
            FriendshipStatus::RequestSent => {
                return Err(DomainError::BusinessRuleViolation { rule: "已发送好友请求".to_string() });
            }
            FriendshipStatus::RequestReceived => {
                return Err(DomainError::BusinessRuleViolation { rule: "对方已向您发送好友请求".to_string() });
            }
            FriendshipStatus::Blocked => {
                return Err(DomainError::BusinessRuleViolation { rule: "无法向该用户发送好友请求".to_string() });
            }
            FriendshipStatus::NotFriends => {
                // 可以发送好友请求
            }
        }

        Ok(())
    }
}

#[async_trait]
impl<F, U> FriendApplicationService for FriendApplicationServiceImpl<F, U>
where
    F: FriendRepository,
    U: UserRepository + 'static,
{
    /// 发送好友请求
    async fn send_friend_request(
        &self,
        requester_id: i64,
        target_id: i64,
        message: Option<&str>,
    ) -> DomainResult<FriendRequest> {
        // 验证请求的有效性
        self.validate_friend_request(requester_id, target_id).await?;

        // 发送好友请求
        let request = self.friend_repository.send_friend_request(requester_id, target_id, message).await?;

        // TODO: 发送通知给目标用户
        // self.notification_service.send_friend_request_notification(target_id, &request).await?;

        Ok(request)
    }

    /// 接受好友请求
    async fn accept_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<Friendship> {
        // 接受好友请求并创建好友关系
        let friendship = self.friend_repository.accept_friend_request(request_id, target_id).await?;

        // TODO: 发送通知给请求者
        // self.notification_service.send_friend_request_accepted_notification(friendship.user_id_1, &friendship).await?;

        Ok(friendship)
    }

    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()> {
        // 拒绝好友请求
        self.friend_repository.reject_friend_request(request_id, target_id).await?;

        // TODO: 可选择性地发送通知给请求者
        // self.notification_service.send_friend_request_rejected_notification(...).await?;

        Ok(())
    }

    /// 移除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()> {
        // 验证用户存在
        self.validate_user_exists(friend_id).await?;

        // 检查是否存在好友关系
        let status = self.friend_repository.check_friendship(user_id, friend_id).await?;
        if !matches!(status, FriendshipStatus::Friends) {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "好友关系不存在".to_string()
            });
        }

        // 移除好友关系
        self.friend_repository.remove_friendship(user_id, friend_id).await?;

        // TODO: 发送通知给对方
        // self.notification_service.send_friendship_removed_notification(friend_id, user_id).await?;

        Ok(())
    }

    /// 获取用户的好友列表
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        // 验证分页参数
        if limit <= 0 || limit > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "分页限制必须在1-100之间".to_string()
            });
        }

        // 获取好友列表
        self.friend_repository.get_friends(user_id, cursor, limit).await
    }

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        // 验证分页参数
        if limit <= 0 || limit > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "分页限制必须在1-100之间".to_string()
            });
        }

        // 获取收到的好友请求
        self.friend_repository.get_received_friend_requests(user_id, cursor, limit).await
    }

    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        // 验证分页参数
        if limit <= 0 || limit > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "分页限制必须在1-100之间".to_string()
            });
        }

        // 获取发送的好友请求
        self.friend_repository.get_sent_friend_requests(user_id, cursor, limit).await
    }

    /// 检查好友关系状态
    async fn check_friendship(&self, user_id: i64, other_user_id: i64) -> DomainResult<FriendshipStatus> {
        // 验证用户存在
        self.validate_user_exists(other_user_id).await?;

        // 检查好友关系状态
        self.friend_repository.check_friendship(user_id, other_user_id).await
    }

    /// 获取待处理的好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        self.friend_repository.get_pending_requests_count(user_id).await
    }
}
*/
