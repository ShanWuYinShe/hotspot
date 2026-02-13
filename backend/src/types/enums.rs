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
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
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

/// 用户账号状态
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "user_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum UserStatus {
    Active,
    Inactive,
    Banned,
    Deleted,
}

pub type UserAccountStatus = UserStatus;

/// 用户在线状态 - API 文档: UserPresenceStatus
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum UserPresenceStatus {
    Online,
    Offline,
}

/// 性别
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "user_gender", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum UserGender {
    Male,
    Female,
    Other,
    Unknown,
}

pub type Gender = UserGender;

// ============================================================
// 互动相关枚举
// ============================================================

/// 反应类型
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "reaction_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum ReactionType {
    Like,
    Love,
    Haha,
    Wow,
    Sad,
    Angry,
}

// ============================================================
// 媒体相关枚举
// ============================================================

/// 媒体分类
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "media_category", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum MediaCategory {
    Image,
    Video,
    Audio,
    Document,
    Other,
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
            MediaCategory::Image => &["image/jpeg", "image/png", "image/webp", "image/gif"],
            MediaCategory::Video => &["video/mp4", "video/quicktime"],
            MediaCategory::Audio => &["audio/mpeg", "audio/aac", "audio/ogg"],
            MediaCategory::Document => &[],
            MediaCategory::Other => &[],
        }
    }

    pub fn max_file_size(&self) -> u64 {
        match self {
            MediaCategory::Avatar | MediaCategory::GroupAvatar => 5 * 1024 * 1024,
            MediaCategory::MomentImage | MediaCategory::ChatImage => 10 * 1024 * 1024,
            MediaCategory::ChatVideo => 100 * 1024 * 1024,
            MediaCategory::ChatAudio => 20 * 1024 * 1024,
            MediaCategory::ChatFile => 100 * 1024 * 1024,
            MediaCategory::Image => 10 * 1024 * 1024,
            MediaCategory::Video => 100 * 1024 * 1024,
            MediaCategory::Audio => 20 * 1024 * 1024,
            MediaCategory::Document => 100 * 1024 * 1024,
            MediaCategory::Other => 100 * 1024 * 1024,
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
#[sqlx(type_name = "request_status", rename_all = "lowercase")]
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
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
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

/// 消息类型
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
    Location,
    System,
}

pub type SendMessageType = MessageType;

/// 隐私级别
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
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
#[sqlx(type_name = "varchar", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum VerificationCodeType {
    Register,
    Login,
    ResetPassword,
    ChangeEmail,
    ChangePhone,
}

impl VerificationCodeType {
    pub fn as_str(&self) -> &'static str {
        match self {
            VerificationCodeType::Register => "register",
            VerificationCodeType::Login => "login",
            VerificationCodeType::ResetPassword => "reset_password",
            VerificationCodeType::ChangeEmail => "change_email",
            VerificationCodeType::ChangePhone => "change_phone",
        }
    }
}

// ============================================================
// Feed & Map
// ============================================================

#[derive(
    Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type, Display, EnumString,
)]
#[sqlx(type_name = "varchar", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum MapItemType {
    All,
    User,
    Group,
    Moment,
    Cluster,
}

impl MapItemType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MapItemType::All => "all",
            MapItemType::User => "user",
            MapItemType::Group => "group",
            MapItemType::Moment => "moment",
            MapItemType::Cluster => "cluster",
        }
    }
}

pub type MapDataType = MapItemType;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum FeedSource {
    All,
    Following,
    Follow,
    Nearby,
    Recommend,
    Hot,
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
    InvalidTopic,
    Forbidden,
    LimitExceeded,
    InternalError,
}

// ============================================================
// 分享与推荐
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "snake_case")]
#[strum(serialize_all = "snake_case")]
pub enum ShareType {
    Moment,
    User,
    Group,
    ReferencePoint,
}

impl ShareType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ShareType::Moment => "moment",
            ShareType::User => "user",
            ShareType::Group => "group",
            ShareType::ReferencePoint => "reference_point",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum RecommendationType {
    User,
    Group,
    Moment,
}

// ============================================================
// 其他
// ============================================================

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Display, EnumString)]
#[serde(rename_all = "lowercase")]
#[strum(serialize_all = "lowercase")]
pub enum FollowSource {
    Search,
    Nearby,
    Recommend,
    Share,
}
