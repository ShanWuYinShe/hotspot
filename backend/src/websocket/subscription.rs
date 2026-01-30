use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tokio::sync::RwLock;

/// 订阅类型 - 根据 API 文档 4.4 节定义
/// 支持两种订阅源类型：group, user
///
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum SubscriptionType {
    /// 群聊消息订阅 - 订阅指定群聊的消息
    /// source_id: group_chat_id (UUID 字符串)
    /// 私聊消息自动推送，不需要订阅
    Group(String),

    /// 用户状态订阅 - 订阅用户的在线状态变化、位置更新等
    /// source_id: user_id
    User(i64),
}

impl SubscriptionType {
    /// 转换为频道字符串
    pub fn to_channel(&self) -> String {
        match self {
            SubscriptionType::Group(group_id) => format!("group:{}", group_id),
            SubscriptionType::User(user_id) => format!("user:{}", user_id),
        }
    }
}

/// 订阅管理器
/// 维护用户订阅和反向索引，支持高效的订阅查询和消息推送
///
/// 注意：内部使用 i64 作为 user_id 键（性能优化），但对外接口使用 String
#[derive(Clone)]
pub struct SubscriptionManager {
    /// 用户订阅列表: user_id (i64) -> Set<SubscriptionType>
    subscriptions: Arc<RwLock<HashMap<i64, HashSet<SubscriptionType>>>>,
    /// 反向索引: channel_key -> Set<user_id (i64)>
    /// 用于快速查找订阅了某个频道的所有用户
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
        let channel_key = subscription.to_channel();

        // 添加到用户订阅列表
        let mut subscriptions = self.subscriptions.write().await;
        subscriptions
            .entry(user_id)
            .or_insert_with(HashSet::new)
            .insert(subscription.clone());

        // 更新反向索引
        let mut reverse_index = self.reverse_index.write().await;
        reverse_index
            .entry(channel_key.clone())
            .or_insert_with(HashSet::new)
            .insert(user_id);

        tracing::debug!("用户订阅: user_id={}, channel={}", user_id, channel_key);
    }

    /// 取消订阅
    pub async fn unsubscribe(&self, user_id: i64, subscription: &SubscriptionType) {
        let channel_key = subscription.to_channel();

        // 从用户订阅列表移除
        let mut subscriptions = self.subscriptions.write().await;
        if let Some(user_subs) = subscriptions.get_mut(&user_id) {
            user_subs.remove(subscription);
            if user_subs.is_empty() {
                subscriptions.remove(&user_id);
            }
        }

        // 更新反向索引
        let mut reverse_index = self.reverse_index.write().await;
        if let Some(users) = reverse_index.get_mut(&channel_key) {
            users.remove(&user_id);
            if users.is_empty() {
                reverse_index.remove(&channel_key);
            }
        }

        tracing::debug!("用户取消订阅: user_id={}, channel={}", user_id, channel_key);
    }

    /// 取消用户的所有订阅（用户断开连接时调用）
    pub async fn unsubscribe_all(&self, user_id: i64) {
        let mut subscriptions = self.subscriptions.write().await;

        if let Some(user_subs) = subscriptions.remove(&user_id) {
            let mut reverse_index = self.reverse_index.write().await;
            for subscription in user_subs {
                let channel_key = subscription.to_channel();
                if let Some(users) = reverse_index.get_mut(&channel_key) {
                    users.remove(&user_id);
                    if users.is_empty() {
                        reverse_index.remove(&channel_key);
                    }
                }
            }
        }

        tracing::debug!("用户取消所有订阅: user_id={}", user_id);
    }

    /// 获取订阅了某个群聊的所有用户（用于群聊消息推送）
    pub async fn get_group_chat_subscribers(&self, group_id: &str) -> Vec<i64> {
        let channel_key = format!("group:{}", group_id);
        let reverse_index = self.reverse_index.read().await;

        reverse_index
            .get(&channel_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }

    /// 获取订阅了某个用户状态的所有用户（用于在线状态推送）
    pub async fn get_user_status_subscribers(&self, target_user_id: i64) -> Vec<i64> {
        let channel_key = format!("user:{}", target_user_id);
        let reverse_index = self.reverse_index.read().await;

        reverse_index
            .get(&channel_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }

    /// 判断用户是否订阅了指定订阅项
    ///
    /// 说明：
    /// - WebSocket 推送必须严格按订阅执行（不允许“直接推送/广播”绕过订阅）
    pub async fn has_subscription(&self, user_id: i64, subscription: &SubscriptionType) -> bool {
        let subscriptions = self.subscriptions.read().await;
        subscriptions
            .get(&user_id)
            .map(|set| set.contains(subscription))
            .unwrap_or(false)
    }

    /// 获取用户的所有订阅
    pub async fn get_subscriptions(&self, user_id: i64) -> Vec<SubscriptionType> {
        let subscriptions = self.subscriptions.read().await;
        subscriptions
            .get(&user_id)
            .map(|set| set.iter().cloned().collect())
            .unwrap_or_default()
    }

    /// 统计某类订阅数量（group / user）
    pub async fn count_by_type(&self, user_id: i64, source_type: &str) -> usize {
        let subscriptions = self.subscriptions.read().await;
        let Some(set) = subscriptions.get(&user_id) else {
            return 0;
        };
        set.iter()
            .filter(|s| match (source_type, s) {
                ("group", SubscriptionType::Group(_)) => true,
                ("user", SubscriptionType::User(_)) => true,
                _ => false,
            })
            .count()
    }
}

impl Default for SubscriptionManager {
    fn default() -> Self {
        Self::new()
    }
}
