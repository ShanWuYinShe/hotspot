/// 事件发布器端口定义
/// 
/// 定义领域事件发布相关的抽象接口
use async_trait::async_trait;
use crate::domain::{DomainResult, UserEvent, GroupEvent, MessageEvent, MomentEvent};
use crate::domain::events::DomainEvent;

/// 事件发布器端口
#[async_trait]
pub trait EventPublisherPort: Send + Sync {
    /// 发布用户事件
    async fn publish_user_event(&self, event: UserEvent) -> DomainResult<()>;
    
    /// 发布群组事件
    async fn publish_group_event(&self, event: GroupEvent) -> DomainResult<()>;
    
    /// 发布消息事件
    async fn publish_message_event(&self, event: MessageEvent) -> DomainResult<()>;
    
    /// 发布瞬间事件
    async fn publish_moment_event(&self, event: MomentEvent) -> DomainResult<()>;
    
    /// 批量发布事件
    async fn publish_events(&self, events: Vec<DomainEventEnvelope>) -> DomainResult<()>;
    
    /// 获取发布器统计信息
    async fn get_publisher_stats(&self) -> DomainResult<PublisherStats>;
}

/// 领域事件封装枚举（用于批量发布），避免与 DomainEvent trait 同名
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum DomainEventEnvelope {
    User(UserEvent),
    Group(GroupEvent),
    Message(MessageEvent),
    Moment(MomentEvent),
}

impl DomainEventEnvelope {
    /// 获取事件类型
    pub fn event_type(&self) -> &'static str {
        match self {
            DomainEventEnvelope::User(_) => "UserEvent",
            DomainEventEnvelope::Group(_) => "GroupEvent", 
            DomainEventEnvelope::Message(_) => "MessageEvent",
            DomainEventEnvelope::Moment(_) => "MomentEvent",
        }
    }
    
    /// 获取事件发生时间
    pub fn occurred_at(&self) -> chrono::DateTime<chrono::Utc> {
        match self {
            DomainEventEnvelope::User(event) => event.occurred_at(),
            DomainEventEnvelope::Group(event) => event.occurred_at(),
            DomainEventEnvelope::Message(event) => event.occurred_at(),
            DomainEventEnvelope::Moment(event) => event.occurred_at(),
        }
    }
    
    /// 获取聚合根ID
    pub fn aggregate_id(&self) -> i64 {
        match self {
            DomainEventEnvelope::User(event) => event.aggregate_id(),
            DomainEventEnvelope::Group(event) => event.aggregate_id(),
            DomainEventEnvelope::Message(event) => event.aggregate_id(),
            DomainEventEnvelope::Moment(event) => event.aggregate_id(),
        }
    }
}

impl From<UserEvent> for DomainEventEnvelope {
    fn from(event: UserEvent) -> Self {
        DomainEventEnvelope::User(event)
    }
}

impl From<GroupEvent> for DomainEventEnvelope {
    fn from(event: GroupEvent) -> Self {
        DomainEventEnvelope::Group(event)
    }
}

impl From<MessageEvent> for DomainEventEnvelope {
    fn from(event: MessageEvent) -> Self {
        DomainEventEnvelope::Message(event)
    }
}

impl From<MomentEvent> for DomainEventEnvelope {
    fn from(event: MomentEvent) -> Self {
        DomainEventEnvelope::Moment(event)
    }
}

/// 发布器统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PublisherStats {
    /// 已发布事件总数
    pub total_published: u64,
    /// 发布失败事件数
    pub failed_published: u64,
    /// 各类型事件发布数量
    pub by_type: std::collections::HashMap<String, u64>,
    /// 平均发布延迟（毫秒）
    pub avg_publish_latency_ms: f64,
    /// 最后发布时间
    pub last_published_at: Option<chrono::DateTime<chrono::Utc>>,
}

/// 密码服务端口（用于用户认证）
#[async_trait]
pub trait PasswordServicePort: Send + Sync {
    /// 哈希密码
    async fn hash_password(&self, password: &str) -> DomainResult<String>;
    
    /// 验证密码
    async fn verify_password(&self, password: &str, hash: &str) -> DomainResult<bool>;
}
