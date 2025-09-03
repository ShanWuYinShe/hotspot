//! WebSocket 端口定义
//! 
//! 定义 WebSocket 通信的抽象接口，支持不同的实时通信实现

use std::future::Future;
use std::pin::Pin;
use uuid::Uuid;
use thiserror::Error;

use crate::ws::message_types::{ServerMessage, SystemNotificationData};

/// WebSocket 端口错误类型
#[derive(Debug, Error)]
pub enum WebSocketPortError {
    #[error("connection error: {0}")]
    Connection(String),
    #[error("message serialization error: {0}")]
    Serialization(String),
    #[error("user not connected: {user_id}")]
    UserNotConnected { user_id: i64 },
    #[error("room not found: {room_id}")]
    RoomNotFound { room_id: String },
    #[error("permission denied: {0}")]
    PermissionDenied(String),
    #[error("other error: {0}")]
    Other(String),
}

pub type WebSocketPortResult<T> = Result<T, WebSocketPortError>;

/// WebSocket 通信端口 - 定义实时通信的抽象接口
pub trait WebSocketPort: Send + Sync {
    /// 获取端口名称
    fn name(&self) -> &str;

    /// 向特定用户发送消息
    fn send_to_user<'a>(
        &'a self,
        user_id: i64,
        message: ServerMessage,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 向房间内所有用户发送消息
    fn send_to_room<'a>(
        &'a self,
        room_id: &'a str,
        message: ServerMessage,
        exclude_user_id: Option<i64>,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 向多个用户发送消息
    fn send_to_users<'a>(
        &'a self,
        user_ids: &'a [i64],
        message: ServerMessage,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 检查用户是否在线
    fn is_user_online<'a>(
        &'a self,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<bool>> + Send + 'a>>;

    /// 获取在线用户数量
    fn get_online_user_count<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<usize>> + Send + 'a>>;

    /// 获取房间内用户数量
    fn get_room_user_count<'a>(
        &'a self,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<usize>> + Send + 'a>>;

    /// 用户加入房间
    fn join_room<'a>(
        &'a self,
        user_id: i64,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 用户离开房间
    fn leave_room<'a>(
        &'a self,
        user_id: i64,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 发送系统通知
    fn send_system_notification<'a>(
        &'a self,
        user_id: i64,
        notification: SystemNotificationData,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;
}

/// WebSocket 连接管理端口 - 定义连接生命周期管理
pub trait WebSocketConnectionPort: Send + Sync {
    /// 获取端口名称
    fn name(&self) -> &str;

    /// 注册新连接
    fn register_connection<'a>(
        &'a self,
        connection_id: Uuid,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 注销连接
    fn unregister_connection<'a>(
        &'a self,
        connection_id: Uuid,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>>;

    /// 获取用户的所有连接
    fn get_user_connections<'a>(
        &'a self,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<Vec<Uuid>>> + Send + 'a>>;

    /// 检查连接是否存在
    fn connection_exists<'a>(
        &'a self,
        connection_id: Uuid,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<bool>> + Send + 'a>>;
}

impl WebSocketPortError {
    /// 从连接错误创建
    pub fn from_connection_error(err: impl std::fmt::Display) -> Self {
        Self::Connection(err.to_string())
    }

    /// 从序列化错误创建
    pub fn from_serialization_error(err: impl std::fmt::Display) -> Self {
        Self::Serialization(err.to_string())
    }

    /// 创建用户未连接错误
    pub fn user_not_connected(user_id: i64) -> Self {
        Self::UserNotConnected { user_id }
    }

    /// 创建房间未找到错误
    pub fn room_not_found(room_id: impl Into<String>) -> Self {
        Self::RoomNotFound { room_id: room_id.into() }
    }

    /// 创建权限拒绝错误
    pub fn permission_denied(msg: impl Into<String>) -> Self {
        Self::PermissionDenied(msg.into())
    }

    /// 从 WebSocket 错误转换
    pub fn from_ws_error(err: crate::ws::errors::WsError) -> Self {
        // WsError 是结构体，根据错误代码进行分类
        if err.code.contains("CONNECTION") {
            Self::Connection(err.message)
        } else if err.code.contains("SERIALIZATION") || err.code.contains("JSON") {
            Self::Serialization(err.message)
        } else if err.code.contains("ROOM") {
            Self::RoomNotFound { room_id: err.details.unwrap_or_default() }
        } else if err.code.contains("FORBIDDEN") || err.code.contains("UNAUTHORIZED") {
            Self::PermissionDenied(err.message)
        } else {
            Self::Other(err.message)
        }
    }
}
