/// CQRS处理器 - 命令和查询的处理逻辑
/// 
/// 处理器特征：
/// - 单一职责
/// - 无状态
/// - 可测试
/// - 事务边界清晰

use async_trait::async_trait;
use std::sync::Arc;
use crate::application::ApplicationResult;
use crate::domain::{DomainEvent, User, Group, MessageId, MomentId};

// === 处理器特征定义 ===

/// 命令处理器特征
#[async_trait]
pub trait CommandHandler<TCommand>: Send + Sync {
    /// 处理结果类型
    type Result;
    
    /// 处理命令
    async fn handle(&self, command: TCommand) -> ApplicationResult<Self::Result>;
}

/// 查询处理器特征
#[async_trait]
pub trait QueryHandler<TQuery>: Send + Sync {
    /// 处理结果类型
    type Result;
    
    /// 处理查询
    async fn handle(&self, query: TQuery) -> ApplicationResult<Self::Result>;
}

/// 事件处理器特征
#[async_trait]
pub trait EventHandler<TEvent>: Send + Sync {
    /// 处理事件
    async fn handle(&self, event: TEvent) -> ApplicationResult<()>;
}

// === 命令处理结果 ===

/// 用户创建结果
#[derive(Debug, Clone)]
pub struct UserCreated {
    pub user: User,
    pub events: Vec<DomainEvent>,
}

/// 用户更新结果
#[derive(Debug, Clone)]
pub struct UserUpdated {
    pub user: User,
    pub events: Vec<DomainEvent>,
}

/// 群组创建结果
#[derive(Debug, Clone)]
pub struct GroupCreated {
    pub group: Group,
    pub events: Vec<DomainEvent>,
}

/// 群组更新结果
#[derive(Debug, Clone)]
pub struct GroupUpdated {
    pub group: Group,
    pub events: Vec<DomainEvent>,
}

/// 消息发送结果
#[derive(Debug, Clone)]
pub struct MessageSent {
    pub message_id: MessageId,
    pub events: Vec<DomainEvent>,
}

/// 瞬间创建结果
#[derive(Debug, Clone)]
pub struct MomentCreated {
    pub moment_id: MomentId,
    pub events: Vec<DomainEvent>,
}

/// 通用操作结果
#[derive(Debug, Clone)]
pub struct OperationResult {
    pub success: bool,
    pub events: Vec<DomainEvent>,
}

// === 分页相关类型 ===

/// 分页元数据
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginationMeta {
    /// 是否有更多数据
    pub has_more: bool,
    /// 下一页游标
    pub next_cursor: Option<i64>,
    /// 当前页大小
    pub page_size: usize,
    /// 总数（可选）
    pub total_count: Option<u64>,
}

impl PaginationMeta {
    /// 创建新的分页元数据
    pub fn new(has_more: bool, next_cursor: Option<i64>, page_size: usize) -> Self {
        Self {
            has_more,
            next_cursor,
            page_size,
            total_count: None,
        }
    }
    
    /// 设置总数
    pub fn with_total_count(mut self, total_count: u64) -> Self {
        self.total_count = Some(total_count);
        self
    }
}

// === 查询结果视图模型 ===

/// 用户视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UserView {
    pub user_id: String, // JS安全的字符串格式
    pub login_id: Option<String>,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub email: Option<String>,
    pub bio: Option<String>,
    pub is_temp: bool,
    pub created_at: String,
    pub last_active_at: String,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub distance: Option<f64>, // 查询时计算的距离
}

/// 群组视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct GroupView {
    pub group_id: String, // UUID字符串
    pub name: String,
    pub description: Option<String>,
    pub owner_id: String,
    pub creator_name: String,
    pub created_at: String,
    pub last_active_at: String,
    pub latitude: f64,
    pub longitude: f64,
    pub member_count: u32,
    pub distance: Option<f64>,
    pub location_name: String,
    pub is_password_required: bool,
    pub is_member: bool,
    pub user_role: Option<String>,
    pub avatar_url: Option<String>,
}

/// 消息视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct MessageView {
    pub message_id: String, // UUID字符串
    pub sender_id: String,
    pub sender_name: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: String,
    pub group_id: Option<String>,
    pub conversation_id: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub is_read: bool,
    pub read_at: Option<String>,
}

/// 瞬间视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct MomentView {
    pub moment_id: String, // UUID字符串
    pub user_id: String,
    pub nickname: String,
    pub user_avatar: Option<String>,
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: String,
    pub likes_count: u32,
    pub comments_count: u32,
    pub liked_by_me: bool,
    pub distance: Option<f64>,
    pub tags: Vec<String>,
    pub image_urls: Vec<String>,
}

/// 会话概要视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConversationView {
    pub conversation_id: String, // UUID字符串
    pub peer_user_id: String,
    pub peer_nickname: String,
    pub peer_avatar: Option<String>,
    pub last_message_preview: String,
    pub last_message_at: String,
    pub unread_count: u32,
}

/// 分页结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginatedResult<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    pub next_cursor: Option<String>, // JS安全的字符串格式
}

impl<T> PaginatedResult<T> {
    /// 创建分页结果
    pub fn new(items: Vec<T>, has_more: bool, next_cursor: Option<i64>) -> Self {
        Self {
            items,
            has_more,
            next_cursor: next_cursor.map(|c| c.to_string()),
        }
    }
    
    /// 创建空结果
    pub fn empty() -> Self {
        Self {
            items: Vec::new(),
            has_more: false,
            next_cursor: None,
        }
    }
}

/// 未读计数结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UnreadCountResult {
    pub total_unread_count: u32,
    pub group_unread_counts: Vec<GroupUnreadCount>,
    pub conversation_unread_counts: Vec<ConversationUnreadCount>,
}

/// 群组未读计数
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct GroupUnreadCount {
    pub group_id: String,
    pub unread_count: u32,
}

/// 会话未读计数
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ConversationUnreadCount {
    pub conversation_id: String,
    pub unread_count: u32,
}

/// 用户设置视图模型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UserSettingsView {
    pub notification_level: String,
    pub message_notifications: bool,
    pub group_notifications: bool,
    pub system_notifications: bool,
    pub location_notifications: bool,
    pub quiet_hours_enabled: bool,
    pub quiet_hours_start: Option<String>,
    pub quiet_hours_end: Option<String>,
    pub notification_radius_km: f64,
    pub sound_enabled: bool,
    pub vibration_enabled: bool,
}

/// 密码状态结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PasswordStatusResult {
    pub has_password: bool,
}

/// 统计信息结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct StatsResult {
    pub total_users: u64,
    pub active_users_24h: u64,
    pub total_groups: u64,
    pub active_groups_24h: u64,
    pub total_messages_24h: u64,
    pub total_moments_24h: u64,
}

// === 处理器注册表 ===

/// 处理器注册表 - 管理所有处理器的注册和查找
pub struct HandlerRegistry {
    // 这里可以存储处理器的映射关系
    // 实际实现中可能使用依赖注入容器
}

impl HandlerRegistry {
    /// 创建新的处理器注册表
    pub fn new() -> Self {
        Self {}
    }
    
    /// 注册命令处理器
    pub fn register_command_handler<TCommand, TResult, THandler>(
        &mut self,
        _handler: Arc<THandler>,
    ) where
        THandler: CommandHandler<TCommand> + 'static,
        TCommand: 'static,
        TResult: 'static,
    {
        // 实际实现中会将处理器存储到映射表中
    }
    
    /// 注册查询处理器
    pub fn register_query_handler<TQuery, TResult, THandler>(
        &mut self,
        _handler: Arc<THandler>,
    ) where
        THandler: QueryHandler<TQuery> + 'static,
        TQuery: 'static,
        TResult: 'static,
    {
        // 实际实现中会将处理器存储到映射表中
    }
    
    /// 注册事件处理器
    pub fn register_event_handler<TEvent, THandler>(
        &mut self,
        _handler: Arc<THandler>,
    ) where
        THandler: EventHandler<TEvent> + 'static,
        TEvent: 'static,
    {
        // 实际实现中会将处理器存储到映射表中
    }
}

impl Default for HandlerRegistry {
    fn default() -> Self {
        Self::new()
    }
}

// === 处理器基础实现 ===

/// 抽象命令处理器基类
pub struct BaseCommandHandler {
    // 可以包含通用的依赖，如事件发布器、日志记录器等
}

impl BaseCommandHandler {
    /// 创建新的基础命令处理器
    pub fn new() -> Self {
        Self {}
    }
    
    /// 发布领域事件
    pub async fn publish_events(&self, events: Vec<DomainEvent>) -> ApplicationResult<()> {
        // 实际实现中会通过事件发布器发布事件
        for event in events {
            tracing::info!("Publishing event: {:?}", event.event_type());
        }
        Ok(())
    }
    
    /// 验证请求幂等性
    pub async fn ensure_idempotency(&self, request_id: &str) -> ApplicationResult<bool> {
        // 实际实现中会检查请求ID是否已经处理过
        // 这里简单返回false表示未处理过
        tracing::debug!("Checking idempotency for request: {}", request_id);
        Ok(false)
    }
}

impl Default for BaseCommandHandler {
    fn default() -> Self {
        Self::new()
    }
}

/// 抽象查询处理器基类
pub struct BaseQueryHandler {
    // 可以包含通用的依赖，如缓存、只读数据库连接等
}

impl BaseQueryHandler {
    /// 创建新的基础查询处理器
    pub fn new() -> Self {
        Self {}
    }
    
    /// 从缓存获取结果
    pub async fn get_from_cache<T>(&self, cache_key: &str) -> ApplicationResult<Option<T>>
    where
        T: serde::de::DeserializeOwned,
    {
        // 实际实现中会从缓存中获取数据
        tracing::debug!("Checking cache for key: {}", cache_key);
        Ok(None)
    }
    
    /// 设置缓存
    pub async fn set_cache<T>(&self, cache_key: &str, _value: &T, ttl: u32) -> ApplicationResult<()>
    where
        T: serde::Serialize,
    {
        // 实际实现中会将数据存储到缓存
        tracing::debug!("Setting cache for key: {} with TTL: {}s", cache_key, ttl);
        Ok(())
    }
}

impl Default for BaseQueryHandler {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_paginated_result() {
        let items = vec!["item1".to_string(), "item2".to_string()];
        let result = PaginatedResult::new(items.clone(), true, Some(123));
        
        assert_eq!(result.items, items);
        assert!(result.has_more);
        assert_eq!(result.next_cursor, Some("123".to_string()));
        
        let empty_result: PaginatedResult<String> = PaginatedResult::empty();
        assert!(empty_result.items.is_empty());
        assert!(!empty_result.has_more);
        assert!(empty_result.next_cursor.is_none());
    }
    
    #[test]
    fn test_handler_registry() {
        let _registry = HandlerRegistry::new();
        
        // 测试注册表创建
        // 实际测试中会验证处理器的注册和查找
        let _base_handler = BaseCommandHandler::new();
        
        // 这里只是验证API的可用性
        assert!(true);
    }
}
