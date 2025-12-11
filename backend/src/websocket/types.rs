use serde::{Deserialize, Serialize};

/// WebSocket 消息类型定义
/// 严格遵循 api_spec.mdc WebSocket 实时事件定义
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(tag = "type", content = "payload")]
pub enum WsMessage {
    // ========== 服务端推送事件 (Server → Client) ==========
    
    // ===== 连接管理事件 =====
    
    /// 连接建立确认
    ConnectionEstablished {
        connection_id: String,
        user_id: i64,
        server_time: String,  // ISO 8601 UTC
    },
    
    /// 心跳
    Heartbeat {
        timestamp: String,  // ISO 8601 UTC
    },
    
    // ===== 在线状态事件 =====
    
    /// 用户上线
    UserOnline {
        user_id: i64,
        nickname: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        avatar_url: Option<String>,
        online_at: String,  // ISO 8601 UTC
    },
    
    /// 用户离线
    UserOffline {
        user_id: i64,
        offline_at: String,  // ISO 8601 UTC
    },
    
    /// 用户位置更新
    UserLocationUpdate {
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        updated_at: String,  // ISO 8601 UTC
    },
    
    // ===== 关注关系事件 =====
    
    /// 新增关注者（有人关注了我）
    NewFollower {
        user: UserInfo,
        followed_at: String,  // ISO 8601 UTC
        is_mutual: bool,      // 是否互相关注（成为好友）
    },
    
    /// 互相关注（成为好友）
    MutualFollow {
        user: UserInfo,
        mutual_at: String,  // ISO 8601 UTC
    },
    
    /// 取消关注（好友关系解除）
    Unfollowed {
        user_id: i64,
        unfollowed_at: String,  // ISO 8601 UTC
        was_mutual: bool,       // 之前是否互相关注
    },
    
    // ===== 生活瞬间互动事件 =====
    
    /// 生活瞬间被点赞
    MomentLiked {
        moment_id: String,
        user: UserInfo,
        liked_at: String,  // ISO 8601 UTC
    },
    
    /// 生活瞬间被评论
    MomentCommented {
        moment_id: String,
        comment_id: String,
        user: UserInfo,
        content: String,
        commented_at: String,  // ISO 8601 UTC
    },
    
    /// 评论被回复
    CommentReply {
        moment_id: String,
        comment_id: String,        // 原评论ID（被回复的评论）
        reply_id: String,          // 回复评论ID
        user: UserInfo,            // 回复者
        content: String,
        replied_at: String,  // ISO 8601 UTC
    },
    
    /// 系统公告
    SystemAnnouncement {
        id: String,
        title: String,
        content: String,
        severity: String,  // "info" | "warning" | "error"
        #[serde(skip_serializing_if = "Option::is_none")]
        action_url: Option<String>,
        created_at: String,  // ISO 8601 UTC
    },
    
    // ===== 聊天消息事件 =====
    
    /// 新消息
    ChatMessageNew {
        id: String,
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: String,  // "text" | "image" | "video" | "audio" | "file" | "system"
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        mentions: Option<Vec<i64>>,
        // 媒体消息字段（type != 'text' 时，由后端解析 file_id 后填充）
        #[serde(skip_serializing_if = "Option::is_none")]
        media_url: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        thumbnail_url: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        file_name: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        file_size: Option<i64>,
        #[serde(skip_serializing_if = "Option::is_none")]
        duration: Option<i32>,
        #[serde(skip_serializing_if = "Option::is_none")]
        width: Option<i32>,
        #[serde(skip_serializing_if = "Option::is_none")]
        height: Option<i32>,
        sent_at: String,  // ISO 8601 UTC
    },
    
    /// 消息被编辑
    ChatMessageEdited {
        id: String,
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        content: String,
        edited_at: String,  // ISO 8601 UTC
    },
    
    /// 消息被撤回
    ChatMessageRecalled {
        id: String,
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        recalled_by: i64,
        recalled_at: String,  // ISO 8601 UTC
    },
    
    /// 消息被删除
    ChatMessageDeleted {
        id: String,
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        deleted_by: i64,
        for_everyone: bool,
        deleted_at: String,  // ISO 8601 UTC
    },
    
    /// 消息反应
    ChatMessageReaction {
        id: String,  // 消息ID
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        user: UserInfo,
        reaction_type: String,  // "like" | "love" | "laugh" | "angry" | "sad" | "wow"
        has_reaction: bool,     // 操作后状态：true=添加了反应，false=移除了反应
        created_at: String,     // ISO 8601 UTC
    },
    
    /// 消息已读回执
    ChatMessageRead {
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        up_to: String,      // 读到哪条消息ID
        reader_id: i64,
        read_at: String,    // ISO 8601 UTC
    },
    
    /// 用户正在输入
    UserTyping {
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        user_id: i64,
        nickname: String,
        is_typing: bool,
    },
    
    // ===== 群组事件 =====
    
    /// 成员加入群组
    GroupMemberJoined {
        group_chat_id: String,
        user: UserInfo,
        joined_at: String,  // ISO 8601 UTC
    },
    
    /// 成员离开群组
    GroupMemberLeft {
        group_chat_id: String,
        user_id: i64,
        left_at: String,  // ISO 8601 UTC
    },
    
    /// 群组资料更新
    GroupProfileUpdated {
        group_chat_id: String,
        updated_fields: GroupUpdatedFields,
        updated_at: String,  // ISO 8601 UTC
    },
    
    /// 成员角色变更
    GroupMemberRoleChanged {
        group_chat_id: String,
        user_id: i64,
        old_role: String,  // "owner" | "admin" | "member"
        new_role: String,  // "owner" | "admin" | "member"
        changed_at: String,  // ISO 8601 UTC
    },
    
    /// 成员被禁言
    GroupMemberMuted {
        group_chat_id: String,
        user_id: i64,
        muted_until: String,  // ISO 8601 UTC
        muted_at: String,     // ISO 8601 UTC
    },
    
    /// 成员取消禁言
    GroupMemberUnmuted {
        group_chat_id: String,
        user_id: i64,
        unmuted_at: String,  // ISO 8601 UTC
    },
    
    /// 群组解散
    GroupDismissed {
        group_chat_id: String,
        dismissed_by: i64,
        dismissed_at: String,  // ISO 8601 UTC
    },
    
    /// 收到群组邀请
    GroupInviteReceived {
        id: String,
        group_chat_id: String,
        group_name: String,
        inviter: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        created_at: String,  // ISO 8601 UTC
    },
    
    /// 群组邀请响应
    GroupInviteResponse {
        id: String,
        group_chat_id: String,
        user: UserInfo,
        status: String,  // "accepted" | "rejected"
        responded_at: String,  // ISO 8601 UTC
    },
    
    /// 收到群组申请
    GroupApplicationReceived {
        id: String,
        group_chat_id: String,
        applicant: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        created_at: String,  // ISO 8601 UTC
    },
    
    /// 群组申请响应
    GroupApplicationResponse {
        id: String,
        group_chat_id: String,
        group_name: String,
        status: String,  // "accepted" | "rejected"
        responded_at: String,  // ISO 8601 UTC
    },
    
    // ===== 系统事件 =====
    
    /// 错误消息
    Error {
        code: i32,
        message: String,
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
    
    /// 订阅（增量订阅）
    Subscribe {
        items: Vec<SubscriptionItem>,
    },
    
    /// 取消订阅
    Unsubscribe {
        items: Vec<SubscriptionItem>,
    },
    
    /// 替换式订阅（覆盖现有订阅）
    SetSubscriptions {
        items: Vec<SubscriptionItem>,
    },
    
    /// 订阅确认（服务端响应）
    SubscribeAck {
        operation: String,  // "subscribe" | "unsubscribe" | "set"
        items: Vec<SubscriptionItem>,
        code: i32,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        failed_items: Option<Vec<FailedSubscriptionItem>>,
    },
    
    /// 标记已读（快速通道）
    MarkAsRead {
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        up_to: String,      // 读到哪条消息ID
    },
    
    /// 正在输入通知
    Typing {
        chat_type: String,  // "private_chat" | "group_chat"
        chat_id: String,
        is_typing: bool,
    },
}

/// 用户基本信息
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserInfo {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    /// 用户类型：temp | normal | admin | system
    pub user_type: crate::types::enums::UserType,
}

/// 群组更新字段
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct GroupUpdatedFields {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub settings: Option<serde_json::Value>,  // Partial<GroupSettings>
}

/// 订阅项定义
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SubscriptionItem {
    /// 数据源类型：group_chat | user | notification
    pub source_type: String,
    /// 数据源标识列表
    pub source_ids: Vec<String>,
}

/// 失败的订阅项
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct FailedSubscriptionItem {
    pub source_type: String,
    pub source_ids: Vec<String>,
    pub reason: String,
}
