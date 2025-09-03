//! 通知端口（抽象接口），供 application 层依赖

use serde_json::Value;
use uuid::Uuid;
use std::future::Future;
use std::pin::Pin;

#[derive(Debug, thiserror::Error)]
pub enum NotificationPortError {
    #[error("transport error: {0}")]
    Transport(String),
    #[error("cache error: {0}")]
    Cache(String),
    #[error("internal error: {0}")]
    Internal(String),
}

pub type NotificationPortResult<T> = Result<T, NotificationPortError>;

pub trait NotificationPort: Send + Sync {
    fn name(&self) -> &str;

    fn send_to_user<'a>(
        &'a self,
        user_id: i64,
        payload: &'a Value,
    ) -> Pin<Box<dyn Future<Output = NotificationPortResult<bool>> + Send + 'a>>;

    fn store_offline<'a>(
        &'a self,
        user_id: i64,
        payload: &'a Value,
        expire_days: u32,
    ) -> Pin<Box<dyn Future<Output = NotificationPortResult<()>> + Send + 'a>>;

    fn send_group_digest<'a>(
        &'a self,
        user_id: i64,
        user_nickname: &'a str,
        group_id: Uuid,
        group_name: &'a str,
        digest_content: &'a str,
    ) -> Pin<Box<dyn Future<Output = NotificationPortResult<bool>> + Send + 'a>>;
}


