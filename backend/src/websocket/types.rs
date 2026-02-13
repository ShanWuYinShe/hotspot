use crate::dto::response::{MessageMedia, UserInfo};
use crate::types::enums::{
    AnnouncementSeverity, GroupRole, MessageType, SubscriptionFailureReason, SubscriptionOperation,
};
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
        server_time: String, // ISO 8601 UTC
        config: ConnectionConfig,
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

    // ===== 关注关系事件 =====
    /// 新增关注者（有人关注了我）
    /// 心理学优化：通过 source 让被关注者了解"为什么被关注"
    NewFollower {
        user: UserInfo,
        followed_at: String, // ISO 8601 UTC
    },

    /// 互相关注（成为好友）
    MutualFollow {
        user: UserInfo,
        mutual_at: String, // ISO 8601 UTC
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
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>, // 所属瞬间的摘要（与 MomentCommented、CommentLiked 保持一致）
        comment_id: String,       // 原评论ID（被回复的评论）
        reply_to_content: String, // 原评论内容摘要
        reply_id: String,         // 回复评论ID
        user: UserInfo,           // 回复者
        content: String,
        replied_at: String, // ISO 8601 UTC
    },

    /// 系统公告
    SystemAnnouncement {
        announcement_id: String,
        title: String,
        content: String,
        severity: AnnouncementSeverity,
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
        message_type: MessageType,
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
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
        recalled_by: UserInfo,
        recalled_at: String, // ISO 8601 UTC
    },

    /// 私聊消息已读回执
    PrivateMessageRead {
        target_user_id: i64,
        up_to: String, // 读到哪条消息ID
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
        group_id: String,
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: MessageType,
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        #[serde(skip_serializing_if = "Option::is_none")]
        mentions: Option<Vec<i64>>,
        sent_at: String, // ISO 8601 UTC
    },

    /// 群聊消息被编辑
    GroupMessageEdited {
        message_id: String,
        group_id: String,
        content: String,
        edited_at: String, // ISO 8601 UTC
    },

    /// 群聊消息被撤回
    GroupMessageRecalled {
        message_id: String,
        group_id: String,
        recalled_by: UserInfo,
        recalled_at: String, // ISO 8601 UTC
    },

    /// 群聊成员正在输入
    GroupUserTyping {
        group_id: String,
        user_id: i64,
        is_typing: bool,
    },

    // ===== 群组事件 =====
    /// 成员加入群组
    GroupMemberJoined {
        group_id: String,
        user: UserInfo,
        role: GroupRole,   // 使用 Enum
        member_count: i32, // 当前成员总数
        joined_at: String, // ISO 8601 UTC
    },

    /// 收到群组邀请
    GroupInviteReceived {
        invite_id: String,
        group_id: String,
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
        group_id: String,
        user: UserInfo,
        status: String,       // 固定为 "accepted"
        responded_at: String, // ISO 8601 UTC
    },

    /// 收到群组申请
    GroupApplicationReceived {
        application_id: String,
        group_id: String,
        applicant: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        created_at: String, // ISO 8601 UTC
    },

    /// 群组申请响应
    GroupApplicationResponse {
        application_id: String,
        group_id: String,
        group_name: String,
        status: String,       // 固定为 "accepted"
        responded_at: String, // ISO 8601 UTC
    },

    // ===== 群公告事件 =====
    /// 群公告创建
    GroupAnnouncementCreated {
        announcement_id: String,
        group_id: String,
        title: String,
        content: String,
        created_by: UserInfo,
        is_pinned: bool,
        created_at: String, // ISO 8601 UTC
    },

    // ===== 提及事件 =====
    /// 瞬间或评论中被 @ 提及
    MessageMentioned {
        target_type: String, // "moment" | "comment"
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_id: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_preview: Option<String>,
        user: UserInfo,       // 提及我的人
        mentioned_at: String, // ISO 8601 UTC
    },

    // ===== 瞬间互动事件（续） =====
    /// 评论被点赞
    CommentLiked {
        comment_id: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
        comment_preview: String,
        user: UserInfo,
        liked_at: String, // ISO 8601 UTC
    },

    // ===== 群组事件（续） =====
    /// 成员主动离开群组
    GroupMemberLeft {
        group_id: String,
        user: UserInfo,
        left_at: String,   // ISO 8601 UTC
        member_count: i32, // 剩余成员总数
    },

    /// 成员被移除（踢出/封禁）
    GroupMemberRemoved {
        group_id: String,
        user: UserInfo,
        removed_by: String, // user_id（操作者）
        #[serde(skip_serializing_if = "Option::is_none")]
        reason: Option<String>, // "kicked" | "banned"
        removed_at: String, // ISO 8601 UTC
        member_count: i32,
    },

    /// 成员被禁言
    GroupMemberMuted {
        group_id: String,
        user: UserInfo,
        muted_by: String,    // user_id（操作者）
        muted_until: String, // 禁言截止时间
        duration_minutes: i32,
        muted_at: String, // ISO 8601 UTC
    },

    /// 成员被解除禁言
    GroupMemberUnmuted {
        group_id: String,
        user: UserInfo,
        unmuted_by: String, // user_id（操作者）
        unmuted_at: String, // ISO 8601 UTC
    },

    /// 群组信息更新
    GroupUpdated {
        group_id: String,
        updated_fields: serde_json::Value, // 只包含变更的字段
        updated_by: String,                // user_id（操作者）
        updated_at: String,                // ISO 8601 UTC
    },

    /// 群组解散（群主主动解散或因违规被解散）
    /// 设计说明：不暴露系统管理员的存在，详细操作记录保留在审计日志中
    GroupDismissed {
        group_id: String,
        dismissed_at: String, // ISO 8601 UTC
        #[serde(skip_serializing_if = "Option::is_none")]
        reason: Option<String>, // "owner_dismissed" | "policy_violation"
    },

    // ===== 用户事件 =====
    /// 用户资料变更
    UserProfileUpdated {
        user: UserInfo,
        updated_at: String, // ISO 8601 UTC
    },

    // ===== 系统事件 =====
    /// 账号被封禁
    AccountBanned {
        reason: String,
        duration_days: i32, // 0=永久
        banned_at: String,  // ISO 8601 UTC
    },

    /// 错误消息
    Error { code: i32, message: String },

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

    /// 订阅管理（统一入口，通过 operation 区分操作类型）
    SetSubscriptions {
        client_msg_id: String,            // 客户端生成，用于与 SubscribeAck 对齐
        operation: SubscriptionOperation, // subscribe / unsubscribe / set
        #[serde(rename = "type")]
        r#type: String,
        ids: Vec<SubscriptionSourceId>,
    },

    /// 订阅确认（服务端响应）
    SubscribeAck {
        client_msg_id: String, // 回传客户端消息ID
        operation: SubscriptionOperation,
        #[serde(rename = "type")]
        r#type: String,
        ids: Vec<SubscriptionSourceId>,
        code: i32,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        failed_ids: Option<Vec<FailedSubscriptionId>>,
    },

    /// 订阅状态快照（响应 GetSubscriptions）
    SubscriptionsSnapshot {
        client_msg_id: String,
        user_ids: Vec<String>,
        group_ids: Vec<String>,
    },

    // ========== 客户端发送事件 (续) ==========
    /// 私聊正在输入
    PrivateTyping {
        target_user_id: i64,
        is_typing: bool,
    },

    /// 群聊正在输入
    GroupTyping { group_id: String, is_typing: bool },

    /// 查询当前订阅状态
    GetSubscriptions { client_msg_id: String },
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
/// 亦在 api_spec_types.mdc 中定义，保持与 spec 一致
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MomentSummary {
    pub moment_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub text_preview: String,
}

/// 订阅源ID：user 为 number，其他为 string
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(untagged)]
pub enum SubscriptionSourceId {
    String(String),
    Number(i64),
}

/// 失败的订阅ID
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(untagged)]
pub enum FailedSubscriptionId {
    Number {
        id: i64,
        reason: SubscriptionFailureReason, // 使用 Enum
    },
    String {
        id: String,
        reason: SubscriptionFailureReason, // 使用 Enum
    },
}

/// ConnectionEstablished 中的服务端配置（spec: config）
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ConnectionConfig {
    pub ping_interval: u32, // 心跳间隔（秒）
    pub ping_timeout: u32,  // Pong 超时（秒）
    pub max_subscriptions: MaxSubscriptions,
}

/// 最大订阅数限制
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MaxSubscriptions {
    pub user: u32,  // 用户订阅上限
    pub group: u32, // 群组订阅上限
}
