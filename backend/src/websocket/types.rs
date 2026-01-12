use serde::{Deserialize, Serialize};
use crate::dto::response::{MessageMedia, UserInfo};
use crate::types::enums::{
    GroupMemberRemovalReason, InviteStatus, MemberRole, MessageType, ReactionType,
    SubscriptionFailureReason,
};

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
        server_time: String, // ISO 8601 UTC
    },

    // ===== 在线状态事件 =====
    /// 用户上线
    UserOnline {
        user_id: i64,
        nickname: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        avatar_url: Option<String>,
        online_at: String, // ISO 8601 UTC
    },

    /// 用户离线
    UserOffline {
        user_id: i64,
        offline_at: String, // ISO 8601 UTC
    },

    /// 用户位置更新
    UserLocationUpdate {
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        updated_at: String, // ISO 8601 UTC
    },

    // ===== 关注关系事件 =====
    /// 新增关注者（有人关注了我）
    NewFollower {
        user: UserInfo,
        followed_at: String, // ISO 8601 UTC
        is_mutual: bool,     // 是否互相关注（成为好友）
    },

    /// 互相关注（成为好友）
    MutualFollow {
        user: UserInfo,
        mutual_at: String, // ISO 8601 UTC
    },

    /// 取消关注（好友关系解除）
    Unfollowed {
        user_id: i64,
        unfollowed_at: String, // ISO 8601 UTC
        was_mutual: bool,      // 之前是否互相关注
    },

    // ===== 用户资料更新 =====
    /// 用户资料更新 (头像/昵称)
    UserProfileUpdated {
        user_id: i64,
        #[serde(skip_serializing_if = "Option::is_none")]
        nickname: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        avatar_url: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        bio: Option<String>,
        updated_at: String, // ISO 8601 UTC
    },

    // ===== 生活瞬间互动事件 =====
    /// 生活瞬间被点赞
    MomentLiked {
        moment_id: String,
        moment_summary: MomentSummary,
        user: UserInfo,
        liked_at: String, // ISO 8601 UTC
    },

    /// 生活瞬间被评论
    MomentCommented {
        moment_id: String,
        moment_summary: MomentSummary,
        comment_id: String,
        user: UserInfo,
        content: String,
        commented_at: String, // ISO 8601 UTC
    },

    /// 评论被回复
    CommentReply {
        moment_id: String,
        comment_id: String, // 原评论ID（被回复的评论）
        reply_to_content: String, // 原评论内容摘要
        reply_id: String,   // 回复评论ID
        user: UserInfo,     // 回复者
        content: String,
        replied_at: String, // ISO 8601 UTC
    },

    /// 系统公告
    SystemAnnouncement {
        announcement_id: String,
        title: String,
        content: String,
        severity: String, // "info" | "warning" | "error"
        #[serde(skip_serializing_if = "Option::is_none")]
        action_url: Option<String>,
        created_at: String, // ISO 8601 UTC
    },

    // ===== 聊天消息事件 =====
    // ----- 私聊 -----
    /// 新私聊消息
    PrivateMessageNew {
        message_id: String,
        target_user_id: i64, // 接收者ID (对于发送者多端同步=接收方ID)
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: MessageType, // 使用 Enum
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        sent_at: String, // ISO 8601 UTC
    },

    /// 私聊消息被编辑
    PrivateMessageEdited {
        message_id: String,
        target_user_id: i64,
        content: String,
        edited_at: String, // ISO 8601 UTC
    },

    /// 私聊消息被撤回
    PrivateMessageRecalled {
        message_id: String,
        target_user_id: i64,
        recalled_by: i64,
        recalled_at: String, // ISO 8601 UTC
    },

    /// 私聊消息被删除
    PrivateMessageDeleted {
        message_id: String,
        target_user_id: i64,
        deleted_by: i64,
        deleted_at: String, // ISO 8601 UTC
    },

    /// 私聊消息反应
    PrivateMessageReaction {
        message_id: String,
        target_user_id: i64,
        user: UserInfo,
        reaction_type: ReactionType, // 使用 Enum
        has_reaction: bool,    // 操作后状态：true=添加了反应，false=移除了反应
        created_at: String,    // ISO 8601 UTC
    },

    /// 私聊消息已读回执
    PrivateMessageRead {
        target_user_id: i64,
        up_to_message_id: String, // 读到哪条消息ID
        reader_id: i64,
        read_at: String, // ISO 8601 UTC
    },

    /// 私聊对方正在输入
    PrivateUserTyping {
        target_user_id: i64,
        user_id: i64,
        is_typing: bool,
    },

    // ----- 群聊 -----
    /// 新群聊消息
    GroupMessageNew {
        message_id: String,
        group_chat_id: String,
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: MessageType, // 使用 Enum
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        #[serde(skip_serializing_if = "Option::is_none")]
        mentions: Option<Vec<i64>>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        sent_at: String, // ISO 8601 UTC
    },

    /// 群聊消息被编辑
    GroupMessageEdited {
        message_id: String,
        group_chat_id: String,
        content: String,
        edited_at: String, // ISO 8601 UTC
    },

    /// 群聊消息被撤回
    GroupMessageRecalled {
        message_id: String,
        group_chat_id: String,
        recalled_by: i64,
        recalled_at: String, // ISO 8601 UTC
    },

    /// 群聊消息被删除
    GroupMessageDeleted {
        message_id: String,
        group_chat_id: String,
        deleted_by: i64,
        deleted_at: String, // ISO 8601 UTC
    },

    /// 群聊消息反应
    GroupMessageReaction {
        message_id: String,
        group_chat_id: String,
        user: UserInfo,
        reaction_type: ReactionType, // 使用 Enum
        has_reaction: bool,
        created_at: String,
    },

    /// 群聊消息已读回执
    GroupMessageRead {
        group_chat_id: String,
        up_to_message_id: String,
        reader_id: i64,
        read_at: String,
    },

    /// 群聊成员正在输入
    GroupUserTyping {
        group_chat_id: String,
        user_id: i64,
        is_typing: bool,
    },

    // ===== 群组事件 =====
    /// 成员加入群组
    GroupMemberJoined {
        group_chat_id: String,
        user: UserInfo,
        role: MemberRole, // 使用 Enum
        member_count: i32, // 当前成员总数
        joined_at: String, // ISO 8601 UTC
    },

    /// 成员离开群组
    GroupMemberLeft {
        group_chat_id: String,
        user_id: i64,
        member_count: i32, // 当前成员总数
        reason: GroupMemberRemovalReason, // 使用 Enum
        left_at: String, // ISO 8601 UTC
    },

    /// 群组资料更新
    GroupProfileUpdated {
        group_chat_id: String,
        updated_fields: GroupUpdatedFields,
        updated_at: String, // ISO 8601 UTC
    },

    /// 成员角色变更
    GroupMemberRoleChanged {
        group_chat_id: String,
        user_id: i64,
        old_role: MemberRole,   // 使用 Enum
        new_role: MemberRole,   // 使用 Enum
        changed_at: String, // ISO 8601 UTC
    },

    /// 成员被禁言
    GroupMemberMuted {
        group_chat_id: String,
        user_id: i64,
        muted_until: String, // ISO 8601 UTC
        muted_at: String,    // ISO 8601 UTC
    },

    /// 成员取消禁言
    GroupMemberUnmuted {
        group_chat_id: String,
        user_id: i64,
        unmuted_at: String, // ISO 8601 UTC
    },

    /// 群组解散
    GroupDismissed {
        group_chat_id: String,
        dismissed_by: i64,
        dismissed_at: String, // ISO 8601 UTC
    },

    /// 收到群组邀请
    GroupInviteReceived {
        invite_id: String,
        group_chat_id: String,
        group_name: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        group_avatar_url: Option<String>,
        inviter: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        created_at: String, // ISO 8601 UTC
    },

    /// 群组邀请响应
    GroupInviteResponse {
        invite_id: String,
        group_chat_id: String,
        user: UserInfo,
        status: InviteStatus, // 使用 Enum
        responded_at: String, // ISO 8601 UTC
    },

    /// 收到群组申请
    GroupApplicationReceived {
        application_id: String,
        group_chat_id: String,
        applicant: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        created_at: String, // ISO 8601 UTC
    },

    /// 群组申请响应
    GroupApplicationResponse {
        application_id: String,
        group_chat_id: String,
        group_name: String,
        status: InviteStatus, // 使用 Enum (Application 状态通常是 Pending/Accepted/Rejected，复用 InviteStatus 或单独定义)
        // Check API Spec: GroupApplicationResponse status: 'accepted' | 'rejected'
        // InviteStatus has 'Accepted' and 'Rejected'
        // OK to reuse InviteStatus
        responded_at: String, // ISO 8601 UTC
    },

    // ===== 系统事件 =====
    /// 错误消息
    Error { code: i32, message: String },

    // ===== 多端同步 (Self-Sync) =====
    /// 自己在其他设备加入了群组
    SelfJoinedGroup {
        group: serde_json::Value, // GroupSearchItem
        joined_at: String,
    },
    /// 自己在其他设备退出了群组
    SelfLeftGroup {
        group_chat_id: String,
        left_at: String,
    },
    /// 私聊会话被删除
    PrivateChatDeleted {
        target_user_id: i64,
        deleted_at: String,
    },
    /// 私聊设置更新
    PrivateChatSettingsUpdated {
        target_user_id: i64,
        #[serde(skip_serializing_if = "Option::is_none")]
        is_muted: Option<bool>,
        #[serde(skip_serializing_if = "Option::is_none")]
        is_pinned: Option<bool>,
        updated_at: String,
    },
    /// 通知已读状态更新
    NotificationReadStateUpdated {
        total_unread_count: i32,
        read_at: String,
    },

    // ========== 客户端发送事件 (Client → Server) ==========
    /// 鉴权（握手后首帧）
    Auth {
        access_token: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        device_fingerprint: Option<String>,
    },

    /// 心跳保活
    Ping {
        timestamp: String, // ISO 8601 UTC
    },

    /// 心跳响应
    Pong {
        timestamp: String, // ISO 8601 UTC
    },

    /// 订阅（增量订阅）
    Subscribe { items: Vec<SubscriptionItem> },

    /// 取消订阅
    Unsubscribe { items: Vec<SubscriptionItem> },

    /// 替换式订阅（覆盖现有订阅）
    SetSubscriptions { items: Vec<SubscriptionItem> },

    /// 订阅确认（服务端响应）
    SubscribeAck {
        operation: String, // "subscribe" | "unsubscribe" | "set"
        items: Vec<SubscriptionItem>,
        code: i32,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        failed_items: Option<Vec<FailedSubscriptionItem>>,
    },

    /// 私聊正在输入
    PrivateTyping {
        target_user_id: i64,
        is_typing: bool,
    },

    /// 群聊正在输入
    GroupTyping {
        group_chat_id: String,
        is_typing: bool,
    },
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
    pub settings: Option<serde_json::Value>, // Partial<GroupSettings>
}

/// 消息回复引用快照
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MessageReplyTo {
    pub message_id: String,
    pub user_id: i64,
    pub nickname: String,
    pub content: String,
}

/// 瞬间摘要快照
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MomentSummary {
    pub moment_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub text_preview: String,
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
    pub reason: SubscriptionFailureReason, // 使用 Enum
}
