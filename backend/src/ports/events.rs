/// 事件发布接口定义 - 事件驱动架构支持
/// 
/// 事件特征：
/// - 异步事件发布
/// - 事件持久化
/// - 事件重放
/// - 事件订阅

use async_trait::async_trait;
use serde::{Serialize, Deserialize};
use crate::domain::{DomainResult, DomainEvent};

// === 事件发布器接口 ===

/// 事件发布器接口
#[async_trait]
pub trait EventPublisher: Send + Sync {
    /// 发布单个事件
    async fn publish(&self, event: &DomainEvent) -> DomainResult<()>;
    
    /// 批量发布事件
    async fn publish_batch(&self, events: &[DomainEvent]) -> DomainResult<()>;
    
    /// 发布事件并等待处理完成
    async fn publish_and_wait(&self, event: &DomainEvent) -> DomainResult<()>;
}

// === 事件存储接口 ===

/// 事件存储接口
#[async_trait]
pub trait EventStore: Send + Sync + Clone {
    /// 保存事件
    async fn save_event(&self, event: &DomainEvent) -> DomainResult<EventId>;
    
    /// 保存事件批次
    async fn save_events(&self, events: &[DomainEvent]) -> DomainResult<Vec<EventId>>;
    
    /// 根据聚合ID获取事件
    async fn get_events_by_aggregate_id(&self, aggregate_id: &str) -> DomainResult<Vec<StoredEvent>>;
    
    /// 根据事件类型获取事件
    async fn get_events_by_type(&self, event_type: &str) -> DomainResult<Vec<StoredEvent>>;
    
    /// 获取指定时间范围内的事件
    async fn get_events_by_time_range(
        &self,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
    ) -> DomainResult<Vec<StoredEvent>>;
    
    /// 获取事件流（分页）
    async fn get_event_stream(
        &self,
        cursor: Option<EventId>,
        limit: usize,
    ) -> DomainResult<EventStream>;
}

/// 事件ID
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
pub struct EventId(pub i64);

impl EventId {
    /// 创建新的事件ID
    pub fn new(id: i64) -> Self {
        Self(id)
    }
    
    /// 获取ID值
    pub fn value(&self) -> i64 {
        self.0
    }
}

/// 存储的事件
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct StoredEvent {
    pub event_id: EventId,
    pub aggregate_id: String,
    pub event_type: String,
    pub event_data: serde_json::Value,
    pub event_version: u32,
    pub occurred_at: chrono::DateTime<chrono::Utc>,
    pub stored_at: chrono::DateTime<chrono::Utc>,
}

/// 事件流
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct EventStream {
    pub events: Vec<StoredEvent>,
    pub has_more: bool,
    pub next_cursor: Option<EventId>,
}

// === 事件处理器接口 ===

/// 事件处理器接口
#[async_trait]
pub trait EventHandler<T>: Send + Sync {
    /// 处理事件
    async fn handle(&self, event: &T) -> DomainResult<()>;
    
    /// 获取处理器名称
    fn handler_name(&self) -> &'static str;
    
    /// 检查是否可以处理该事件
    fn can_handle(&self, event_type: &str) -> bool;
}

// === 事件总线接口 ===

/// 事件总线接口
#[async_trait]
pub trait EventBus: Send + Sync + Clone {
    /// 注册事件处理器
    async fn register_handler<T>(&self, handler: Box<dyn EventHandler<T>>) -> DomainResult<()>
    where
        T: Send + Sync + 'static;
    
    /// 发布事件到总线
    async fn publish(&self, event: &DomainEvent) -> DomainResult<()>;
    
    /// 启动事件总线
    async fn start(&self) -> DomainResult<()>;
    
    /// 停止事件总线
    async fn stop(&self) -> DomainResult<()>;
    
    /// 获取总线状态
    async fn get_status(&self) -> DomainResult<EventBusStatus>;
}

/// 事件总线状态
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct EventBusStatus {
    pub is_running: bool,
    pub registered_handlers: u32,
    pub processed_events: u64,
    pub failed_events: u64,
    pub last_processed_at: Option<chrono::DateTime<chrono::Utc>>,
}

// === 事件订阅接口 ===

/// 事件订阅器接口
#[async_trait]
pub trait EventSubscriber: Send + Sync + Clone {
    /// 订阅事件类型
    async fn subscribe(&self, event_type: &str, callback: EventCallback) -> DomainResult<SubscriptionId>;
    
    /// 取消订阅
    async fn unsubscribe(&self, subscription_id: SubscriptionId) -> DomainResult<()>;
    
    /// 订阅所有事件
    async fn subscribe_all(&self, callback: EventCallback) -> DomainResult<SubscriptionId>;
    
    /// 获取订阅状态
    async fn get_subscription_status(&self, subscription_id: SubscriptionId) -> DomainResult<SubscriptionStatus>;
}

/// 订阅ID
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
pub struct SubscriptionId(pub uuid::Uuid);

impl SubscriptionId {
    /// 创建新的订阅ID
    pub fn new() -> Self {
        Self(uuid::Uuid::new_v4())
    }
    
    /// 获取UUID值
    pub fn value(&self) -> uuid::Uuid {
        self.0
    }
}

impl Default for SubscriptionId {
    fn default() -> Self {
        Self::new()
    }
}

/// 事件回调函数类型
pub type EventCallback = Box<dyn Fn(&DomainEvent) -> DomainResult<()> + Send + Sync>;

/// 订阅状态
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct SubscriptionStatus {
    pub subscription_id: SubscriptionId,
    pub event_type: Option<String>, // None表示订阅所有事件
    pub is_active: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub last_event_at: Option<chrono::DateTime<chrono::Utc>>,
    pub processed_events: u64,
    pub failed_events: u64,
}

// === 事件重放接口 ===

/// 事件重放器接口
#[async_trait]
pub trait EventReplayer: Send + Sync + Clone {
    /// 重放聚合的所有事件
    async fn replay_aggregate(&self, aggregate_id: &str) -> DomainResult<()>;
    
    /// 重放指定时间范围内的事件
    async fn replay_time_range(
        &self,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
    ) -> DomainResult<()>;
    
    /// 重放指定类型的事件
    async fn replay_event_type(&self, event_type: &str) -> DomainResult<()>;
    
    /// 获取重放状态
    async fn get_replay_status(&self) -> DomainResult<ReplayStatus>;
}

/// 重放状态
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ReplayStatus {
    pub is_replaying: bool,
    pub total_events: u64,
    pub processed_events: u64,
    pub failed_events: u64,
    pub started_at: Option<chrono::DateTime<chrono::Utc>>,
    pub estimated_completion: Option<chrono::DateTime<chrono::Utc>>,
}

// === 事件工厂接口 ===

/// 事件工厂 - 创建和管理事件相关服务
pub trait EventFactory: Send + Sync + Clone {
    type Publisher: EventPublisher;
    type Store: EventStore;
    type Bus: EventBus;
    type Subscriber: EventSubscriber;
    type Replayer: EventReplayer;
    
    /// 创建事件发布器
    fn event_publisher(&self) -> Self::Publisher;
    
    /// 创建事件存储
    fn event_store(&self) -> Self::Store;
    
    /// 创建事件总线
    fn event_bus(&self) -> Self::Bus;
    
    /// 创建事件订阅器
    fn event_subscriber(&self) -> Self::Subscriber;
    
    /// 创建事件重放器
    fn event_replayer(&self) -> Self::Replayer;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_id() {
        let event_id = EventId::new(123);
        assert_eq!(event_id.value(), 123);
    }
    
    #[test]
    fn test_subscription_id() {
        let sub_id = SubscriptionId::new();
        assert!(!sub_id.value().is_nil());
        
        let sub_id2 = SubscriptionId::new();
        assert_ne!(sub_id, sub_id2);
    }
    
    #[test]
    fn test_event_stream() {
        let stream = EventStream {
            events: vec![],
            has_more: false,
            next_cursor: None,
        };
        
        assert!(stream.events.is_empty());
        assert!(!stream.has_more);
        assert!(stream.next_cursor.is_none());
    }
}

// === 类型别名（向后兼容） ===

/// 事件发布器端口（向后兼容别名）
pub type EventPublisherPort = dyn EventPublisher;

/// 事件存储端口（向后兼容别名）
pub type EventStorePort = dyn EventStore;

/// 事件信封 - 包装事件的元数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventEnvelope {
    /// 事件ID
    pub event_id: String,
    /// 事件类型
    pub event_type: String,
    /// 事件数据
    pub event_data: serde_json::Value,
    /// 聚合ID
    pub aggregate_id: String,
    /// 聚合类型
    pub aggregate_type: String,
    /// 事件版本
    pub version: i64,
    /// 事件时间戳
    pub timestamp: chrono::DateTime<chrono::Utc>,
    /// 事件元数据
    pub metadata: std::collections::HashMap<String, String>,
}

/// 领域事件信封（向后兼容别名）
pub type DomainEventEnvelope = EventEnvelope;
