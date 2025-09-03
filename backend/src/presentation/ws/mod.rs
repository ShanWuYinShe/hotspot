/// WebSocket表现层模块
/// 
/// 处理WebSocket协议相关的实时通信，包括连接管理、消息路由等
/// 遵循"策略选人→定向推送"原则，禁止全局广播

pub mod connection_handler;
pub mod message_handler;
pub mod event_handler;
pub mod websocket_manager;

// 重新导出WebSocket表现层类型
pub use connection_handler::*;
pub use message_handler::*;
pub use event_handler::*;
pub use websocket_manager::*;

use serde::{Deserialize, Serialize};

/// WebSocket消息类型枚举
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "type")]
pub enum WSMessageType {
    /// 认证消息
    Auth { token: String },
    /// 心跳消息
    Ping,
    /// 心跳响应
    Pong,
    /// 订阅房间
    Subscribe { room_id: String },
    /// 取消订阅
    Unsubscribe { room_id: String },
    /// 发送消息
    SendMessage { room_id: String, content: String },
    /// 接收消息
    ReceiveMessage { room_id: String, message: MessageData },
    /// 消息已接收（广播用）
    MessageReceived { room_id: String, sender_id: i64, content: String, timestamp: i64 },
    /// 系统通知
    SystemNotification { message: String },
    /// 错误消息
    Error { code: u32, message: String },
}

/// 消息数据结构
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MessageData {
    pub id: i64,
    pub sender_id: i64,
    pub sender_nickname: String,
    pub content: String,
    pub message_type: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// WebSocket事件类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "event")]
pub enum WSEventType {
    /// 用户上线
    UserOnline { user_id: i64, nickname: String },
    /// 用户下线
    UserOffline { user_id: i64 },
    /// 用户加入群组
    UserJoinedGroup { user_id: i64, group_id: i64, nickname: String },
    /// 用户离开群组
    UserLeftGroup { user_id: i64, group_id: i64 },
    /// 新消息通知
    NewMessage { group_id: i64, message: MessageData },
    /// 群组信息更新
    GroupInfoUpdated { group_id: i64, name: String },
}

/// WebSocket连接状态
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ConnectionState {
    /// 连接中
    Connecting,
    /// 已连接但未认证
    Connected,
    /// 已认证
    Authenticated { user_id: i64 },
    /// 连接关闭中
    Closing,
    /// 已关闭
    Closed,
}

/// WebSocket错误类型
#[derive(Debug, Clone, thiserror::Error)]
pub enum WSError {
    #[error("认证失败: {reason}")]
    AuthenticationFailed { reason: String },
    
    #[error("需要认证: {operation}")]
    AuthenticationRequired { operation: String },
    
    #[error("认证超时")]
    AuthenticationTimeout,
    
    #[error("连接已关闭")]
    ConnectionClosed,
    
    #[error("无效的消息格式: {message}")]
    InvalidMessageFormat { message: String },
    
    #[error("房间不存在: {room_id}")]
    RoomNotFound { room_id: String },
    
    #[error("无效的房间ID: {room_id}")]
    InvalidRoomId { room_id: String },
    
    #[error("权限不足")]
    InsufficientPermission,
    
    #[error("消息发送失败: {reason}")]
    MessageSendFailed { reason: String },
    
    #[error("内部错误: {error}")]
    InternalError { error: String },
}

impl WSError {
    /// 转换为WebSocket消息
    pub fn to_ws_message(&self) -> WSMessageType {
        let (code, message) = match self {
            WSError::AuthenticationFailed { reason } => (4001, format!("认证失败: {}", reason)),
            WSError::AuthenticationRequired { operation } => (4002, format!("需要认证: {}", operation)),
            WSError::AuthenticationTimeout => (4003, "认证超时".to_string()),
            WSError::ConnectionClosed => (4004, "连接已关闭".to_string()),
            WSError::InvalidMessageFormat { message } => (4005, format!("无效消息格式: {}", message)),
            WSError::RoomNotFound { room_id } => (4006, format!("房间不存在: {}", room_id)),
            WSError::InvalidRoomId { room_id } => (4007, format!("无效的房间ID: {}", room_id)),
            WSError::InsufficientPermission => (4008, "权限不足".to_string()),
            WSError::MessageSendFailed { reason } => (4009, format!("消息发送失败: {}", reason)),
            WSError::InternalError { error } => (5000, format!("内部错误: {}", error)),
        };
        
        WSMessageType::Error { code, message }
    }
}
