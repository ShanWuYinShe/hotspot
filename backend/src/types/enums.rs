//! 数据库枚举类型定义
//!
//! 使用 strum 宏简化 Display 和 FromStr 实现
//! 对应 API 规范和数据库 Schema

use serde::{Deserialize, Serialize};
use sqlx::Type;
use strum_macros::{Display, EnumString};

// ============================================================
// 用户相关枚举
// ============================================================

/// 用户类型（仅表达账号生命周期，不含权限角色）
/// - Temp: 临时用户（访客，未注册）
/// - Normal: 正式用户（已注册）
/// 管理员身份由 RBAC 系统管理，不体现在 UserType 中
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "user_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum UserType {
    Temp,
    Normal,
}

impl Default for UserType {
    fn default() -> Self {
        UserType::Normal
    }
}

/// 用户账号状态（与 spec 对齐：normal | banned）
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "user_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum UserStatus {
    Normal,
    Banned,
}

/// API 层用户账号状态（spec: 'normal' | 'banned'）
/// 仅在 admin API 响应中使用
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum UserAccountStatus {
    Normal,
    Banned,
}

impl From<UserStatus> for UserAccountStatus {
    fn from(status: UserStatus) -> Self {
        match status {
            UserStatus::Banned => UserAccountStatus::Banned,
            _ => UserAccountStatus::Normal,
        }
    }
}

/// 用户在线状态 - API 文档: UserPresenceStatus
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum UserPresenceStatus {
    Online,
    Offline,
}

/// 性别（spec: male | female | apache_helicopter）
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "user_gender", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum UserGender {
    Male,
    Female,
    ApacheHelicopter,
}

pub type Gender = UserGender;

/// 不可发消息原因（spec: privacy | stranger_limit | account_suspended）
/// 设计说明：被拉黑场景统一返回 Privacy，不暴露拉黑状态（隐私保护）
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "cant_message_reason", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum CantMessageReason {
    Privacy,
    StrangerLimit,
    AccountSuspended,
}

// ============================================================
// 媒体相关枚举
// ============================================================

/// 媒体分类（spec: avatar | group_avatar | moment_image | chat_image | chat_video | chat_audio | chat_file）
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "media_category", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum MediaCategory {
    Avatar,
    GroupAvatar,
    MomentImage,
    ChatImage,
    ChatVideo,
    ChatAudio,
    ChatFile,
}

impl MediaCategory {
    pub fn allowed_mime_types(&self) -> &'static [&'static str] {
        match self {
            MediaCategory::Avatar | MediaCategory::GroupAvatar => {
                &["image/jpeg", "image/png", "image/webp"]
            }
            MediaCategory::MomentImage | MediaCategory::ChatImage => {
                &["image/jpeg", "image/png", "image/webp", "image/gif"]
            }
            MediaCategory::ChatVideo => &["video/mp4", "video/quicktime"],
            MediaCategory::ChatAudio => &["audio/mpeg", "audio/aac", "audio/ogg"],
            MediaCategory::ChatFile => &[],
        }
    }

    pub fn max_file_size(&self) -> u64 {
        match self {
            MediaCategory::Avatar | MediaCategory::GroupAvatar => 5 * 1024 * 1024,
            MediaCategory::MomentImage | MediaCategory::ChatImage => 10 * 1024 * 1024,
            MediaCategory::ChatVideo => 100 * 1024 * 1024,
            MediaCategory::ChatAudio => 20 * 1024 * 1024,
            MediaCategory::ChatFile => 100 * 1024 * 1024,
        }
    }

    pub fn validate_mime_type(&self, mime_type: &str) -> bool {
        let allowed = self.allowed_mime_types();
        if allowed.is_empty() {
            return true;
        }
        allowed.iter().any(|m| *m == mime_type)
    }

    pub fn validate_file_size(&self, size_bytes: u64) -> bool {
        size_bytes <= self.max_file_size()
    }
}

// ============================================================
// 群组与申请枚举
// ============================================================

/// 申请状态 (通用)
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "application_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum RequestStatus {
    Pending,
    Accepted,
    Rejected,
}

pub type ApplicationStatus = RequestStatus;

/// 申请状态过滤
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ApplicationStatusFilter {
    All,
    Pending,
    Accepted,
    Rejected,
}

/// 群组加入方式
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "group_join_method", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum GroupJoinMethod {
    Open,
    Approval,
    InviteOnly,
}

impl GroupJoinMethod {
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupJoinMethod::Open => "open",
            GroupJoinMethod::Approval => "approval",
            GroupJoinMethod::InviteOnly => "invite_only",
        }
    }
}

/// 群组角色
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "group_role", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum GroupRole {
    Owner,
    Admin,
    Member,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum SetableGroupRole {
    Admin,
    Member,
}

/// 成员离开原因
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum MemberLeaveReason {
    Quit,
    Kick,
    Dismiss,
}

// ============================================================
// 隐私与消息枚举
// ============================================================

/// 消息类型（spec: text | image | video | audio | file | share | system）
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum MessageType {
    Text,
    Image,
    Video,
    Audio,
    File,
    Share,
    System,
}

impl MessageType {
    /// 返回消息类型的预览占位文本，如 "[图片]", "[视频]" 等
    pub fn preview_text(&self) -> &'static str {
        match self {
            MessageType::Text => "[文本]",
            MessageType::Image => "[图片]",
            MessageType::Video => "[视频]",
            MessageType::Audio => "[语音]",
            MessageType::File => "[文件]",
            MessageType::Share => "[分享]",
            MessageType::System => "[系统消息]",
        }
    }
}

/// 可发送的消息类型（spec: Exclude<MessageType, 'system'>）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum SendMessageType {
    Text,
    Image,
    Video,
    Audio,
    File,
    Share,
}

impl From<SendMessageType> for MessageType {
    fn from(smt: SendMessageType) -> Self {
        match smt {
            SendMessageType::Text => MessageType::Text,
            SendMessageType::Image => MessageType::Image,
            SendMessageType::Video => MessageType::Video,
            SendMessageType::Audio => MessageType::Audio,
            SendMessageType::File => MessageType::File,
            SendMessageType::Share => MessageType::Share,
        }
    }
}

/// 隐私级别
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "privacy_level", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum PrivacyLevel {
    Public,
    Private,
}

/// 私聊权限设置
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum PrivateChatPermission {
    Everyone,
    Friends,
    Nobody,
}

// ============================================================
// 举报相关枚举
// ============================================================

/// 举报状态
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "report_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ReportStatus {
    Pending,
    Reviewing,
    Resolved,
    Dismissed,
}

impl ReportStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReportStatus::Pending => "pending",
            ReportStatus::Reviewing => "reviewing",
            ReportStatus::Resolved => "resolved",
            ReportStatus::Dismissed => "dismissed",
        }
    }
}

/// 举报对象类型
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "report_target_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ReportTargetType {
    User,
    Moment,
    Comment,
    Message,
    Group,
}

impl ReportTargetType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReportTargetType::User => "user",
            ReportTargetType::Moment => "moment",
            ReportTargetType::Comment => "comment",
            ReportTargetType::Message => "message",
            ReportTargetType::Group => "group",
        }
    }
}

/// 举报处理动作
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "report_handle_action", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ReportHandleAction {
    Resolve,
    Dismiss,
    Escalate,
}

/// 举报原因枚举 - API 文档: ReportReason
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum ReportReason {
    Spam,
    Harassment,
    Violence,
    HateSpeech,
    SexualContent,
    Misinformation,
    Impersonation,
    SelfHarm,
    Other,
}

// ============================================================
// 公告相关
// ============================================================

/// 公告严重程度/等级
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "announcement_severity", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum AnnouncementSeverity {
    Info,
    Warning,
    Error,
}

impl AnnouncementSeverity {
    pub fn as_str(&self) -> &'static str {
        match self {
            AnnouncementSeverity::Info => "info",
            AnnouncementSeverity::Warning => "warning",
            AnnouncementSeverity::Error => "error",
        }
    }
}

// ============================================================
// 平台与设备
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ClientType {
    App,
    Web,
}

// ============================================================
// 验证码
// ============================================================

#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "verification_code_type", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum VerificationCodeType {
    Register,
    Login,
    ResetPassword,
    ChangePhone,
}

// ============================================================
// Feed & Map
// ============================================================

/// 地图渲染项类型（spec: 'cluster' | 'user' | 'group' | 'moment'）
/// 用于 response 中的 MapRenderItem.type
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum MapItemType {
    Cluster,
    User,
    Group,
    Moment,
}

/// 地图数据过滤类型（spec: 'user' | 'group' | 'moment' | 'all'）
/// 用于 request 中的 MapFilter.data_type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum MapDataType {
    User,
    Group,
    Moment,
    All,
}

impl MapDataType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MapDataType::User => "user",
            MapDataType::Group => "group",
            MapDataType::Moment => "moment",
            MapDataType::All => "all",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum FeedSource {
    Following,
    Nearby,
    All,
}

// ============================================================
// 通知相关
// ============================================================

#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum NotificationType {
    NewFollower,
    MutualFollow,
    MomentLiked,
    MomentCommented,
    CommentLiked,
    CommentReply,
    Mentioned,
    GroupInviteReceived,
    GroupApplicationAccepted,
    SystemAnnouncement,
}

// ============================================================
// 系统状态
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum SystemHealthStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ResponseStatus {
    Accepted,
    Rejected,
}

impl ResponseStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ResponseStatus::Accepted => "accepted",
            ResponseStatus::Rejected => "rejected",
        }
    }
}

// ============================================================
// 标签与趋势
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum TagTrend {
    Rising,
    Falling,
    Stable,
}

// ============================================================
// WebSocket 订阅
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum SubscriptionOperation {
    Subscribe,
    Unsubscribe,
    Set,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum SubscriptionFailureReason {
    #[serde(rename = "invalid")]
    InvalidTopic,
    #[serde(rename = "denied")]
    Forbidden,
    #[serde(rename = "limit_exceeded")]
    LimitExceeded,
    #[serde(rename = "error")]
    InternalError,
}

// ============================================================
// 分享与推荐
// ============================================================

#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "share_type", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum ShareType {
    Moment,
    User,
    Group,
    Location,
}

#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "dismiss_target_type", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum DismissTargetType {
    Moment,
    User,
    Group,
}

impl ShareType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ShareType::Moment => "moment",
            ShareType::User => "user",
            ShareType::Group => "group",
            ShareType::Location => "location",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum RecommendationType {
    FriendLiked,
    Nearby,
    Trending,
    Following,
    TagMatch,
}

// ============================================================
// 其他
// ============================================================

/// 关注来源（spec: moment | search | recommendation | profile | group）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum FollowSource {
    Moment,
    Search,
    Recommendation,
    Profile,
    Group,
}
