use crate::dto::response::{GroupInfo, MessageMedia, MessageReplyTo, MomentSummary, UserInfo};
use crate::types::enums::{
    AnnouncementSeverity, ClientType, GroupDismissReason, GroupJoinMethod, GroupRole,
    MemberRemoveReason, MentionTargetType, MessageType, SubscriptionFailureReason,
    SubscriptionOperation,
};
use serde::{Deserialize, Serialize};

/// Serialize i64 as string, deserialize from string or number (spec: 所有 ID 在 API 层为 string)
mod ws_id_as_string {
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(id: &i64, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&id.to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<i64, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        #[serde(untagged)]
        enum StringOrNumber {
            Str(String),
            Num(i64),
        }
        match StringOrNumber::deserialize(deserializer)? {
            StringOrNumber::Str(s) => s.parse::<i64>().map_err(serde::de::Error::custom),
            StringOrNumber::Num(n) => Ok(n),
        }
    }
}

/// Serialize Option<Vec<i64>> as string[] or null, deserialize from string[] or number[]
mod ws_option_vec_id_as_string {
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(ids: &Option<Vec<i64>>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match ids {
            Some(ids) => {
                use serde::ser::SerializeSeq;
                let mut seq = serializer.serialize_seq(Some(ids.len()))?;
                for id in ids {
                    seq.serialize_element(&id.to_string())?;
                }
                seq.end()
            }
            None => serializer.serialize_none(),
        }
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Option<Vec<i64>>, D::Error>
    where
        D: Deserializer<'de>,
    {
        let opt: Option<Vec<serde_json::Value>> = Option::deserialize(deserializer)?;
        match opt {
            None => Ok(None),
            Some(vals) => {
                let mut ids = Vec::with_capacity(vals.len());
                for val in vals {
                    match val {
                        serde_json::Value::String(s) => {
                            ids.push(s.parse::<i64>().map_err(serde::de::Error::custom)?);
                        }
                        serde_json::Value::Number(n) => {
                            ids.push(
                                n.as_i64()
                                    .ok_or_else(|| serde::de::Error::custom("invalid id"))?,
                            );
                        }
                        _ => return Err(serde::de::Error::custom("invalid id type")),
                    }
                }
                Ok(Some(ids))
            }
        }
    }
}

/// WebSocket 消息类型定义
/// 严格遵循 api_spec.md WebSocket 实时事件定义
#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(tag = "type", content = "payload")]
pub enum WsMessage {
    // ========== 服务端推送事件 (Server → Client) ==========

    // ===== 连接管理事件 =====
    /// 连接建立确认
    ConnectionEstablished {
        connection_id: String,
        #[serde(with = "ws_id_as_string")]
        user_id: i64,
        server_time: String, // ISO 8601 UTC
        config: ConnectionConfig,
    },

    // ===== 在线状态事件 =====
    /// 用户上线（spec: user_id, online_at）
    UserOnline {
        #[serde(with = "ws_id_as_string")]
        user_id: i64,
        online_at: String, // ISO 8601 UTC
    },

    /// 用户离线
    UserOffline {
        #[serde(with = "ws_id_as_string")]
        user_id: i64,
        offline_at: String, // ISO 8601 UTC
    },

    // ===== 关注关系事件 =====
    /// 新增关注者（有人关注了我）
    /// 心理学优化：通过 source 让被关注者了解"为什么被关注"
    NewFollower {
        actor: UserInfo,
        followed_at: String, // ISO 8601 UTC
        #[serde(skip_serializing_if = "Option::is_none")]
        source: Option<crate::types::enums::FollowSource>,
        #[serde(skip_serializing_if = "Option::is_none")]
        source_moment_id: Option<String>,
    },

    /// 互相关注（成为好友）
    MutualFollow {
        actor: UserInfo,
        mutual_at: String, // ISO 8601 UTC
    },

    // ===== 生活瞬间互动事件 =====
    /// 生活瞬间被点赞
    MomentLiked {
        moment_id: String,
        moment_summary: MomentSummary,
        actor: UserInfo,
        liked_at: String, // ISO 8601 UTC
    },

    /// 生活瞬间被评论
    MomentCommented {
        moment_id: String,
        moment_summary: MomentSummary,
        comment_id: String,
        actor: UserInfo,
        content: String,
        commented_at: String, // ISO 8601 UTC
    },

    /// 评论被回复
    CommentReply {
        moment_id: String,
        moment_summary: MomentSummary,
        comment_id: String,
        replied_comment_preview: String,
        reply_id: String,
        actor: UserInfo,
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
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64, // 接收者ID (对于发送者多端同步=接收方ID)
        sender: UserInfo,
        #[serde(rename = "type")]
        message_type: MessageType,
        #[serde(skip_serializing_if = "Option::is_none")]
        content: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        share_code: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        sent_at: String, // ISO 8601 UTC
    },

    /// 私聊消息被编辑
    PrivateMessageEdited {
        message_id: String,
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64,
        content: String,
        edited_at: String, // ISO 8601 UTC
    },

    /// 私聊消息被撤回
    PrivateMessageRecalled {
        message_id: String,
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64,
        recalled_by: UserInfo,
        recalled_at: String, // ISO 8601 UTC
    },

    /// 私聊消息已读回执
    PrivateMessageRead {
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64,
        up_to_message_id: String, // 读到哪条消息ID
        #[serde(with = "ws_id_as_string")]
        reader_id: i64,
        read_at: String, // ISO 8601 UTC
    },

    /// 私聊对方正在输入
    PrivateUserTyping {
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64,
        #[serde(with = "ws_id_as_string")]
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
        share_code: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        media: Option<MessageMedia>,
        #[serde(skip_serializing_if = "Option::is_none")]
        reply_to: Option<MessageReplyTo>,
        #[serde(skip_serializing_if = "Option::is_none")]
        #[serde(default, with = "ws_option_vec_id_as_string")]
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
        #[serde(with = "ws_id_as_string")]
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
        group: GroupInfo,
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
        group: GroupInfo,
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
        target_type: MentionTargetType,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_id: Option<String>,
        moment_summary: MomentSummary,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_preview: Option<String>,
        actor: UserInfo,
        mentioned_at: String, // ISO 8601 UTC
    },

    // ===== 瞬间互动事件（续） =====
    /// 评论被点赞
    CommentLiked {
        comment_id: String,
        moment_id: String,
        moment_summary: MomentSummary,
        comment_preview: String,
        actor: UserInfo,
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
        removed_by: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        reason: Option<MemberRemoveReason>,
        removed_at: String,
        member_count: i32,
    },

    /// 成员被禁言
    GroupMemberMuted {
        group_id: String,
        user: UserInfo,
        muted_by: UserInfo,  // 操作者
        muted_until: String, // 禁言截止时间
        duration_minutes: i32,
        muted_at: String, // ISO 8601 UTC
    },

    /// 成员被解除禁言
    GroupMemberUnmuted {
        group_id: String,
        user: UserInfo,
        unmuted_by: UserInfo, // 操作者
        unmuted_at: String,   // ISO 8601 UTC
    },

    /// 群组信息更新
    GroupUpdated {
        group_id: String,
        updated_fields: GroupUpdatedFields,
        updated_by: String,
        updated_at: String,
    },

    /// 群组解散（群主主动解散或因违规被解散）
    /// 设计说明：不暴露系统管理员的存在，详细操作记录保留在审计日志中
    GroupDismissed {
        group_id: String,
        dismissed_at: String, // ISO 8601 UTC
        #[serde(skip_serializing_if = "Option::is_none")]
        reason: Option<GroupDismissReason>,
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

    // ⏳ 延期实现：多端同步事件（spec: api_spec_websocket.md）
    // 实现时添加以下变体：
    // SelfReadSync { chat_type, target_id, up_to_message_id? }
    // SelfChatSettingsSync { chat_type, target_id, is_muted?, is_pinned? }
    // SelfSettingsUpdated { settings: UserSettings }
    // SelfRelationSync { action, target_user_id }
    /// 错误消息
    Error {
        code: i32,
        message: String,
        severity: String, // "fatal" | "recoverable" | "warning"
        retryable: bool,
        #[serde(skip_serializing_if = "Option::is_none")]
        retry_after: Option<u32>,
        #[serde(skip_serializing_if = "Option::is_none")]
        context: Option<serde_json::Value>,
    },

    // ========== 客户端发送事件 (Client → Server) ==========
    /// 鉴权（握手后首帧，spec: client_type 必填, access_token App 端必填）
    Auth {
        client_type: ClientType,
        #[serde(skip_serializing_if = "Option::is_none")]
        access_token: Option<String>,
    },

    /// 心跳保活（spec: 空 payload）
    Ping {},

    /// 心跳响应（payload 为空对象）
    Pong {},

    /// 订阅管理（统一入口，通过 operation 区分操作类型）
    SetSubscriptions {
        client_request_id: String,        // 客户端生成，用于与 SubscribeAck 对齐
        operation: SubscriptionOperation, // subscribe / unsubscribe / set
        #[serde(rename = "type")]
        r#type: String,
        ids: Vec<String>, // spec: ids: string[]
    },

    /// 订阅确认（服务端响应）
    SubscribeAck {
        client_request_id: String, // 回传客户端请求ID
        operation: SubscriptionOperation,
        #[serde(rename = "type")]
        r#type: String,
        ids: Vec<String>, // spec: ids: string[]
        code: i32,
        #[serde(skip_serializing_if = "Option::is_none")]
        message: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        failed_ids: Option<Vec<FailedSubscriptionId>>,
    },

    /// 订阅状态快照（响应 GetSubscriptions）
    SubscriptionsSnapshot {
        client_request_id: String,
        user_ids: Vec<String>,
        group_ids: Vec<String>,
    },

    // ========== 客户端发送事件 (续) ==========
    /// 私聊正在输入
    PrivateTyping {
        #[serde(with = "ws_id_as_string")]
        target_user_id: i64,
        is_typing: bool,
    },

    /// 群聊正在输入
    GroupTyping { group_id: String, is_typing: bool },

    /// 查询当前订阅状态
    GetSubscriptions { client_request_id: String },
}

/// 失败的订阅ID — spec: { subscription_id: string, reason: SubscriptionFailureReason }
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct FailedSubscriptionId {
    pub subscription_id: String,
    pub reason: SubscriptionFailureReason,
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

/// GroupUpdated 事件中变更字段的强类型定义（spec: GroupUpdatedPayload.updated_fields）
#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct GroupUpdatedFields {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_method: Option<GroupJoinMethod>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_searchable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub has_password: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<crate::dto::response::common::Location>,
}

/// 从 AppError 构建 WebSocket Error 消息
pub fn ws_error_from_app_error(err: &crate::error::AppError) -> WsMessage {
    let (severity, retryable) = match err {
        crate::error::AppError::WebSocketTokenExpired => ("recoverable".to_string(), false),
        _ => ("fatal".to_string(), false),
    };

    WsMessage::Error {
        code: err.error_code(),
        message: err.to_string(),
        severity,
        retryable,
        retry_after: None,
        context: None,
    }
}
