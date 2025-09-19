/// HotSpot WebSocket 实时通信系统
/// 
/// 功能职责：
/// - 提供完整的 WebSocket 实时通信解决方案
/// - 实现连接管理、消息路由、房间管理等核心功能
/// - 支持多种消息类型和通信模式
/// - 提供高性能的实时数据推送和状态同步
/// - 实现智能通知策略和精准消息分发
/// 
/// 系统架构：
/// - 连接管理：WebSocket 连接的生命周期管理
/// - 房间管理：基于房间的消息分组和订阅管理
/// - 消息路由：智能消息路由和分发机制
/// - 通知策略：基于用户关系的精准通知推送
/// - 性能优化：连接池、消息批处理、内存管理
/// 
/// 核心模块：
/// - manager: WebSocket 管理器和核心协调逻辑
/// - handler: WebSocket 连接处理和协议实现
/// - connection_manager: 连接池管理和用户映射
/// - room_manager: 房间管理和订阅控制
/// - message_router: 消息路由和分发策略
/// - message_types: 消息类型定义和序列化
/// - notification_policy: 智能通知策略和用户筛选
/// 
/// 技术特性：
/// - 支持大规模并发连接和高吞吐量消息处理
/// - 实现连接的自动重连、心跳检测、超时管理
/// - 提供消息的可靠性保证和去重机制
/// - 支持房间的动态创建、订阅、取消订阅
/// - 实现基于用户关系的智能通知分发
pub mod constants;
pub mod errors;
pub mod connection_manager;
pub mod room_manager;
pub mod message_router;
pub mod handler;
pub mod message_types;
pub mod manager;
pub mod types;
pub mod connection_optimizer;
pub mod notification_policy;
pub mod performance_optimizer;

// 新的统一实现模块
pub mod unified_message_types;
pub mod unified_connection_manager;

// 重新导出主要类型
pub use connection_manager::{ConnectionManager, ConnectionStats};
pub use room_manager::{RoomManager, RoomStats};
pub use message_router::MessageRouter;
pub use manager::WebSocketStats;
pub use message_types::{ServerMessage, ClientMessage, RoomId, RoomType};
pub use errors::WsError;
pub use notification_policy::{
    NotificationPolicyService, NotificationPolicy, NotificationPolicyConfig,
    NotificationResult, SmartNotificationScheduler, MessagePriority
};
pub use performance_optimizer::{
    WebSocketPerformanceOptimizer, PerformanceStats, OptimizationConfig,
    PerformanceOptimizationSuggestion, ConnectionHealth
};

// 重新导出统一实现的类型
pub use unified_message_types::{
    WebSocketMessage, ClientMessage as UnifiedClientMessage, ServerMessage as UnifiedServerMessage,
    ConnectionInfo, RoomInfo, RoomType as UnifiedRoomType, MessageRoute, MessagePriority as UnifiedMessagePriority,
    WebSocketError, ConnectionStats as UnifiedConnectionStats, RoomStats as UnifiedRoomStats
};
pub use unified_connection_manager::UnifiedConnectionManager;

/// WebSocket 扩展钩子（默认 Noop）。
/// 使用 feature gate `ws_hooks` 启用具体实现。
#[cfg_attr(not(feature = "ws_hooks"), allow(dead_code))]
pub mod hooks {
    use uuid::Uuid;
    use std::sync::Arc;

    /// WebSocket 生命周期与订阅事件钩子
    pub trait WsHooks: Send + Sync {
        fn on_subscribe(&self, _connection_id: Uuid, _room: &crate::ws::message_types::RoomId) {}
        fn on_unsubscribe(&self, _connection_id: Uuid, _room: &crate::ws::message_types::RoomId) {}
        fn on_disconnect(&self, _connection_id: Uuid) {}
    }

    /// 默认空实现
    #[derive(Clone, Debug, Default)]
    pub struct NoopWsHooks;
    impl WsHooks for NoopWsHooks {}

    /// 钩子服务容器，便于未来扩展为多路分发
    #[derive(Clone)]
    pub struct WsHookService<H: WsHooks + ?Sized> {
        inner: Arc<H>,
    }
    impl<H: WsHooks + ?Sized> WsHookService<H> {
        pub fn new(inner: Arc<H>) -> Self { Self { inner } }
        pub fn inner(&self) -> &Arc<H> { &self.inner }
    }
}
