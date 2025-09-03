//! WebSocket消息类型定义
//! 
//! 这个模块定义了所有WebSocket消息的类型，包括：
//! - 客户端发送到服务器的消息（ClientMessage）
//! - 服务器发送到客户端的消息（ServerMessage）

use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};
use crate::utils::JsSafeI64;

// ==================== 客户端到服务器的消息 ====================

/// 客户端发送到服务器的消息
#[derive(Debug, Clone, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum ClientMessage {
    /// 连接认证
    Connect { token: String },
    /// 心跳检测
    Ping,
    /// 加入当前用户的私有房间（用于接收系统通知等）
    JoinUserRoom,
    /// 离开当前用户的私有房间
    LeaveUserRoom,
    /// 加入群组房间
    JoinGroup { group_id: Uuid },
    /// 离开群组房间
    LeaveGroup { group_id: Uuid },
    /// 加入私聊房间
    JoinConversation { conversation_id: Uuid },
    /// 离开私聊房间
    LeaveConversation { conversation_id: Uuid },
}

/// 消息数据
#[derive(Debug, Clone, Deserialize)]
pub struct MessageData {
    #[allow(dead_code)]
    pub room_id: Uuid,
    #[allow(dead_code)]
    pub room_type: RoomType,
    #[allow(dead_code)]
    pub recipient_id: Option<JsSafeI64>,
    #[allow(dead_code)]
    pub content: String,
    #[allow(dead_code)]
    pub message_type: MessageType,
    #[allow(dead_code)]
    pub latitude: Option<f64>,
    #[allow(dead_code)]
    pub longitude: Option<f64>,
}

/// 房间类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
#[serde(rename_all = "snake_case")]
pub enum RoomType {
    Group,
    Conversation,
}

/// 消息类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MessageType {
    Text,
    Image,
    Voice,
    Video,
    Location,
    File,
}

impl MessageType {
    pub fn from_string_or_default(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "text" => MessageType::Text,
            "image" => MessageType::Image,
            "voice" => MessageType::Voice,
            "video" => MessageType::Video,
            "file" => MessageType::File,
            "location" => MessageType::Location,
            _ => MessageType::Text,
        }
    }
}

// ==================== 服务器到客户端的消息 ====================

/// 服务器发送到客户端的消息
#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", content = "data")]
pub enum ServerMessage {
    /// 心跳响应
    Pong,
    /// 连接成功
    Connected { user_id: JsSafeI64, session_id: String },
    /// 错误消息
    Error { code: String, message: String },
    /// 成功消息
    Success { message: String },
    /// 新消息通知
    NewMessage(MessageNotification),
    /// 用户开始输入
    UserStartedTyping { room_id: Uuid, user_id: JsSafeI64, nickname: String },
    /// 用户停止输入
    UserStoppedTyping { room_id: Uuid, user_id: JsSafeI64 },
    /// 消息已读通知
    MessageRead { room_id: Uuid, user_id: JsSafeI64, message_id: Option<Uuid> },
    /// 群组事件通知
    GroupEvent(GroupEventData),
    /// 系统通知
    SystemNotification(SystemNotificationData),
    /// 消息删除通知
    MessageDeleted { message_id: Uuid, room_id: Uuid, room_type: RoomType },
}

/// 消息通知
#[derive(Debug, Clone, Serialize)]
pub struct MessageNotification {
    pub message_id: Uuid,
    pub room_id: Uuid,
    pub room_type: RoomType,
    pub sender_id: JsSafeI64,
    pub sender_nickname: String,
    pub content: String,
    pub message_type: MessageType,
    pub created_at: DateTime<Utc>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 群组事件数据
#[derive(Debug, Clone, Serialize)]
pub struct GroupEventData {
    pub group_id: Uuid,
    pub event_type: GroupEventType,
    pub actor_id: JsSafeI64,
    pub actor_nickname: String,
    pub target_id: Option<JsSafeI64>,
    pub target_nickname: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// 群组事件类型
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum GroupEventType {
    MemberJoined,
    MemberLeft,
    MemberKicked,
    OwnerTransferred,
    GroupUpdated,
    #[allow(dead_code)]
    GroupDeleted,
}

/// 系统通知数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemNotificationData {
    pub notification_id: Uuid,
    pub notification_type: SystemNotificationType,
    pub title: String,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub created_at: DateTime<Utc>,
}

/// 系统通知类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SystemNotificationType {
    FriendRequest,
    GroupInvitation,
    System,
    Announcement,
    NewMoment,
    GroupDigest,
    UserDigest,
    Welcome,
    NewMessage,
}

// ==================== 系统内部事件 ====================

/// 系统内部事件（用于服务间通信）
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub enum InternalEvent {
    /// 用户连接
    UserConnected { user_id: JsSafeI64, session_id: String },
    /// 用户断开连接
    UserDisconnected { user_id: JsSafeI64, session_id: String },
    /// 消息发送事件
    MessageSent {
        message_id: Uuid,
        room_id: Uuid,
        room_type: RoomType,
        sender_id: JsSafeI64,
        content: String,
        message_type: MessageType,
    },
    /// 群组事件
    GroupEvent {
        group_id: Uuid,
        event_type: GroupEventType,
        actor_id: JsSafeI64,
        target_id: Option<JsSafeI64>,
    },
    /// 用户位置更新
    UserLocationUpdate {
        user_id: JsSafeI64,
        latitude: f64,
        longitude: f64,
        location_name: String,
    },
}

// ==================== 辅助类型 ====================

/// WebSocket房间ID
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum RoomId {
    Group(Uuid),
    Conversation(Uuid),
    User(i64),
}

impl RoomId {
    
    pub fn from_group(group_id: Uuid) -> Self {
        RoomId::Group(group_id)
    }
    
    pub fn from_conversation(conversation_id: Uuid) -> Self {
        RoomId::Conversation(conversation_id)
    }
    
    pub fn from_user(user_id: i64) -> Self {
        RoomId::User(user_id)
    }
    
    pub fn from_room_type(id: Uuid, room_type: super::message_types::RoomType) -> Self {
        match room_type {
            RoomType::Group => RoomId::Group(id),
            RoomType::Conversation => RoomId::Conversation(id),
        }
    }
}

impl std::fmt::Display for RoomId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            RoomId::Group(id) => write!(f, "group:{}", id),
            RoomId::Conversation(id) => write!(f, "conversation:{}", id),
            RoomId::User(id) => write!(f, "user:{}", id),
        }
    }
}

// 移除重复的 WsError 定义，请使用 crate::ws::errors::WsError
