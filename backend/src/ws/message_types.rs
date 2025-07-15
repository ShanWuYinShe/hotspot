//! WebSocket消息类型定义
//! 
//! 这个模块定义了所有WebSocket消息的类型，包括：
//! - 客户端发送到服务器的消息（ClientMessage）
//! - 服务器发送到客户端的消息（ServerMessage）

use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};

// ==================== 客户端到服务器的消息 ====================

/// 客户端发送到服务器的消息
#[derive(Debug, Clone, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum ClientMessage {
    /// 连接认证
    Connect { token: String },
    /// 心跳检测
    Ping,
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
    pub recipient_id: Option<i64>,
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
#[derive(Debug, Clone, Serialize, Deserialize)]
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

// ==================== 服务器到客户端的消息 ====================

/// 服务器发送到客户端的消息
#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", content = "data")]
pub enum ServerMessage {
    /// 心跳响应
    Pong,
    /// 连接成功
    Connected { user_id: i64, session_id: String },
    /// 错误消息
    Error { code: String, message: String },
    /// 成功消息
    Success { message: String },
    /// 新消息通知
    NewMessage(MessageNotification),
    /// 用户开始输入
    UserStartedTyping { room_id: Uuid, user_id: i64, nickname: String },
    /// 用户停止输入
    UserStoppedTyping { room_id: Uuid, user_id: i64 },
    /// 消息已读通知
    MessageRead { room_id: Uuid, user_id: i64, message_id: Option<Uuid> },
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
    pub sender_id: i64,
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
    pub actor_id: i64,
    pub actor_nickname: String,
    pub target_id: Option<i64>,
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
    NewCheckIn,
    GroupDigest,
}

// ==================== 系统内部事件 ====================

/// 系统内部事件（用于服务间通信）
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub enum InternalEvent {
    /// 用户连接
    UserConnected { user_id: i64, session_id: String },
    /// 用户断开连接
    UserDisconnected { user_id: i64, session_id: String },
    /// 消息发送事件
    MessageSent {
        message_id: Uuid,
        room_id: Uuid,
        room_type: RoomType,
        sender_id: i64,
        content: String,
        message_type: MessageType,
    },
    /// 群组事件
    GroupEvent {
        group_id: Uuid,
        event_type: GroupEventType,
        actor_id: i64,
        target_id: Option<i64>,
    },
    /// 用户位置更新
    UserLocationUpdate {
        user_id: i64,
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
    #[allow(dead_code)]
    pub fn to_string(&self) -> String {
        match self {
            RoomId::Group(id) => format!("group:{}", id),
            RoomId::Conversation(id) => format!("conversation:{}", id),
            RoomId::User(id) => format!("user:{}", id),
        }
    }
    
    pub fn from_group(group_id: Uuid) -> Self {
        RoomId::Group(group_id)
    }
    
    pub fn from_conversation(conversation_id: Uuid) -> Self {
        RoomId::Conversation(conversation_id)
    }
    
    pub fn from_user(user_id: i64) -> Self {
        RoomId::User(user_id)
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

/// WebSocket错误类型
#[derive(Debug, Clone, Serialize)]
pub struct WsError {
    pub code: String,
    pub message: String,
}

impl WsError {
    #[allow(dead_code)]
    pub fn invalid_message(message: &str) -> Self {
        Self {
            code: "INVALID_MESSAGE".to_string(),
            message: message.to_string(),
        }
    }
    
    #[allow(dead_code)]
    pub fn unauthorized() -> Self {
        Self {
            code: "UNAUTHORIZED".to_string(),
            message: "未授权访问".to_string(),
        }
    }
    
    #[allow(dead_code)]
    pub fn room_not_found() -> Self {
        Self {
            code: "ROOM_NOT_FOUND".to_string(),
            message: "房间不存在".to_string(),
        }
    }
    
    #[allow(dead_code)]
    pub fn internal_error() -> Self {
        Self {
            code: "INTERNAL_ERROR".to_string(),
            message: "服务器内部错误".to_string(),
        }
    }
}
