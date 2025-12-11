use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tokio::sync::RwLock;

/// 订阅类型 - 根据 API 文档 4.4 节定义
/// 支持三种订阅源类型：group_chat, user, notification
/// 
/// 注意：所有 ID 都使用 String 类型（JavaScript 不支持超过 2^53 的整数）
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum SubscriptionType {
    /// 群聊消息订阅 - 订阅指定群聊的消息
    /// source_id: group_chat_id (UUID 字符串)
    /// 私聊消息自动推送，不需要订阅
    GroupChat(String),
    
    /// 用户状态订阅 - 订阅用户的在线状态变化、位置更新等
    /// source_id: user_id (字符串，客户端传递字符串，后端内部解析)
    User(String),
    
    /// 通知订阅 - 订阅特定类型的通知
    /// source_id: notification_type (如 "new_follower", "moment_like" 等)
    Notification(String),
}

impl SubscriptionType {
    /// 转换为频道字符串
    pub fn to_channel(&self) -> String {
        match self {
            SubscriptionType::GroupChat(group_id) => format!("group_chat:{}", group_id),
            SubscriptionType::User(user_id) => format!("user:{}", user_id),
            SubscriptionType::Notification(notification_type) => format!("notification:{}", notification_type),
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
        let channel_key = format!("group_chat:{}", group_id);
        let reverse_index = self.reverse_index.read().await;
        
        reverse_index
            .get(&channel_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }

    /// 获取订阅了某个用户状态的所有用户（用于在线状态推送）
    /// 
    /// 注意：target_user_id 作为字符串传入（与客户端一致）
    pub async fn get_user_status_subscribers(&self, target_user_id: &str) -> Vec<i64> {
        let channel_key = format!("user:{}", target_user_id);
        let reverse_index = self.reverse_index.read().await;
        
        reverse_index
            .get(&channel_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }

    /// 获取订阅了某种通知类型的所有用户
    pub async fn get_notification_subscribers(&self, notification_type: &str) -> Vec<i64> {
        let channel_key = format!("notification:{}", notification_type);
        let reverse_index = self.reverse_index.read().await;
        
        reverse_index
            .get(&channel_key)
            .map(|users| users.iter().copied().collect())
            .unwrap_or_default()
    }
}

impl Default for SubscriptionManager {
    fn default() -> Self {
        Self::new()
    }
}
