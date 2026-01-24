use crate::dto::response::{Location, MessageMedia, MessageLocation, UserInfo};
use crate::types::enums::{
    AnnouncementSeverity, FollowRequestAction, GroupRole, MemberLeaveReason, MessageType,
    ReactionType, ResponseStatus, SubscriptionFailureReason, SubscriptionOperation,
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
        location: Location,
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

    /// 收到关注请求（当 allow_follow 设置为需要审批时）
    FollowRequestReceived {
        request_id: String,
        requester: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        requested_at: String, // ISO 8601 UTC
    },

    /// 关注请求被处理（发送给请求方）
    FollowRequestHandled {
        request_id: String,
        target_user: UserInfo, // 被请求关注的用户
        action: FollowRequestAction, // "accept" | "reject"
        handled_at: String,    // ISO 8601 UTC
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
        location: Option<MessageLocation>,
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
        has_reaction: bool,          // 操作后状态：true=添加了反应，false=移除了反应
        created_at: String,          // ISO 8601 UTC
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
        group_chat_id: String,
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: MessageType,
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        #[serde(skip_serializing_if = "Option::is_none")]
        location: Option<MessageLocation>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        #[serde(skip_serializing_if = "Option::is_none")]
        mentions: Option<Vec<i64>>,
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
        up_to: String,
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
        role: GroupRole,  // 使用 Enum
        member_count: i32, // 当前成员总数
        joined_at: String, // ISO 8601 UTC
    },

    /// 成员离开群组
    GroupMemberLeft {
        group_chat_id: String,
        user_id: i64,
        member_count: i32,                // 当前成员总数
        reason: MemberLeaveReason, // 使用 Enum
        left_at: String,                  // ISO 8601 UTC
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
        old_role: GroupRole, // 使用 Enum
        new_role: GroupRole, // 使用 Enum
        changed_at: String,   // ISO 8601 UTC
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
        status: ResponseStatus,
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
        status: ResponseStatus,
        responded_at: String, // ISO 8601 UTC
    },

    // ===== 群公告事件 =====
    /// 群公告创建
    GroupAnnouncementCreated {
        announcement_id: String,
        group_chat_id: String,
        title: String,
        content: String,
        created_by: UserInfo,
        is_pinned: bool,
        created_at: String, // ISO 8601 UTC
    },

    /// 群公告更新
    GroupAnnouncementUpdated {
        announcement_id: String,
        group_chat_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        title: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        is_pinned: Option<bool>,
        updated_at: String, // ISO 8601 UTC
    },

    /// 群公告删除
    GroupAnnouncementDeleted {
        announcement_id: String,
        group_chat_id: String,
        deleted_at: String, // ISO 8601 UTC
    },

    // ===== 提及事件 =====
    /// 瞬间中被 @ 提及
    MomentMentioned {
        moment_id: String,
        moment_summary: MomentSummary,
        user: UserInfo, // 提及我的人
        mentioned_at: String, // ISO 8601 UTC
    },

    /// 评论中被 @ 提及
    CommentMentioned {
        moment_id: String,
        comment_id: String,
        user: UserInfo, // 提及我的人
        content: String, // 评论内容摘要
        mentioned_at: String, // ISO 8601 UTC
    },

    // ===== 系统事件 =====
    /// 错误消息
    Error { code: i32, message: String },

    // ===== 多端同步 (Self-Sync) =====
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
    Subscribe {
        client_msg_id: String, // 客户端生成，用于与 SubscribeAck 对齐
        items: Vec<SubscriptionItem>,
    },

    /// 取消订阅
    Unsubscribe {
        client_msg_id: String, // 客户端生成，用于与 SubscribeAck 对齐
        items: Vec<SubscriptionItem>,
    },

    /// 替换式订阅（覆盖现有订阅）
    SetSubscriptions {
        client_msg_id: String, // 客户端生成，用于与 SubscribeAck 对齐
        items: Vec<SubscriptionItem>,
    },

    /// 订阅确认（服务端响应）
    SubscribeAck {
        client_msg_id: String, // 回传客户端消息ID
        operation: SubscriptionOperation,
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
    /// 数据源标识列表（按 source_type 约束类型）
    pub source_ids: Vec<SubscriptionSourceId>,
}

/// 订阅源ID：user 为 number，其他为 string
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(untagged)]
pub enum SubscriptionSourceId {
    String(String),
    Number(i64),
}

/// 失败的订阅项
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct FailedSubscriptionItem {
    pub source_type: String,
    pub source_ids: Vec<SubscriptionSourceId>,
    pub reason: SubscriptionFailureReason, // 使用 Enum
}
