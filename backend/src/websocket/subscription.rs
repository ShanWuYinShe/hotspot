use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tokio::sync::RwLock;

/// 订阅类型
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum SubscriptionType {
    /// 用户消息（单聊）
    UserMessage(i64),
    /// 群组消息
    GroupMessage(String),
    /// 收件箱通知
    Inbox,
    /// 用户在线状态
    UserOnlineStatus(i64),
    /// 附近的人更新
    NearbyUsers,
    /// 群组事件
    GroupEvents(String),
    /// 生活瞬间更新
    MomentUpdates,
}

/// 订阅管理器
/// 
/// 实现双层订阅机制：
/// 1. 持久化订阅偏好（存储在数据库的notification_subscriptions表）
/// 2. 实时订阅控制（通过WebSocket Subscribe/Unsubscribe消息）
#[derive(Clone)]
pub struct SubscriptionManager {
    // user_id -> Set<SubscriptionType>
    subscriptions: Arc<RwLock<HashMap<i64, HashSet<SubscriptionType>>>>,
    // 反向索引：subscription_type -> Set<user_id>
    // 用于快速查找订阅了某个类型的所有用户
    reverse_index: Arc<RwLock<HashMap<String, HashSet<i64>>>>,
}

impl SubscriptionManager {
    pub fn new() -> Self {
        Self {
            subscriptions: Arc::new(RwLock::new(HashMap::new())),
            reverse_index: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 添加订阅
    pub async fn subscribe(&self, user_id: i64, subscription: SubscriptionType) {
        let sub_key = self.subscription_key(&subscription);

        // 添加到用户订阅列表
        let mut subscriptions = self.subscriptions.write().await;
        subscriptions
            .entry(user_id)
            .or_insert_with(HashSet::new)
            .insert(subscription.clone());

        // 更新反向索引
        let mut reverse_index = self.reverse_index.write().await;
        reverse_index
            .entry(sub_key)
            .or_insert_with(HashSet::new)
            .insert(user_id);

        tracing::debug!("用户订阅: user_id={}, subscription={:?}", user_id, subscription);
    }

    /// 取消订阅
    pub async fn unsubscribe(&self, user_id: i64, subscription: &SubscriptionType) {
        let sub_key = self.subscription_key(subscription);

        // 从用户订阅列表中移除
        let mut subscriptions = self.subscriptions.write().await;
        if let Some(user_subs) = subscriptions.get_mut(&user_id) {
            user_subs.remove(subscription);
            
            // 如果用户没有其他订阅，移除该用户
            if user_subs.is_empty() {
                subscriptions.remove(&user_id);
            }
        }

        // 更新反向索引
        let mut reverse_index = self.reverse_index.write().await;
        if let Some(users) = reverse_index.get_mut(&sub_key) {
            users.remove(&user_id);
            
            // 如果没有用户订阅该类型，移除该类型
            if users.is_empty() {
                reverse_index.remove(&sub_key);
            }
        }

        tracing::debug!("用户取消订阅: user_id={}, subscription={:?}", user_id, subscription);
    }

    /// 取消用户的所有订阅（用户断开连接时调用）
    pub async fn unsubscribe_all(&self, user_id: i64) {
        let mut subscriptions = self.subscriptions.write().await;
        
        if let Some(user_subs) = subscriptions.remove(&user_id) {
            // 更新反向索引
            let mut reverse_index = self.reverse_index.write().await;
            for subscription in user_subs {
                let sub_key = self.subscription_key(&subscription);
                if let Some(users) = reverse_index.get_mut(&sub_key) {
                    users.remove(&user_id);
                    if users.is_empty() {
                        reverse_index.remove(&sub_key);
                    }
                }
            }
        }

        tracing::debug!("用户取消所有订阅: user_id={}", user_id);
    }

    /// 检查用户是否订阅了某个类型
    pub async fn is_subscribed(&self, user_id: i64, subscription: &SubscriptionType) -> bool {
        let subscriptions = self.subscriptions.read().await;
        subscriptions
            .get(&user_id)
            .map(|subs| subs.contains(subscription))
            .unwrap_or(false)
    }

    /// 获取订阅了某个类型的所有用户
    pub async fn get_subscribers(&self, subscription: &SubscriptionType) -> Vec<i64> {
        let sub_key = self.subscription_key(subscription);
        let reverse_index = self.reverse_index.read().await;
        
        reverse_index
            .get(&sub_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }

    /// 获取订阅了某个用户消息的所有用户（用于单聊消息推送）
    pub async fn get_user_message_subscribers(&self, target_user_id: i64) -> Vec<i64> {
        self.get_subscribers(&SubscriptionType::UserMessage(target_user_id))
            .await
    }

    /// 获取订阅了某个群组消息的所有用户（用于群聊消息推送）
    pub async fn get_group_message_subscribers(&self, group_id: &str) -> Vec<i64> {
        self.get_subscribers(&SubscriptionType::GroupMessage(group_id.to_string()))
            .await
    }

    /// 获取订阅了收件箱通知的所有用户
    pub async fn get_inbox_subscribers(&self) -> Vec<i64> {
        self.get_subscribers(&SubscriptionType::Inbox).await
    }

    /// 获取用户的所有订阅
    pub async fn get_user_subscriptions(&self, user_id: i64) -> Vec<SubscriptionType> {
        let subscriptions = self.subscriptions.read().await;
        subscriptions
            .get(&user_id)
            .map(|subs| subs.iter().cloned().collect())
            .unwrap_or_default()
    }

    /// 批量订阅
    pub async fn subscribe_batch(&self, user_id: i64, subscriptions: Vec<SubscriptionType>) {
        for subscription in subscriptions {
            self.subscribe(user_id, subscription).await;
        }
    }

    /// 生成订阅键（用于反向索引）
    fn subscription_key(&self, subscription: &SubscriptionType) -> String {
        match subscription {
            SubscriptionType::UserMessage(user_id) => format!("user_message:{}", user_id),
            SubscriptionType::GroupMessage(group_id) => format!("group_message:{}", group_id),
            SubscriptionType::Inbox => "inbox".to_string(),
            SubscriptionType::UserOnlineStatus(user_id) => format!("user_online:{}", user_id),
            SubscriptionType::NearbyUsers => "nearby_users".to_string(),
            SubscriptionType::GroupEvents(group_id) => format!("group_events:{}", group_id),
            SubscriptionType::MomentUpdates => "moment_updates".to_string(),
        }
    }

    /// 获取订阅统计信息
    pub async fn get_stats(&self) -> SubscriptionStats {
        let subscriptions = self.subscriptions.read().await;
        let reverse_index = self.reverse_index.read().await;

        SubscriptionStats {
            total_users: subscriptions.len(),
            total_subscription_types: reverse_index.len(),
            total_subscriptions: subscriptions.values().map(|s| s.len()).sum(),
        }
    }
}

impl Default for SubscriptionManager {
    fn default() -> Self {
        Self::new()
    }
}

/// 订阅统计信息
#[derive(Debug)]
pub struct SubscriptionStats {
    pub total_users: usize,
    pub total_subscription_types: usize,
    pub total_subscriptions: usize,
}
