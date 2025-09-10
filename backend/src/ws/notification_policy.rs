/// WebSocket通知策略服务
/// 
/// 实现智能通知策略，避免全局广播，提供精准的用户定向通知
use crate::domain::DomainResult;
use crate::ws::{
    message_types::ServerMessage,
    message_router::MessageRouter,
};
use std::sync::Arc;
use std::collections::HashSet;
use serde::{Serialize, Deserialize};
use tracing::{debug, info, warn};
use uuid::Uuid;

/// 通知策略类型
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum NotificationPolicy {
    /// 好友关系通知：只向好友发送状态更新
    FriendNotification,
    /// 群组成员通知：向共同群组成员发送通知
    GroupMemberNotification,
    /// 地理位置通知：向附近用户发送通知（可配置距离半径）
    NearbyNotification { radius_km: f64 },
    /// 智能组合：支持多种策略同时启用，自动去重
    CombinedPolicy(Vec<NotificationPolicy>),
}

/// 通知策略配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationPolicyConfig {
    /// 启用好友通知
    pub enable_friend_notifications: bool,
    /// 启用群组通知
    pub enable_group_notifications: bool,
    /// 启用附近通知
    pub enable_nearby_notifications: bool,
    /// 附近通知半径（公里）
    pub nearby_radius_km: f64,
    /// 启用全局广播（生产环境建议为false）
    pub enable_global_broadcast: bool,
}

impl Default for NotificationPolicyConfig {
    fn default() -> Self {
        Self {
            enable_friend_notifications: true,
            enable_group_notifications: true,
            enable_nearby_notifications: false,
            nearby_radius_km: 5.0,
            enable_global_broadcast: false,
        }
    }
}

/// 通知目标用户查询接口
#[async_trait::async_trait]
pub trait NotificationTargetResolver: Send + Sync {
    /// 获取用户的好友列表
    async fn get_user_friends(&self, user_id: i64) -> DomainResult<Vec<i64>>;
    
    /// 获取用户所在的群组成员
    async fn get_user_group_members(&self, user_id: i64) -> DomainResult<Vec<i64>>;
    
    /// 获取附近的用户
    async fn get_nearby_users(
        &self, 
        user_id: i64, 
        radius_km: f64
    ) -> DomainResult<Vec<i64>>;
    
    /// 获取群组的所有成员
    async fn get_group_members(&self, group_id: Uuid) -> DomainResult<Vec<i64>>;
}

/// 通知策略服务
pub struct NotificationPolicyService {
    message_router: Arc<MessageRouter>,
    target_resolver: Arc<dyn NotificationTargetResolver>,
    config: NotificationPolicyConfig,
}

impl NotificationPolicyService {
    /// 创建新的通知策略服务
    pub fn new(
        message_router: Arc<MessageRouter>,
        target_resolver: Arc<dyn NotificationTargetResolver>,
        config: NotificationPolicyConfig,
    ) -> Self {
        Self {
            message_router,
            target_resolver,
            config,
        }
    }
    
    /// 根据策略发送通知
    pub async fn send_notification_with_policy(
        &self,
        sender_id: i64,
        message: ServerMessage,
        policy: NotificationPolicy,
    ) -> DomainResult<NotificationResult> {
        let target_users = self.resolve_notification_targets(sender_id, &policy).await?;
        
        debug!(
            "策略 {:?} 解析出 {} 个目标用户",
            policy,
            target_users.len()
        );
        
        let mut success_count = 0;
        let mut failed_users = Vec::new();
        
        for user_id in &target_users {
            match self.message_router.send_to_user(*user_id, message.clone()).await {
                Ok(()) => success_count += 1,
                Err(e) => {
                    warn!("向用户 {} 发送通知失败: {}", user_id, e);
                    failed_users.push(*user_id);
                }
            }
        }
        
        Ok(NotificationResult {
            total_targets: target_users.len(),
            success_count,
            failed_count: failed_users.len(),
            failed_users,
            policy_used: policy,
        })
    }
    
    /// 发送用户状态更新通知
    pub async fn notify_user_status_update(
        &self,
        user_id: i64,
        status_message: ServerMessage,
    ) -> DomainResult<NotificationResult> {
        let mut policies = Vec::new();
        
        if self.config.enable_friend_notifications {
            policies.push(NotificationPolicy::FriendNotification);
        }
        
        if self.config.enable_group_notifications {
            policies.push(NotificationPolicy::GroupMemberNotification);
        }
        
        if self.config.enable_nearby_notifications {
            policies.push(NotificationPolicy::NearbyNotification {
                radius_km: self.config.nearby_radius_km,
            });
        }
        
        if policies.is_empty() {
            return Ok(NotificationResult::empty(NotificationPolicy::CombinedPolicy(vec![])));
        }
        
        let combined_policy = NotificationPolicy::CombinedPolicy(policies);
        self.send_notification_with_policy(user_id, status_message, combined_policy).await
    }
    
    /// 发送群组事件通知
    pub async fn notify_group_event(
        &self,
        group_id: Uuid,
        event_message: ServerMessage,
        exclude_user_id: Option<i64>,
    ) -> DomainResult<NotificationResult> {
        let group_members = self.target_resolver.get_group_members(group_id).await?;
        
        let target_users: Vec<i64> = if let Some(exclude_id) = exclude_user_id {
            group_members.into_iter().filter(|&id| id != exclude_id).collect()
        } else {
            group_members
        };
        
        debug!(
            "向群组 {} 的 {} 个成员发送事件通知",
            group_id,
            target_users.len()
        );
        
        let mut success_count = 0;
        let mut failed_users = Vec::new();
        
        for user_id in &target_users {
            match self.message_router.send_to_user(*user_id, event_message.clone()).await {
                Ok(()) => success_count += 1,
                Err(e) => {
                    warn!("向群组成员 {} 发送通知失败: {}", user_id, e);
                    failed_users.push(*user_id);
                }
            }
        }
        
        Ok(NotificationResult {
            total_targets: target_users.len(),
            success_count,
            failed_count: failed_users.len(),
            failed_users,
            policy_used: NotificationPolicy::GroupMemberNotification,
        })
    }
    
    /// 发送新生活瞬间通知
    pub async fn notify_new_moment(
        &self,
        creator_id: i64,
        moment_message: ServerMessage,
    ) -> DomainResult<NotificationResult> {
        // 使用组合策略：好友 + 群组成员 + 附近用户
        let mut policies = Vec::new();
        
        if self.config.enable_friend_notifications {
            policies.push(NotificationPolicy::FriendNotification);
        }
        
        if self.config.enable_group_notifications {
            policies.push(NotificationPolicy::GroupMemberNotification);
        }
        
        if self.config.enable_nearby_notifications {
            policies.push(NotificationPolicy::NearbyNotification {
                radius_km: self.config.nearby_radius_km,
            });
        }
        
        let combined_policy = NotificationPolicy::CombinedPolicy(policies);
        self.send_notification_with_policy(creator_id, moment_message, combined_policy).await
    }
    
    /// 发送系统通知
    pub async fn send_system_notification(
        &self,
        target_users: Vec<i64>,
        system_message: ServerMessage,
    ) -> DomainResult<NotificationResult> {
        debug!("发送系统通知给 {} 个用户", target_users.len());
        
        let mut success_count = 0;
        let mut failed_users = Vec::new();
        
        for user_id in &target_users {
            match self.message_router.send_to_user(*user_id, system_message.clone()).await {
                Ok(()) => success_count += 1,
                Err(e) => {
                    warn!("向用户 {} 发送系统通知失败: {}", user_id, e);
                    failed_users.push(*user_id);
                }
            }
        }
        
        Ok(NotificationResult {
            total_targets: target_users.len(),
            success_count,
            failed_count: failed_users.len(),
            failed_users,
            policy_used: NotificationPolicy::CombinedPolicy(vec![]),
        })
    }
    
    /// 解析通知目标用户
    fn resolve_notification_targets<'a>(
        &'a self,
        sender_id: i64,
        policy: &'a NotificationPolicy,
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = DomainResult<Vec<i64>>> + Send + 'a>> {
        Box::pin(async move {
        match policy {
            NotificationPolicy::FriendNotification => {
                self.target_resolver.get_user_friends(sender_id).await
            }
            NotificationPolicy::GroupMemberNotification => {
                self.target_resolver.get_user_group_members(sender_id).await
            }
            NotificationPolicy::NearbyNotification { radius_km } => {
                self.target_resolver.get_nearby_users(sender_id, *radius_km).await
            }
            NotificationPolicy::CombinedPolicy(policies) => {
                let mut all_targets = HashSet::new();
                
                for sub_policy in policies {
                    match self.resolve_notification_targets(sender_id, sub_policy).await {
                        Ok(targets) => {
                            for target in targets {
                                all_targets.insert(target);
                            }
                        }
                        Err(e) => {
                            warn!("策略 {:?} 解析失败: {}", sub_policy, e);
                        }
                    }
                }
                
                // 排除发送者自己
                all_targets.remove(&sender_id);
                
                Ok(all_targets.into_iter().collect())
            }
        }
        })
    }
    
    /// 获取配置
    pub fn get_config(&self) -> &NotificationPolicyConfig {
        &self.config
    }
    
    /// 更新配置
    pub fn update_config(&mut self, config: NotificationPolicyConfig) {
        self.config = config;
        info!("通知策略配置已更新: {:?}", self.config);
    }
    
    /// 获取策略统计信息
    pub async fn get_policy_statistics(&self) -> DomainResult<PolicyStatistics> {
        // 这里可以添加统计逻辑，比如各种策略的使用频率等
        Ok(PolicyStatistics {
            friend_notifications_enabled: self.config.enable_friend_notifications,
            group_notifications_enabled: self.config.enable_group_notifications,
            nearby_notifications_enabled: self.config.enable_nearby_notifications,
            nearby_radius_km: self.config.nearby_radius_km,
            global_broadcast_enabled: self.config.enable_global_broadcast,
        })
    }
}

/// 通知结果
#[derive(Debug, Clone, Serialize)]
pub struct NotificationResult {
    /// 总目标用户数
    pub total_targets: usize,
    /// 成功发送数
    pub success_count: usize,
    /// 失败发送数
    pub failed_count: usize,
    /// 失败的用户ID列表
    pub failed_users: Vec<i64>,
    /// 使用的策略
    pub policy_used: NotificationPolicy,
}

impl NotificationResult {
    /// 创建空结果
    pub fn empty(policy: NotificationPolicy) -> Self {
        Self {
            total_targets: 0,
            success_count: 0,
            failed_count: 0,
            failed_users: vec![],
            policy_used: policy,
        }
    }
    
    /// 是否完全成功
    pub fn is_fully_successful(&self) -> bool {
        self.failed_count == 0 && self.total_targets > 0
    }
    
    /// 获取成功率
    pub fn success_rate(&self) -> f64 {
        if self.total_targets == 0 {
            0.0
        } else {
            self.success_count as f64 / self.total_targets as f64
        }
    }
}

/// 策略统计信息
#[derive(Debug, Clone, Serialize)]
pub struct PolicyStatistics {
    pub friend_notifications_enabled: bool,
    pub group_notifications_enabled: bool,
    pub nearby_notifications_enabled: bool,
    pub nearby_radius_km: f64,
    pub global_broadcast_enabled: bool,
}

/// 消息优先级
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum MessagePriority {
    /// 低优先级（如状态更新）
    Low,
    /// 普通优先级（如普通消息）
    Normal,
    /// 高优先级（如群组事件）
    High,
    /// 紧急优先级（如系统通知）
    Urgent,
}

/// 智能通知调度器
pub struct SmartNotificationScheduler {
    policy_service: Arc<NotificationPolicyService>,
    /// 消息队列（按优先级排序）
    message_queue: tokio::sync::Mutex<std::collections::BinaryHeap<PrioritizedMessage>>,
}

/// 带优先级的消息
#[derive(Debug, Clone)]
struct PrioritizedMessage {
    priority: MessagePriority,
    sender_id: i64,
    message: ServerMessage,
    policy: NotificationPolicy,
    timestamp: chrono::DateTime<chrono::Utc>,
}

impl PartialEq for PrioritizedMessage {
    fn eq(&self, other: &Self) -> bool {
        self.priority == other.priority && self.timestamp == other.timestamp
    }
}

impl Eq for PrioritizedMessage {}

impl PartialOrd for PrioritizedMessage {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for PrioritizedMessage {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        // 优先级高的排在前面，时间早的排在前面
        other.priority.cmp(&self.priority)
            .then_with(|| self.timestamp.cmp(&other.timestamp))
    }
}

// PartialEq is now derived

impl PartialOrd for MessagePriority {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for MessagePriority {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        use MessagePriority::*;
        match (self, other) {
            (Urgent, Urgent) | (High, High) | (Normal, Normal) | (Low, Low) => std::cmp::Ordering::Equal,
            (Urgent, _) => std::cmp::Ordering::Greater,
            (_, Urgent) => std::cmp::Ordering::Less,
            (High, _) => std::cmp::Ordering::Greater,
            (_, High) => std::cmp::Ordering::Less,
            (Normal, Low) => std::cmp::Ordering::Greater,
            (Low, Normal) => std::cmp::Ordering::Less,
        }
    }
}

impl SmartNotificationScheduler {
    /// 创建新的智能通知调度器
    pub fn new(policy_service: Arc<NotificationPolicyService>) -> Self {
        Self {
            policy_service,
            message_queue: tokio::sync::Mutex::new(std::collections::BinaryHeap::new()),
        }
    }
    
    /// 调度通知发送
    pub async fn schedule_notification(
        &self,
        sender_id: i64,
        message: ServerMessage,
        policy: NotificationPolicy,
        priority: MessagePriority,
    ) -> DomainResult<()> {
        let prioritized_message = PrioritizedMessage {
            priority,
            sender_id,
            message,
            policy,
            timestamp: chrono::Utc::now(),
        };
        
        let mut queue = self.message_queue.lock().await;
        queue.push(prioritized_message);
        
        debug!("消息已加入调度队列，当前队列长度: {}", queue.len());
        Ok(())
    }
    
    /// 处理队列中的消息
    pub async fn process_queue(&self) -> DomainResult<Vec<NotificationResult>> {
        let mut queue = self.message_queue.lock().await;
        let mut results = Vec::new();
        
        while let Some(prioritized_message) = queue.pop() {
            match self.policy_service.send_notification_with_policy(
                prioritized_message.sender_id,
                prioritized_message.message,
                prioritized_message.policy,
            ).await {
                Ok(result) => results.push(result),
                Err(e) => {
                    warn!("处理队列消息失败: {}", e);
                }
            }
        }
        
        Ok(results)
    }
    
    /// 获取队列状态
    pub async fn get_queue_status(&self) -> QueueStatus {
        let queue = self.message_queue.lock().await;
        QueueStatus {
            total_messages: queue.len(),
            urgent_count: queue.iter().filter(|m| m.priority == MessagePriority::Urgent).count(),
            high_count: queue.iter().filter(|m| m.priority == MessagePriority::High).count(),
            normal_count: queue.iter().filter(|m| m.priority == MessagePriority::Normal).count(),
            low_count: queue.iter().filter(|m| m.priority == MessagePriority::Low).count(),
        }
    }
}

/// 队列状态
#[derive(Debug, Clone, Serialize)]
pub struct QueueStatus {
    pub total_messages: usize,
    pub urgent_count: usize,
    pub high_count: usize,
    pub normal_count: usize,
    pub low_count: usize,
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_message_priority_ordering() {
        use MessagePriority::*;
        
        assert!(Urgent > High);
        assert!(High > Normal);
        assert!(Normal > Low);
        assert_eq!(Normal, Normal);
    }
    
    #[test]
    fn test_notification_result_success_rate() {
        let result = NotificationResult {
            total_targets: 10,
            success_count: 8,
            failed_count: 2,
            failed_users: vec![1, 2],
            policy_used: NotificationPolicy::FriendNotification,
        };
        
        assert_eq!(result.success_rate(), 0.8);
        assert!(!result.is_fully_successful());
        
        let perfect_result = NotificationResult {
            total_targets: 5,
            success_count: 5,
            failed_count: 0,
            failed_users: vec![],
            policy_used: NotificationPolicy::FriendNotification,
        };
        
        assert_eq!(perfect_result.success_rate(), 1.0);
        assert!(perfect_result.is_fully_successful());
    }
}
