//! 统一WebSocket消息类型定义
//! 
//! 根据需求文档实现完整的WebSocket消息类型系统
//! 支持实时通信、用户状态、群组事件、生活瞬间、系统通知等

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use std::collections::HashMap;

// ==================== 基础消息格式 ====================

/// WebSocket消息的基础格式
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketMessage {
    /// 消息类型
    pub message_type: String,
    /// 消息数据
    pub data: serde_json::Value,
    /// 消息ID（用于确认和去重）
    pub message_id: String,
    /// 时间戳
    pub timestamp: DateTime<Utc>,
    /// 发送者ID（服务器消息可为空）
    pub sender_id: Option<String>,
}

// ==================== 客户端到服务器的消息 ====================

/// 客户端发送到服务器的消息
#[derive(Debug, Clone, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum ClientMessage {
    /// 连接认证
    Auth { 
        token: String 
    },
    
    /// 心跳检测
    Ping,
    
    /// 加入群组房间
    JoinGroup { 
        group_id: String 
    },
    
    /// 离开群组房间
    LeaveGroup { 
        group_id: String 
    },
    
    /// 发送消息
    SendMessage {
        group_id: String,
        content: String,
        message_type: String,
        reply_to_message_id: Option<String>,
        mentioned_users: Option<Vec<String>>,
        latitude: Option<f64>,
        longitude: Option<f64>,
    },
    
    /// 标记消息已读
    MarkRead {
        group_id: String,
        message_id: Option<String>,
    },
    
    /// 正在输入状态
    Typing {
        group_id: String,
        is_typing: bool,
    },
    
    /// 更新用户状态
    UpdateStatus {
        status: UserStatus,
        location: Option<LocationUpdate>,
    },
}

/// 用户状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum UserStatus {
    Online,
    Away,
    Busy,
    Offline,
}

/// 位置更新
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationUpdate {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
    pub accuracy: Option<f64>,
}

// ==================== 服务器到客户端的消息 ====================

/// 服务器发送到客户端的消息
#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", content = "data")]
pub enum ServerMessage {
    /// 连接确认
    Connected {
        user_id: String,
        session_id: String,
    },
    
    /// 心跳响应
    Pong,
    
    /// 认证成功
    AuthSuccess {
        user_id: String,
        nickname: String,
    },
    
    /// 认证失败
    AuthError {
        error: String,
    },
    
    /// 新消息
    NewMessage {
        message: MessagePayload,
    },
    
    /// 消息已读状态更新
    MessageRead {
        group_id: String,
        user_id: String,
        message_id: String,
        read_at: DateTime<Utc>,
    },
    
    /// 正在输入指示器
    TypingIndicator {
        group_id: String,
        user_id: String,
        nickname: String,
        is_typing: bool,
    },
    
    /// 用户在线状态更新
    UserOnline {
        user_id: String,
        nickname: String,
        status: UserStatus,
    },
    
    /// 用户离线状态更新
    UserOffline {
        user_id: String,
        nickname: String,
    },
    
    /// 用户位置更新
    UserLocationUpdate {
        user_id: String,
        nickname: String,
        location: LocationUpdate,
    },
    
    /// 群组成员加入
    GroupMemberJoined {
        group_id: String,
        user_id: String,
        nickname: String,
        role: String,
        joined_at: DateTime<Utc>,
    },
    
    /// 群组成员离开
    GroupMemberLeft {
        group_id: String,
        user_id: String,
        nickname: String,
        left_at: DateTime<Utc>,
    },
    
    /// 群组角色变更
    GroupRoleChanged {
        group_id: String,
        user_id: String,
        nickname: String,
        old_role: String,
        new_role: String,
        changed_by: String,
    },
    
    /// 群组信息更新
    GroupInfoUpdated {
        group_id: String,
        updated_fields: HashMap<String, serde_json::Value>,
        updated_by: String,
    },
    
    /// 附近新的生活瞬间
    NewMomentNearby {
        moment: MomentPayload,
        distance: f64,
    },
    
    /// 生活瞬间被点赞
    MomentLiked {
        moment_id: String,
        user_id: String,
        nickname: String,
        likes_count: i32,
    },
    
    /// 生活瞬间被评论
    MomentCommented {
        moment_id: String,
        comment: CommentPayload,
    },
    
    /// 系统通知
    SystemNotification {
        notification: NotificationPayload,
    },
    
    /// 好友请求接收
    FriendRequestReceived {
        from_user_id: String,
        from_nickname: String,
        message: Option<String>,
        request_id: String,
    },
    
    /// 好友请求被接受
    FriendRequestAccepted {
        user_id: String,
        nickname: String,
        request_id: String,
    },
    
    /// 内容被举报
    ContentReported {
        target_type: String,
        target_id: String,
        report_id: String,
        reason: String,
    },
    
    /// 错误消息
    Error {
        error_code: String,
        error_message: String,
        context: Option<HashMap<String, serde_json::Value>>,
    },
}

// ==================== 消息载荷数据结构 ====================

/// 消息载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessagePayload {
    pub message_id: String,
    pub group_id: String,
    pub sender_id: String,
    pub sender_nickname: String,
    pub sender_avatar: Option<String>,
    pub content: String,
    pub message_type: String,
    pub reply_to_message_id: Option<String>,
    pub mentioned_users: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub media_attachments: Vec<MediaAttachment>,
}

/// 媒体附件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MediaAttachment {
    pub media_id: String,
    pub media_type: String,
    pub file_url: String,
    pub thumbnail_url: Option<String>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub duration: Option<i32>,
}

/// 生活瞬间载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentPayload {
    pub moment_id: String,
    pub user_id: String,
    pub nickname: String,
    pub user_avatar: Option<String>,
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub images: Vec<String>,
    pub tags: Vec<String>,
    pub likes_count: i32,
    pub comments_count: i32,
    pub created_at: DateTime<Utc>,
}

/// 评论载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommentPayload {
    pub comment_id: String,
    pub moment_id: String,
    pub user_id: String,
    pub nickname: String,
    pub user_avatar: Option<String>,
    pub content: String,
    pub parent_comment_id: Option<String>,
    pub replies_count: i32,
    pub created_at: DateTime<Utc>,
}

/// 通知载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationPayload {
    pub notification_id: String,
    pub notification_type: String,
    pub title: String,
    pub content: String,
    pub data: HashMap<String, serde_json::Value>,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
}

// ==================== 连接管理相关 ====================

/// WebSocket连接状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConnectionState {
    Connecting,
    Connected,
    Authenticated,
    Disconnecting,
    Disconnected,
    Error,
}

/// 连接信息
#[derive(Debug, Clone)]
pub struct ConnectionInfo {
    pub connection_id: String,
    pub user_id: Option<String>,
    pub session_id: String,
    pub state: ConnectionState,
    pub connected_at: DateTime<Utc>,
    pub last_ping_at: Option<DateTime<Utc>>,
    pub subscribed_groups: Vec<String>,
    pub user_agent: Option<String>,
    pub ip_address: Option<String>,
}

/// 房间信息
#[derive(Debug, Clone)]
pub struct RoomInfo {
    pub room_id: String,
    pub room_type: RoomType,
    pub connected_users: Vec<String>,
    pub created_at: DateTime<Utc>,
    pub last_activity_at: DateTime<Utc>,
}

/// 房间类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum RoomType {
    /// 用户私有房间（接收系统通知）
    User,
    /// 群组房间（群聊）
    Group,
    /// 私聊房间
    PrivateChat,
    /// 全局广播房间
    Global,
}

// ==================== 消息路由和分发 ====================

/// 消息路由信息
#[derive(Debug, Clone)]
pub struct MessageRoute {
    /// 目标房间ID
    pub target_room: String,
    /// 房间类型
    pub room_type: RoomType,
    /// 排除的用户ID列表（不发送给这些用户）
    pub exclude_users: Vec<String>,
    /// 只发送给指定用户ID列表（如果为空则发送给房间所有用户）
    pub include_users: Vec<String>,
    /// 消息优先级
    pub priority: MessagePriority,
}

/// 消息优先级
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub enum MessagePriority {
    Low,
    Normal,
    High,
    Critical,
}

// ==================== 错误类型 ====================

/// WebSocket错误类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum WebSocketError {
    /// 认证错误
    AuthenticationError { message: String },
    /// 权限错误
    PermissionError { message: String },
    /// 消息格式错误
    MessageFormatError { message: String },
    /// 房间不存在
    RoomNotFound { room_id: String },
    /// 用户不在房间中
    UserNotInRoom { user_id: String, room_id: String },
    /// 连接错误
    ConnectionError { message: String },
    /// 内部服务器错误
    InternalError { message: String },
}

// ==================== 统计和监控 ====================

/// WebSocket连接统计
#[derive(Debug, Clone, Serialize)]
pub struct ConnectionStats {
    pub total_connections: usize,
    pub authenticated_connections: usize,
    pub active_rooms: usize,
    pub messages_sent_per_minute: f64,
    pub messages_received_per_minute: f64,
    pub average_latency_ms: f64,
    pub error_rate: f64,
}

/// 房间统计
#[derive(Debug, Clone, Serialize)]
pub struct RoomStats {
    pub room_id: String,
    pub room_type: RoomType,
    pub connected_users: usize,
    pub messages_per_minute: f64,
    pub created_at: DateTime<Utc>,
    pub last_activity_at: DateTime<Utc>,
}

// ==================== 辅助函数 ====================

impl WebSocketMessage {
    /// 创建新的WebSocket消息
    pub fn new(message_type: String, data: serde_json::Value, sender_id: Option<String>) -> Self {
        Self {
            message_type,
            data,
            message_id: uuid::Uuid::new_v4().to_string(),
            timestamp: Utc::now(),
            sender_id,
        }
    }
    
    /// 创建错误消息
    pub fn error(error_code: String, error_message: String) -> Self {
        let error_data = serde_json::json!({
            "error_code": error_code,
            "error_message": error_message
        });
        
        Self::new("error".to_string(), error_data, None)
    }
}

impl ConnectionInfo {
    /// 创建新的连接信息
    pub fn new(connection_id: String, session_id: String) -> Self {
        Self {
            connection_id,
            user_id: None,
            session_id,
            state: ConnectionState::Connecting,
            connected_at: Utc::now(),
            last_ping_at: None,
            subscribed_groups: Vec::new(),
            user_agent: None,
            ip_address: None,
        }
    }
    
    /// 设置用户认证信息
    pub fn authenticate(&mut self, user_id: String) {
        self.user_id = Some(user_id);
        self.state = ConnectionState::Authenticated;
    }
    
    /// 更新心跳时间
    pub fn update_ping(&mut self) {
        self.last_ping_at = Some(Utc::now());
    }
    
    /// 加入群组
    pub fn join_group(&mut self, group_id: String) {
        if !self.subscribed_groups.contains(&group_id) {
            self.subscribed_groups.push(group_id);
        }
    }
    
    /// 离开群组
    pub fn leave_group(&mut self, group_id: &str) {
        self.subscribed_groups.retain(|id| id != group_id);
    }
}

impl RoomInfo {
    /// 创建新的房间信息
    pub fn new(room_id: String, room_type: RoomType) -> Self {
        Self {
            room_id,
            room_type,
            connected_users: Vec::new(),
            created_at: Utc::now(),
            last_activity_at: Utc::now(),
        }
    }
    
    /// 添加用户到房间
    pub fn add_user(&mut self, user_id: String) {
        if !self.connected_users.contains(&user_id) {
            self.connected_users.push(user_id);
            self.last_activity_at = Utc::now();
        }
    }
    
    /// 从房间移除用户
    pub fn remove_user(&mut self, user_id: &str) {
        self.connected_users.retain(|id| id != user_id);
        self.last_activity_at = Utc::now();
    }
    
    /// 更新活动时间
    pub fn update_activity(&mut self) {
        self.last_activity_at = Utc::now();
    }
}
