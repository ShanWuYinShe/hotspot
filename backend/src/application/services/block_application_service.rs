/// 屏蔽应用服务
/// 
/// 功能职责：
/// - 实现屏蔽相关的业务逻辑
/// - 协调屏蔽仓储和其他服务
/// - 处理屏蔽操作的权限验证和业务规则
/// - 提供屏蔽设置管理功能
use async_trait::async_trait;
use std::sync::Arc;

use crate::{
    domain::{DomainResult, Block, BlockSettings, BlockStats, BlockType, BlockReason},
    ports::repository_ports::{BlockRepository, UserRepository, FriendRepository},
};

/// 屏蔽应用服务接口
#[async_trait]
pub trait BlockApplicationService: Send + Sync {
    /// 屏蔽用户
    async fn block_user(
        &self,
        blocker_id: i64,
        blocked_id: i64,
        block_type: BlockType,
        reason: BlockReason,
    ) -> DomainResult<Block>;

    /// 取消屏蔽用户
    async fn unblock_user(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<()>;

    /// 检查是否被屏蔽
    async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<bool>;

    /// 获取屏蔽关系详情
    async fn get_block_details(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<Option<Block>>;

    /// 获取用户屏蔽的用户列表
    async fn get_blocked_users(&self, blocker_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>>;

    /// 获取屏蔽当前用户的用户列表
    async fn get_blocking_users(&self, blocked_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>>;

    /// 获取用户屏蔽设置
    async fn get_block_settings(&self, user_id: i64) -> DomainResult<Option<BlockSettings>>;

    /// 更新用户屏蔽设置
    async fn update_block_settings(&self, settings: &BlockSettings) -> DomainResult<BlockSettings>;

    /// 获取屏蔽统计信息
    async fn get_block_stats(&self, user_id: i64) -> DomainResult<BlockStats>;

    /// 批量检查屏蔽状态
    async fn batch_check_blocked(&self, blocker_id: i64, user_ids: &[i64]) -> DomainResult<Vec<i64>>;

    /// 检查消息是否应被屏蔽
    async fn should_block_message(&self, sender_id: i64, receiver_id: i64, content: &str) -> DomainResult<bool>;
}

/// 屏蔽应用服务实现
pub struct BlockApplicationServiceImpl<B, U, F>
where
    B: BlockRepository,
    U: UserRepository,
    F: FriendRepository,
{
    block_repository: Arc<B>,
    user_repository: Arc<U>,
    friend_repository: Arc<F>,
}

impl<B, U, F> BlockApplicationServiceImpl<B, U, F>
where
    B: BlockRepository,
    U: UserRepository,
    F: FriendRepository,
{
    /// 创建新的屏蔽应用服务实例
    pub fn new(
        block_repository: Arc<B>,
        user_repository: Arc<U>,
        friend_repository: Arc<F>,
    ) -> Self {
        Self {
            block_repository,
            user_repository,
            friend_repository,
        }
    }

    /// 验证用户是否存在
    async fn validate_user_exists(&self, user_id: i64) -> DomainResult<()> {
        let user = self.user_repository.find_user_profile(user_id).await?;
        if user.is_none() {
            return Err(crate::domain::DomainError::BusinessRuleViolation { rule: "用户不存在".to_string() });
        }
        Ok(())
    }

    /// 验证屏蔽操作的业务规则
    async fn validate_block_operation(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<()> {
        // 不能屏蔽自己
        if blocker_id == blocked_id {
            return Err(crate::domain::DomainError::BusinessRuleViolation { rule: "不能屏蔽自己".to_string() });
        }

        // 验证目标用户存在
        self.validate_user_exists(blocked_id).await?;

        Ok(())
    }

    /// 检查关键词过滤
    fn contains_blocked_keywords(&self, content: &str, keywords: &[String]) -> bool {
        let content_lower = content.to_lowercase();
        keywords.iter().any(|keyword| {
            let keyword_lower = keyword.to_lowercase();
            content_lower.contains(&keyword_lower)
        })
    }
}

#[async_trait]
impl<B, U, F> BlockApplicationService for BlockApplicationServiceImpl<B, U, F>
where
    B: BlockRepository,
    U: UserRepository,
    F: FriendRepository,
{
    /// 屏蔽用户
    async fn block_user(
        &self,
        blocker_id: i64,
        blocked_id: i64,
        block_type: BlockType,
        reason: BlockReason,
    ) -> DomainResult<Block> {
        // 验证屏蔽操作的有效性
        self.validate_block_operation(blocker_id, blocked_id).await?;

        // 屏蔽用户
        let block = self.block_repository.block_user(blocker_id, blocked_id, block_type, reason).await?;

        // TODO: 发送通知或记录日志
        // self.audit_service.log_block_action(blocker_id, blocked_id, &block).await?;

        Ok(block)
    }

    /// 取消屏蔽用户
    async fn unblock_user(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<()> {
        // 验证用户存在
        self.validate_user_exists(blocked_id).await?;

        // 取消屏蔽
        self.block_repository.unblock_user(blocker_id, blocked_id).await?;

        // TODO: 发送通知或记录日志
        // self.audit_service.log_unblock_action(blocker_id, blocked_id).await?;

        Ok(())
    }

    /// 检查是否被屏蔽
    async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<bool> {
        self.block_repository.is_blocked(blocker_id, blocked_id).await
    }

    /// 获取屏蔽关系详情
    async fn get_block_details(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<Option<Block>> {
        self.block_repository.get_block_details(blocker_id, blocked_id).await
    }

    /// 获取用户屏蔽的用户列表
    async fn get_blocked_users(&self, blocker_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>> {
        // 验证分页参数
        if limit <= 0 || limit > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "分页限制必须在1-100之间".to_string()
            });
        }

        self.block_repository.get_blocked_users(blocker_id, cursor, limit).await
    }

    /// 获取屏蔽当前用户的用户列表
    async fn get_blocking_users(&self, blocked_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>> {
        // 验证分页参数
        if limit <= 0 || limit > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "分页限制必须在1-100之间".to_string()
            });
        }

        self.block_repository.get_blocking_users(blocked_id, cursor, limit).await
    }

    /// 获取用户屏蔽设置
    async fn get_block_settings(&self, user_id: i64) -> DomainResult<Option<BlockSettings>> {
        self.block_repository.get_block_settings(user_id).await
    }

    /// 更新用户屏蔽设置
    async fn update_block_settings(&self, settings: &BlockSettings) -> DomainResult<BlockSettings> {
        // 验证设置的有效性
        if settings.blocked_keywords.len() > 50 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "关键词过滤器数量不能超过50个".to_string()
            });
        }

        for keyword in &settings.blocked_keywords {
            if keyword.trim().is_empty() || keyword.len() > 50 {
                return Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: "关键词长度必须在1-50字符之间".to_string()
                });
            }
        }

        self.block_repository.update_block_settings(settings).await
    }

    /// 获取屏蔽统计信息
    async fn get_block_stats(&self, user_id: i64) -> DomainResult<BlockStats> {
        self.block_repository.get_block_stats(user_id).await
    }

    /// 批量检查屏蔽状态
    async fn batch_check_blocked(&self, blocker_id: i64, user_ids: &[i64]) -> DomainResult<Vec<i64>> {
        if user_ids.len() > 100 {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: "批量检查的用户数量不能超过100个".to_string()
            });
        }

        self.block_repository.batch_check_blocked(blocker_id, user_ids).await
    }

    /// 检查消息是否应被屏蔽
    async fn should_block_message(&self, sender_id: i64, receiver_id: i64, content: &str) -> DomainResult<bool> {
        // 检查直接屏蔽关系
        let is_blocked = self.block_repository.is_blocked(receiver_id, sender_id).await?;
        if is_blocked {
            return Ok(true);
        }

        // 获取接收者的屏蔽设置
        if let Some(settings) = self.block_repository.get_block_settings(receiver_id).await? {
            // 检查是否屏蔽未知用户
            if settings.block_stranger_messages {
                let friendship_status = self.friend_repository.check_friendship(receiver_id, sender_id).await?;
                if !matches!(friendship_status, crate::domain::FriendshipStatus::Friends) {
                    return Ok(true);
                }
            }

            // 检查关键词过滤
            if self.contains_blocked_keywords(content, &settings.blocked_keywords) {
                return Ok(true);
            }
        }

        Ok(false)
    }
}
