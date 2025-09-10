/// 通知端口定义
/// 
/// 定义通知发送相关的抽象接口
/// 通知端口接口定义
/// 
/// 功能职责：
/// - 定义通知系统的抽象接口
/// - 提供多种通知渠道的统一接口
/// - 支持通知的个性化和模板化
/// - 实现通知的可靠性和容错机制
/// - 提供通知的状态跟踪和统计分析
/// 
/// 通知渠道：
/// - WebSocket通知：实时通知的主要渠道
/// - 邮件通知：重要事件和定期提醒
/// - 短信通知：安全验证和紧急通知
/// - 推送通知：移动端的离线通知
/// 
/// 通知类型：
/// - 即时通知：实时消息、状态更新等
/// - 延迟通知：定时提醒、周期报告等
/// - 批量通知：群发消息、系统公告等
/// - 个性化通知：基于用户偏好的定制通知
/// 
/// 质量保证：
/// - 发送状态跟踪和重试机制
/// - 通知去重和限流控制
/// - 通知内容的安全过滤和验证
use async_trait::async_trait;
use crate::domain::DomainResult;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// 通知传输端口
#[async_trait]
pub trait NotificationTransportPort: Send + Sync {
    /// 发送单个通知
    async fn send_notification(&self, notification: Notification) -> DomainResult<()>;
    
    /// 批量发送通知
    async fn send_notifications(&self, notifications: Vec<Notification>) -> DomainResult<()>;
    
    /// 发送广播通知（谨慎使用）
    async fn broadcast_notification(&self, notification: Notification) -> DomainResult<()>;
    
    /// 获取传输统计信息
    async fn get_transport_stats(&self) -> DomainResult<TransportStats>;
}

/// 通知策略端口
#[async_trait]
pub trait NotificationPolicyPort: Send + Sync {
    /// 根据策略获取目标用户列表
    async fn get_target_users(
        &self,
        policy: NotificationPolicy,
        context: NotificationContext,
    ) -> DomainResult<Vec<i64>>;
    
    /// 检查用户是否应该接收通知
    async fn should_notify_user(
        &self,
        user_id: i64,
        notification_type: NotificationType,
        context: &NotificationContext,
    ) -> DomainResult<bool>;
    
    /// 获取策略统计信息
    async fn get_policy_stats(&self) -> DomainResult<PolicyStats>;
}

/// 通知定义
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Notification {
    /// 通知ID
    pub id: String,
    /// 目标用户ID
    pub user_id: i64,
    /// 通知类型
    pub notification_type: NotificationType,
    /// 通知标题
    pub title: String,
    /// 通知内容
    pub content: String,
    /// 通知数据
    pub data: HashMap<String, serde_json::Value>,
    /// 通知渠道
    pub channels: Vec<NotificationChannel>,
    /// 创建时间
    pub created_at: chrono::DateTime<chrono::Utc>,
}

impl Notification {
    /// 创建新通知
    pub fn new(
        user_id: i64,
        notification_type: NotificationType,
        title: String,
        content: String,
    ) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            user_id,
            notification_type,
            title,
            content,
            data: HashMap::new(),
            channels: vec![NotificationChannel::WebSocket],
            created_at: chrono::Utc::now(),
        }
    }
    
    /// 添加数据
    pub fn with_data(mut self, key: String, value: serde_json::Value) -> Self {
        self.data.insert(key, value);
        self
    }
    
    /// 设置通知渠道
    pub fn with_channels(mut self, channels: Vec<NotificationChannel>) -> Self {
        self.channels = channels;
        self
    }
}

/// 通知类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum NotificationType {
    /// 系统通知
    System,
    /// 用户消息
    Message,
    /// 群组邀请
    GroupInvite,
    /// 群组消息
    GroupMessage,
    /// 好友请求
    FriendRequest,
    /// 瞬间点赞
    MomentLike,
    /// 瞬间评论
    MomentComment,
    /// 位置更新
    LocationUpdate,
}

/// 通知渠道
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum NotificationChannel {
    /// WebSocket实时通知
    WebSocket,
    /// 邮件通知
    Email,
    /// 短信通知
    Sms,
    /// 推送通知
    Push,
}

/// 通知策略
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum NotificationPolicy {
    /// 发送给好友
    Friends,
    /// 发送给群组成员
    GroupMembers { group_id: i64 },
    /// 发送给附近的用户
    Nearby { 
        latitude: f64,
        longitude: f64,
        radius_km: f64,
    },
    /// 发送给特定用户列表
    Specific { user_ids: Vec<i64> },
    /// 组合策略
    Combined { policies: Vec<NotificationPolicy> },
}

/// 通知上下文
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationContext {
    /// 触发用户ID
    pub trigger_user_id: i64,
    /// 相关群组ID
    pub group_id: Option<i64>,
    /// 相关消息ID
    pub message_id: Option<i64>,
    /// 相关瞬间ID
    pub moment_id: Option<i64>,
    /// 地理位置信息
    pub location: Option<LocationContext>,
    /// 额外上下文数据
    pub extra_data: HashMap<String, serde_json::Value>,
}

/// 地理位置上下文
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationContext {
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
}

/// 传输统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransportStats {
    /// 已发送通知数量
    pub sent_notifications: u64,
    /// 发送失败数量
    pub failed_notifications: u64,
    /// 平均发送时间（毫秒）
    pub avg_send_time_ms: u64,
}

/// 策略统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyStats {
    /// 策略使用次数统计
    pub policy_usage: HashMap<String, u64>,
    /// 平均目标用户数量
    pub avg_target_users: f64,
    /// 策略评估时间（毫秒）
    pub avg_evaluation_time_ms: u64,
}
