/// 事件驱动架构 - 领域事件和异步事件处理
/// 
/// 架构特征：
/// - 领域事件发布和订阅
/// - 异步事件处理
/// - 事件存储和重放
/// - 事件版本控制
/// - 分布式事件传播

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock, Mutex};
use uuid::Uuid;
use chrono::{DateTime, Utc};
use crate::domain::{DomainResult, DomainError, UserId, GroupId};

// === 事件核心定义 ===

/// 事件ID - 全局唯一标识符
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct EventId(Uuid);

impl EventId {
    /// 生成新的事件ID
    pub fn new() -> Self {
        Self(Uuid::new_v4())
    }
    
    /// 从UUID创建事件ID
    pub fn from_uuid(uuid: Uuid) -> Self {
        Self(uuid)
    }
    
    /// 从字符串创建事件ID
    pub fn from_string(s: String) -> Self {
        Self(Uuid::parse_str(&s).unwrap_or_else(|_| Uuid::new_v4()))
    }
    
    /// 获取UUID值
    pub fn value(&self) -> Uuid {
        self.0
    }
}

impl Default for EventId {
    fn default() -> Self {
        Self::new()
    }
}

/// 事件元数据 - 包含事件的上下文信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventMetadata {
    /// 事件ID
    pub event_id: EventId,
    /// 事件类型
    pub event_type: String,
    /// 事件版本
    pub version: u32,
    /// 聚合根ID
    pub aggregate_id: String,
    /// 聚合根类型
    pub aggregate_type: String,
    /// 事件发生时间
    pub occurred_at: DateTime<Utc>,
    /// 因果关系ID（用于追踪事件链）
    pub causation_id: Option<EventId>,
    /// 相关性ID（用于关联相关事件）
    pub correlation_id: Option<String>,
    /// 用户ID（如果事件与用户相关）
    pub user_id: Option<UserId>,
    /// 会话ID
    pub session_id: Option<String>,
    /// 额外的元数据
    pub metadata: HashMap<String, String>,
}

impl EventMetadata {
    /// 创建新的事件元数据
    pub fn new(
        event_type: &str,
        aggregate_id: &str,
        aggregate_type: &str,
    ) -> Self {
        Self {
            event_id: EventId::new(),
            event_type: event_type.to_string(),
            version: 1,
            aggregate_id: aggregate_id.to_string(),
            aggregate_type: aggregate_type.to_string(),
            occurred_at: Utc::now(),
            causation_id: None,
            correlation_id: None,
            user_id: None,
            session_id: None,
            metadata: HashMap::new(),
        }
    }
    
    /// 设置因果关系ID
    pub fn with_causation_id(mut self, causation_id: EventId) -> Self {
        self.causation_id = Some(causation_id);
        self
    }
    
    /// 设置相关性ID
    pub fn with_correlation_id(mut self, correlation_id: String) -> Self {
        self.correlation_id = Some(correlation_id);
        self
    }
    
    /// 设置用户ID
    pub fn with_user_id(mut self, user_id: UserId) -> Self {
        self.user_id = Some(user_id);
        self
    }
    
    /// 添加元数据
    pub fn with_metadata(mut self, key: &str, value: &str) -> Self {
        self.metadata.insert(key.to_string(), value.to_string());
        self
    }
}

/// 领域事件特征 - 所有领域事件必须实现此特征
pub trait DomainEvent: Send + Sync + std::fmt::Debug {
    /// 获取事件类型名称
    fn event_type(&self) -> &'static str;
    
    /// 获取聚合根ID
    fn aggregate_id(&self) -> String;
    
    /// 获取聚合根类型
    fn aggregate_type(&self) -> &'static str;
    
    /// 序列化事件数据
    fn serialize(&self) -> DomainResult<Vec<u8>>;
    
    /// 反序列化事件数据
    fn deserialize(data: &[u8]) -> DomainResult<Box<dyn DomainEvent>>
    where
        Self: Sized;
    
    /// 获取事件版本
    fn version(&self) -> u32 {
        1
    }
}

/// 事件包装器 - 包含事件和元数据
#[derive(Debug, Clone)]
pub struct EventEnvelope {
    /// 事件元数据
    pub metadata: EventMetadata,
    /// 序列化的事件数据
    pub data: Vec<u8>,
    /// 事件的原始类型信息
    pub event_type_info: String,
}

impl EventEnvelope {
    /// 创建新的事件包装器
    pub fn new(event: Box<dyn DomainEvent>, metadata: EventMetadata) -> DomainResult<Self> {
        let data = event.serialize()?;
        let event_type_info = event.event_type().to_string();
        
        Ok(Self {
            metadata,
            data,
            event_type_info,
        })
    }
    
    /// 从事件包装器中提取事件
    pub fn extract_event<T>(&self) -> DomainResult<T>
    where
        T: DomainEvent + 'static,
    {
        // 这里需要一个事件注册表来反序列化具体的事件类型
        // 简化实现，实际项目中需要更复杂的类型注册机制
        todo!("需要实现事件类型注册表")
    }
}

// === 具体的领域事件定义 ===

/// 用户注册事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserRegisteredEvent {
    pub user_id: UserId,
    pub login_id: String,
    pub nickname: String,
    pub email: Option<String>,
    pub user_type: crate::domain::UserType,
    pub registered_at: DateTime<Utc>,
}

impl DomainEvent for UserRegisteredEvent {
    fn event_type(&self) -> &'static str {
        "UserRegistered"
    }
    
    fn aggregate_id(&self) -> String {
        self.user_id.value().to_string()
    }
    
    fn aggregate_type(&self) -> &'static str {
        "User"
    }
    
    fn serialize(&self) -> DomainResult<Vec<u8>> {
        serde_json::to_vec(self).map_err(|e| {
            DomainError::ValidationError {
                field: "event_serialization".to_string(),
                reason: e.to_string(),
            }
        })
    }
    
    fn deserialize(data: &[u8]) -> DomainResult<Box<dyn DomainEvent>> {
        let event: UserRegisteredEvent = serde_json::from_slice(data).map_err(|e| {
            DomainError::ValidationError {
                field: "event_deserialization".to_string(),
                reason: e.to_string(),
            }
        })?;
        Ok(Box::new(event))
    }
}

/// 群组创建事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupCreatedEvent {
    pub group_id: GroupId,
    pub name: String,
    pub creator_id: UserId,
    pub group_type: String, // 简化为字符串，避免复杂的类型依赖
    pub created_at: DateTime<Utc>,
}

impl DomainEvent for GroupCreatedEvent {
    fn event_type(&self) -> &'static str {
        "GroupCreated"
    }
    
    fn aggregate_id(&self) -> String {
        self.group_id.value().to_string()
    }
    
    fn aggregate_type(&self) -> &'static str {
        "Group"
    }
    
    fn serialize(&self) -> DomainResult<Vec<u8>> {
        serde_json::to_vec(self).map_err(|e| {
            DomainError::ValidationError {
                field: "event_serialization".to_string(),
                reason: e.to_string(),
            }
        })
    }
    
    fn deserialize(data: &[u8]) -> DomainResult<Box<dyn DomainEvent>> {
        let event: GroupCreatedEvent = serde_json::from_slice(data).map_err(|e| {
            DomainError::ValidationError {
                field: "event_deserialization".to_string(),
                reason: e.to_string(),
            }
        })?;
        Ok(Box::new(event))
    }
}

/// 用户加入群组事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserJoinedGroupEvent {
    pub user_id: UserId,
    pub group_id: GroupId,
    pub role: String,
    pub joined_at: DateTime<Utc>,
}

impl DomainEvent for UserJoinedGroupEvent {
    fn event_type(&self) -> &'static str {
        "UserJoinedGroup"
    }
    
    fn aggregate_id(&self) -> String {
        format!("{}:{}", self.user_id.value(), self.group_id.value())
    }
    
    fn aggregate_type(&self) -> &'static str {
        "GroupMembership"
    }
    
    fn serialize(&self) -> DomainResult<Vec<u8>> {
        serde_json::to_vec(self).map_err(|e| {
            DomainError::ValidationError {
                field: "event_serialization".to_string(),
                reason: e.to_string(),
            }
        })
    }
    
    fn deserialize(data: &[u8]) -> DomainResult<Box<dyn DomainEvent>> {
        let event: UserJoinedGroupEvent = serde_json::from_slice(data).map_err(|e| {
            DomainError::ValidationError {
                field: "event_deserialization".to_string(),
                reason: e.to_string(),
            }
        })?;
        Ok(Box::new(event))
    }
}

// === 事件处理器特征 ===

/// 事件处理器特征 - 处理特定类型的事件
#[async_trait]
pub trait EventHandler<T: DomainEvent>: Send + Sync {
    /// 处理事件
    async fn handle(&self, event: &T, metadata: &EventMetadata) -> DomainResult<()>;
    
    /// 获取处理器名称
    fn name(&self) -> &str;
    
    /// 是否支持重试
    fn supports_retry(&self) -> bool {
        true
    }
    
    /// 最大重试次数
    fn max_retries(&self) -> u32 {
        3
    }
}

/// 通用事件处理器特征 - 处理任意类型的事件
#[async_trait]
pub trait GenericEventHandler: Send + Sync {
    /// 处理事件包装器
    async fn handle(&self, envelope: &EventEnvelope) -> DomainResult<()>;
    
    /// 获取处理器名称
    fn name(&self) -> &str;
    
    /// 获取支持的事件类型
    fn supported_event_types(&self) -> Vec<&'static str>;
}

// === 事件存储特征 ===

/// 事件存储特征 - 持久化事件
#[async_trait]
pub trait EventStore: Send + Sync {
    /// 保存事件
    async fn save_event(&self, envelope: EventEnvelope) -> DomainResult<()>;
    
    /// 批量保存事件
    async fn save_events(&self, envelopes: Vec<EventEnvelope>) -> DomainResult<()>;
    
    /// 根据聚合根ID获取事件流
    async fn get_events_for_aggregate(
        &self,
        aggregate_id: &str,
        from_version: Option<u32>,
    ) -> DomainResult<Vec<EventEnvelope>>;
    
    /// 获取所有事件（用于重放）
    async fn get_all_events(
        &self,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>>;
    
    /// 根据事件类型获取事件
    async fn get_events_by_type(
        &self,
        event_type: &str,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>>;
}

// === 事件发布器 ===

/// 事件发布器 - 负责事件的发布和分发
#[derive(Clone)]
pub struct EventPublisher {
    /// 事件存储
    event_store: Arc<dyn EventStore>,
    /// 事件处理器注册表
    handlers: Arc<RwLock<HashMap<String, Vec<Arc<dyn GenericEventHandler>>>>>,
    /// 异步事件通道
    event_sender: mpsc::UnboundedSender<EventEnvelope>,
    /// 事件处理统计
    stats: Arc<RwLock<EventStats>>,
}

/// 事件处理统计
#[derive(Debug, Default)]
pub struct EventStats {
    pub total_published: u64,
    pub total_processed: u64,
    pub total_failed: u64,
    pub processing_times: HashMap<String, Vec<u64>>, // 事件类型 -> 处理时间列表（毫秒）
}

impl EventPublisher {
    /// 创建新的事件发布器
    pub fn new(event_store: Arc<dyn EventStore>) -> Self {
        let (event_sender, event_receiver) = mpsc::unbounded_channel();
        let handlers = Arc::new(RwLock::new(HashMap::new()));
        let stats = Arc::new(RwLock::new(EventStats::default()));
        
        let publisher = Self {
            event_store,
            handlers: handlers.clone(),
            event_sender,
            stats: stats.clone(),
        };
        
        // 启动事件处理循环
        publisher.start_event_processing_loop(event_receiver, handlers, stats);
        
        publisher
    }
    
    /// 发布事件
    pub async fn publish(&self, event: Box<dyn DomainEvent>) -> DomainResult<EventId> {
        let metadata = EventMetadata::new(
            event.event_type(),
            &event.aggregate_id(),
            event.aggregate_type(),
        );
        
        let event_id = metadata.event_id.clone();
        let envelope = EventEnvelope::new(event, metadata)?;
        
        // 保存到事件存储
        self.event_store.save_event(envelope.clone()).await?;
        
        // 发送到异步处理通道
        self.event_sender.send(envelope).map_err(|e| {
            DomainError::BusinessRuleViolation {
                rule: format!("Failed to send event to processing channel: {}", e),
            }
        })?;
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_published += 1;
        }
        
        Ok(event_id)
    }
    
    /// 批量发布事件
    pub async fn publish_batch(&self, events: Vec<Box<dyn DomainEvent>>) -> DomainResult<Vec<EventId>> {
        let mut envelopes = Vec::new();
        let mut event_ids = Vec::new();
        
        for event in events {
            let metadata = EventMetadata::new(
                event.event_type(),
                &event.aggregate_id(),
                event.aggregate_type(),
            );
            
            event_ids.push(metadata.event_id.clone());
            let envelope = EventEnvelope::new(event, metadata)?;
            envelopes.push(envelope);
        }
        
        // 批量保存到事件存储
        self.event_store.save_events(envelopes.clone()).await?;
        
        // 发送到异步处理通道
        for envelope in envelopes {
            self.event_sender.send(envelope).map_err(|e| {
                DomainError::BusinessRuleViolation {
                    rule: format!("Failed to send event to processing channel: {}", e),
                }
            })?;
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_published += event_ids.len() as u64;
        }
        
        Ok(event_ids)
    }
    
    /// 注册事件处理器
    pub async fn register_handler(&self, handler: Arc<dyn GenericEventHandler>) {
        let mut handlers = self.handlers.write().await;
        
        for event_type in handler.supported_event_types() {
            handlers
                .entry(event_type.to_string())
                .or_insert_with(Vec::new)
                .push(handler.clone());
        }
        
        tracing::info!(
            handler_name = handler.name(),
            supported_types = ?handler.supported_event_types(),
            "Event handler registered"
        );
    }
    
    /// 启动事件处理循环
    fn start_event_processing_loop(
        &self,
        mut event_receiver: mpsc::UnboundedReceiver<EventEnvelope>,
        handlers: Arc<RwLock<HashMap<String, Vec<Arc<dyn GenericEventHandler>>>>>,
        stats: Arc<RwLock<EventStats>>,
    ) {
        tokio::spawn(async move {
            while let Some(envelope) = event_receiver.recv().await {
                let start_time = std::time::Instant::now();
                let event_type = envelope.metadata.event_type.clone();
                
                // 获取该事件类型的处理器
                let event_handlers = {
                    let handlers_guard = handlers.read().await;
                    handlers_guard
                        .get(&event_type)
                        .cloned()
                        .unwrap_or_default()
                };
                
                // 并行处理所有处理器
                let mut tasks = Vec::new();
                for handler in event_handlers {
                    let envelope_clone = envelope.clone();
                    let handler_clone = handler.clone();
                    
                    let task = tokio::spawn(async move {
                        match handler_clone.handle(&envelope_clone).await {
                            Ok(()) => {
                                tracing::debug!(
                                    handler_name = handler_clone.name(),
                                    event_type = %envelope_clone.metadata.event_type,
                                    event_id = %envelope_clone.metadata.event_id.value(),
                                    "Event processed successfully"
                                );
                                Ok(())
                            }
                            Err(e) => {
                                tracing::error!(
                                    handler_name = handler_clone.name(),
                                    event_type = %envelope_clone.metadata.event_type,
                                    event_id = %envelope_clone.metadata.event_id.value(),
                                    error = %e,
                                    "Event processing failed"
                                );
                                Err(e)
                            }
                        }
                    });
                    
                    tasks.push(task);
                }
                
                // 等待所有处理器完成
                let results = futures::future::join_all(tasks).await;
                
                // 更新统计
                let processing_time = start_time.elapsed().as_millis() as u64;
                let mut stats_guard = stats.write().await;
                
                let mut success_count = 0;
                let mut failure_count = 0;
                
                for result in results {
                    match result {
                        Ok(Ok(())) => success_count += 1,
                        Ok(Err(_)) | Err(_) => failure_count += 1,
                    }
                }
                
                stats_guard.total_processed += success_count;
                stats_guard.total_failed += failure_count;
                stats_guard
                    .processing_times
                    .entry(event_type)
                    .or_insert_with(Vec::new)
                    .push(processing_time);
            }
        });
    }
    
    /// 获取事件处理统计
    pub async fn get_stats(&self) -> EventStats {
        self.stats.read().await.clone()
    }
}

// === 内存事件存储实现（用于测试和开发） ===

/// 内存事件存储 - 简单的内存实现
#[derive(Default)]
pub struct InMemoryEventStore {
    events: Arc<Mutex<Vec<EventEnvelope>>>,
}

impl InMemoryEventStore {
    /// 创建新的内存事件存储
    pub fn new() -> Self {
        Self {
            events: Arc::new(Mutex::new(Vec::new())),
        }
    }
}

#[async_trait]
impl EventStore for InMemoryEventStore {
    async fn save_event(&self, envelope: EventEnvelope) -> DomainResult<()> {
        let mut events = self.events.lock().await;
        events.push(envelope);
        Ok(())
    }
    
    async fn save_events(&self, envelopes: Vec<EventEnvelope>) -> DomainResult<()> {
        let mut events = self.events.lock().await;
        events.extend(envelopes);
        Ok(())
    }
    
    async fn get_events_for_aggregate(
        &self,
        aggregate_id: &str,
        from_version: Option<u32>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        let events = self.events.lock().await;
        let filtered_events: Vec<EventEnvelope> = events
            .iter()
            .filter(|e| e.metadata.aggregate_id == aggregate_id)
            .filter(|e| {
                if let Some(min_version) = from_version {
                    e.metadata.version >= min_version
                } else {
                    true
                }
            })
            .cloned()
            .collect();
        
        Ok(filtered_events)
    }
    
    async fn get_all_events(
        &self,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        let events = self.events.lock().await;
        let mut filtered_events: Vec<EventEnvelope> = events
            .iter()
            .filter(|e| {
                if let Some(min_time) = from_timestamp {
                    e.metadata.occurred_at >= min_time
                } else {
                    true
                }
            })
            .cloned()
            .collect();
        
        // 按时间排序
        filtered_events.sort_by(|a, b| a.metadata.occurred_at.cmp(&b.metadata.occurred_at));
        
        // 应用限制
        if let Some(max_count) = limit {
            filtered_events.truncate(max_count);
        }
        
        Ok(filtered_events)
    }
    
    async fn get_events_by_type(
        &self,
        event_type: &str,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        let events = self.events.lock().await;
        let mut filtered_events: Vec<EventEnvelope> = events
            .iter()
            .filter(|e| e.metadata.event_type == event_type)
            .filter(|e| {
                if let Some(min_time) = from_timestamp {
                    e.metadata.occurred_at >= min_time
                } else {
                    true
                }
            })
            .cloned()
            .collect();
        
        // 按时间排序
        filtered_events.sort_by(|a, b| a.metadata.occurred_at.cmp(&b.metadata.occurred_at));
        
        // 应用限制
        if let Some(max_count) = limit {
            filtered_events.truncate(max_count);
        }
        
        Ok(filtered_events)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_id_creation() {
        let event_id = EventId::new();
        assert!(!event_id.value().is_nil());
        
        let uuid = Uuid::new_v4();
        let event_id2 = EventId::from_uuid(uuid);
        assert_eq!(event_id2.value(), uuid);
    }
    
    #[test]
    fn test_event_metadata_creation() {
        let metadata = EventMetadata::new("TestEvent", "123", "TestAggregate")
            .with_correlation_id("corr-123".to_string())
            .with_metadata("key1", "value1");
        
        assert_eq!(metadata.event_type, "TestEvent");
        assert_eq!(metadata.aggregate_id, "123");
        assert_eq!(metadata.aggregate_type, "TestAggregate");
        assert_eq!(metadata.correlation_id, Some("corr-123".to_string()));
        assert_eq!(metadata.metadata.get("key1"), Some(&"value1".to_string()));
    }
    
    #[tokio::test]
    async fn test_in_memory_event_store() {
        let store = InMemoryEventStore::new();
        
        let metadata = EventMetadata::new("TestEvent", "123", "TestAggregate");
        let envelope = EventEnvelope {
            metadata,
            data: vec![1, 2, 3],
            event_type_info: "TestEvent".to_string(),
        };
        
        // 保存事件
        store.save_event(envelope.clone()).await.unwrap();
        
        // 获取事件
        let events = store.get_events_for_aggregate("123", None).await.unwrap();
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].metadata.aggregate_id, "123");
    }
}
