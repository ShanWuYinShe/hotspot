//! 通知服务接口
use serde_json::Value;
use std::future::Future;
use uuid::Uuid;

// 导出子模块
pub mod group_notification;
pub mod manager;
pub mod policy;
pub mod websocket_notification;

// 从子模块中重新导出具体实现
pub use group_notification::GroupNotificationService;
pub use manager::NotificationManager;
pub use policy::{GroupNotificationPolicy, NotificationPolicy, NotificationPolicyService};
pub use websocket_notification::WebSocketNotificationService;

/// 通知错误类型
#[derive(Debug, thiserror::Error)]
pub enum NotificationError {
    #[error("WebSocket错误: {0}")]
    WebSocket(String),
    #[error("缓存错误: {0}")]
    Cache(#[from] crate::cache::CacheError),
    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),
    #[error("仓库错误: {0}")]
    Repository(#[from] crate::repositories::RepositoryError),
    #[allow(dead_code)]
    #[error("内部错误: {0}")]
    Internal(String),
    #[allow(dead_code)]
    #[error("通知服务不可用")]
    ServiceUnavailable,
    #[allow(dead_code)]
    #[error("用户不在线")]
    UserOffline,
    #[error("其他通知错误: {0}")]
    #[allow(dead_code)]
    Other(String),
}

/// 通知结果
pub type NotificationResult<T> = Result<T, NotificationError>;

/// 通知服务接口
#[allow(dead_code)]
pub trait NotificationService: Send + Sync {
    /// 获取服务名称
    #[allow(dead_code)]
    fn name(&self) -> &str;

    /// 发送通知
    #[allow(dead_code)]
    fn send_notification(
        &self,
        user_id: i64,
        notification: &Value,
    ) -> impl Future<Output = NotificationResult<bool>> + Send;

    /// 发送离线通知
    #[allow(dead_code)]
    fn store_offline_notification(
        &self,
        user_id: i64,
        notification: &Value,
        expire_days: u32,
    ) -> impl Future<Output = NotificationResult<()>> + Send;

    /// 发送群组摘要通知
    #[allow(dead_code)]
    fn send_group_digest(
        &self,
        user_id: i64,
        user_nickname: &str,
        group_id: Uuid,
        group_name: &str,
        digest_content: &str,
    ) -> impl Future<Output = NotificationResult<bool>> + Send;
}
