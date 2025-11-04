use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// WebSocket 消息类型定义
/// 参考 api_spec.mdc 第 4 章 WebSocket 实时事件定义
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(tag = "type", content = "payload")]
pub enum WsMessage {
    // ========== 服务端推送事件 (Server → Client) ==========
    
    // ===== 连接管理事件 =====
    ConnectionEstablished {
        connection_id: String,
        user_id: String,
        server_time: String,  // ISO 8601 UTC
    },
    
    ConnectionClosed {
        reason: String,
        code: u16,
    },
    
    Heartbeat {
        timestamp: String,  // ISO 8601 UTC
    },
    
    // ===== 聊天消息事件 =====
    
    /// 新消息
    NewMessage {
        id: String,
        target_id: String,
        target_type: String,  // "chat" | "group"
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: String,  // "text" | "image" | "video" | "audio" | "file" | "location" | "system"
        content: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        mentions: Option<Vec<String>>,
        sent_at: String,  // ISO 8601 UTC
        
        // 位置消息字段
        #[serde(skip_serializing_if = "Option::is_none")]
        latitude: Option<f64>,
        #[serde(skip_serializing_if = "Option::is_none")]
        longitude: Option<f64>,
        #[serde(skip_serializing_if = "Option::is_none")]
        location_name: Option<String>,
        
        // 文件/媒体字段
        #[serde(skip_serializing_if = "Option::is_none")]
        file_name: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        file_size: Option<i64>,
        #[serde(skip_serializing_if = "Option::is_none")]
        thumbnail_url: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        duration: Option<i32>,
        
        // 系统消息字段
        #[serde(skip_serializing_if = "Option::is_none")]
        system_event: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        affected_users: Option<Vec<String>>,
    },
    
    /// 消息被编辑
    MessageEdited {
        id: String,
        target_id: String,
        target_type: String,
        content: String,
        edited_at: String,  // ISO 8601 UTC
    },
    
    /// 消息被撤回
    MessageRecalled {
        id: String,
        target_id: String,
        target_type: String,
        recalled_by: String,
        recalled_at: String,  // ISO 8601 UTC
    },
    
    /// 消息被删除
    MessageDeleted {
        id: String,
        target_id: String,
        target_type: String,
        deleted_by: String,
        for_everyone: bool,
        deleted_at: String,  // ISO 8601 UTC
    },
    
    /// 消息反应
    MessageReaction {
        id: String,  // 消息ID
        target_id: String,
        target_type: String,
        user: UserInfo,
        reaction_type: String,  // "like" | "love" | "laugh" | "angry" | "sad" | "wow"
        action: String,  // "add" | "remove"
        created_at: String,  // ISO 8601 UTC
    },
    
    /// 消息已读回执
    MessageRead {
        target_id: String,
        target_type: String,
        up_to: String,  // 读到哪条消息ID
        reader_id: String,
        read_at: String,  // ISO 8601 UTC
    },
    
    /// 用户正在输入
    UserTyping {
        target_id: String,
        target_type: String,
        user_id: String,
        nickname: String,
        is_typing: bool,
    },
    
    // ===== 在线状态事件 =====
    
    /// 用户上线
    UserOnline {
        user_id: String,
        nickname: String,
        avatar_url: String,
        online_at: String,  // ISO 8601 UTC
    },
    
    /// 用户离线
    UserOffline {
        user_id: String,
        offline_at: String,  // ISO 8601 UTC
    },
    
    /// 用户位置更新
    UserLocationUpdate {
        user_id: String,
        latitude: f64,
        longitude: f64,
        location_name: String,
        updated_at: String,  // ISO 8601 UTC
    },
    
    // ===== 群组事件 =====
    
    /// 成员加入群组
    GroupMemberJoined {
        group_id: String,
        user: UserInfo,
        joined_at: String,  // ISO 8601 UTC
    },
    
    /// 成员离开群组
    GroupMemberLeft {
        group_id: String,
        user_id: String,
        left_at: String,  // ISO 8601 UTC
    },
    
    /// 成员被移除
    GroupMemberRemoved {
        group_id: String,
        user_id: String,
        removed_by: String,
        removed_at: String,  // ISO 8601 UTC
    },
    
    /// 成员角色变更
    GroupRoleChanged {
        group_id: String,
        user_id: String,
        old_role: String,
        new_role: String,
        changed_by: String,
        changed_at: String,  // ISO 8601 UTC
    },
    
    /// 群组资料更新
    GroupProfileUpdated {
        group_id: String,
        updated_fields: HashMap<String, String>,
        updated_at: String,  // ISO 8601 UTC
    },
    
    /// 私聊升级为群组
    ChatUpgradedToGroup {
        old_chat_id: String,
        new_group_id: String,
        group_type: String,  // "public_group" | "private_group"
        upgraded_at: String,  // ISO 8601 UTC
    },
    
    // ===== 收件箱事件 =====
    
    /// 收件箱提醒推送
    InboxItem {
        id: String,
        category: String,  // "social" | "system" | "content"
        #[serde(rename = "type")]
        item_type: String,  // 具体类型（如 friend_request, moment_like 等）
        #[serde(skip_serializing_if = "Option::is_none")]
        title: Option<String>,
        content: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        data: Option<HashMap<String, serde_json::Value>>,
        #[serde(skip_serializing_if = "Option::is_none")]
        action_url: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        severity: Option<String>,  // "info" | "warning" | "error"
        created_at: String,  // ISO 8601 UTC
    },
    
    // ===== 系统事件 =====
    
    /// 系统广播消息
    SystemBroadcast {
        message: String,
        severity: String,  // "info" | "warning" | "error"
        #[serde(skip_serializing_if = "Option::is_none")]
        action_required: Option<bool>,
        timestamp: String,  // ISO 8601 UTC
    },
    
    /// 错误消息
    Error {
        error_code: String,
        error_message: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        details: Option<serde_json::Value>,
    },
    
    // ========== 客户端发送事件 (Client → Server) ==========
    
    /// 心跳保活
    Ping {
        timestamp: String,  // ISO 8601 UTC
    },
    
    /// 心跳响应
    Pong {
        timestamp: String,  // ISO 8601 UTC
    },
    
    /// 订阅频道
    Subscribe {
        channels: Vec<String>,
    },
    
    /// 取消订阅
    Unsubscribe {
        channels: Vec<String>,
    },
    
    /// 订阅确认（服务端响应）
    SubscribeAck {
        subscribed: Vec<String>,
        failed: Vec<SubscribeError>,
    },
    
    /// 标记已读（快速通道）
    MarkAsRead {
        target_id: String,
        target_type: String,  // "chat" | "group"
        up_to: String,  // 读到哪条消息ID
    },
    
    /// 正在输入通知
    Typing {
        target_id: String,
        target_type: String,  // "chat" | "group"
        is_typing: bool,
    },
}

/// 用户基本信息
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserInfo {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

/// 订阅错误信息
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SubscribeError {
    pub channel: String,
    pub reason: String,
}

