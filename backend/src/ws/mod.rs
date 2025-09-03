pub mod constants;
pub mod errors;
pub mod connection_manager;
pub mod room_manager;
pub mod message_router;
pub mod handler;
pub mod message_types;
pub mod manager;
pub mod types;

// 重新导出主要类型
pub use connection_manager::{ConnectionManager, ConnectionStats};
pub use room_manager::{RoomManager, RoomStats};
pub use message_router::MessageRouter;
pub use manager::WebSocketStats;
pub use message_types::{ServerMessage, ClientMessage, RoomId, RoomType};
pub use errors::WsError;

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
