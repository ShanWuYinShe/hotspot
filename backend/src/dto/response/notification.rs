//! 通知相关响应类型

use crate::types::enums::{AnnouncementSeverity, FollowSource, NotificationType};
use serde::{Deserialize, Serialize};

use super::user::UserInfo;

/// 瞬间摘要快照 - API 文档: MomentSummary
/// 用于通知负载中的 moment_summary 字段
#[derive(Debug, Serialize, Clone)]
pub struct MomentSummary {
    pub moment_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub text_preview: String,
}

/// 通知负载 - 按 type 区分的联合类型
/// 序列化为 JSON 时使用 serde(tag = "type", rename_all = "snake_case")
#[derive(Debug, Serialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum NotificationPayload {
    NewFollower {
        actor: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        source: Option<FollowSource>,
        #[serde(skip_serializing_if = "Option::is_none")]
        source_moment_id: Option<String>,
    },
    MutualFollow {
        actor: UserInfo,
    },
    MomentLiked {
        actor: UserInfo,
        moment_id: String,
        moment_summary: MomentSummary,
    },
    MomentCommented {
        actor: UserInfo,
        moment_id: String,
        moment_summary: MomentSummary,
        comment_id: String,
        comment_preview: String,
    },
    CommentLiked {
        actor: UserInfo,
        comment_id: String,
        comment_preview: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
    },
    CommentReply {
        actor: UserInfo,
        comment_id: String,
        comment_preview: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
    },
    Mentioned {
        actor: UserInfo,
        target_type: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_id: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_preview: Option<String>,
    },
    GroupInviteReceived {
        actor: UserInfo,
        group_id: String,
        group_name: String,
        invite_id: String,
    },
    GroupApplicationAccepted {
        group_id: String,
        group_name: String,
        application_id: String,
    },
    SystemAnnouncement {
        announcement_id: String,
        severity: AnnouncementSeverity,
    },
}

/// 通知项 - API 文档: NotificationItem
#[derive(Debug, Serialize)]
pub struct NotificationItem {
    pub notification_id: String,
    #[serde(rename = "type")]
    pub notification_type: NotificationType,
    pub is_read: bool,
    pub created_at: String,
    pub payload: NotificationPayload,
}

/// 未读通知计数
#[derive(Debug, Serialize)]
pub struct UnreadCountResult {
    pub total: i64,
    pub by_type: UnreadCountByType,
}

#[derive(Debug, Serialize)]
pub struct UnreadCountByType {
    pub social: i64,
    pub interaction: i64,
    pub group: i64,
    pub system: i64,
}

/// 通知订阅设置 - API 文档: SubscriptionsData
#[derive(Debug, Serialize, Deserialize)]
pub struct SubscriptionsData {
    pub new_follower_enabled: bool,
    pub mutual_follow_enabled: bool,
    pub follow_request_enabled: bool, // 收到关注请求（需审批时）
    pub follow_request_handled_enabled: bool, // 关注请求被处理
    pub group_invite_enabled: bool,
    pub group_invite_response_enabled: bool,
    pub group_application_enabled: bool,
    pub group_application_response_enabled: bool,
    pub group_announcement_enabled: bool,
    pub message_mention_enabled: bool,
    pub moment_like_enabled: bool,
    pub moment_comment_enabled: bool,
    pub comment_reply_enabled: bool,
    pub system_announcement_enabled: bool,
}
