//! 数据库枚举类型定义
//!
//! 这些枚举类型与 PostgreSQL ENUM 类型一一对应
//! 部分枚举暂未使用，但为未来功能预留

use serde::{Deserialize, Serialize};
use sqlx::Type;

// ============================================================
// 1. 成员角色枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "member_role", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum MemberRole {
    Owner,  // 群主
    Admin,  // 管理员
    Member, // 普通成员
}

impl MemberRole {
    pub fn as_str(&self) -> &'static str {
        match self {
            MemberRole::Owner => "owner",
            MemberRole::Admin => "admin",
            MemberRole::Member => "member",
        }
    }

    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "owner" => Ok(MemberRole::Owner),
            "admin" => Ok(MemberRole::Admin),
            "member" => Ok(MemberRole::Member),
            _ => Err(format!("无效的成员角色: {}", s)),
        }
    }
}

impl std::fmt::Display for MemberRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 2. 消息类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "message_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum MessageType {
    Any,      // 🔍 匹配所有值（仅用于查询过滤）
    Text,     // 文本
    Image,    // 图片
    Video,    // 视频
    Audio,    // 音频
    File,     // 文件
    Location, // 位置
    System,   // 系统消息
}

impl MessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MessageType::Any => "any",
            MessageType::Text => "text",
            MessageType::Image => "image",
            MessageType::Video => "video",
            MessageType::Audio => "audio",
            MessageType::File => "file",
            MessageType::Location => "location",
            MessageType::System => "system",
        }
    }

    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_lowercase().as_str() {
            "any" => Some(MessageType::Any),
            "text" => Some(MessageType::Text),
            "image" => Some(MessageType::Image),
            "video" => Some(MessageType::Video),
            "audio" => Some(MessageType::Audio),
            "file" => Some(MessageType::File),
            "location" => Some(MessageType::Location),
            "system" => Some(MessageType::System),
            _ => None,
        }
    }

    pub fn is_media(&self) -> bool {
        matches!(
            self,
            MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File
        )
    }
}

impl std::fmt::Display for MessageType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 6. 举报状态枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "report_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReportStatus {
    Any,       // 🔍 匹配所有值（仅用于查询过滤）
    Pending,   // 待处理
    Reviewing, // 审核中
    Resolved,  // 已解决
    Dismissed, // 已驳回
}

impl ReportStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReportStatus::Any => "any",
            ReportStatus::Pending => "pending",
            ReportStatus::Reviewing => "reviewing",
            ReportStatus::Resolved => "resolved",
            ReportStatus::Dismissed => "dismissed",
        }
    }
}

impl std::fmt::Display for ReportStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReportStatus {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "any" => Ok(ReportStatus::Any),
            "pending" => Ok(ReportStatus::Pending),
            "reviewing" => Ok(ReportStatus::Reviewing),
            "resolved" => Ok(ReportStatus::Resolved),
            "dismissed" => Ok(ReportStatus::Dismissed),
            _ => Err(format!("无效的举报状态: {}", s)),
        }
    }
}

// ============================================================
// 7. 邀请状态枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "invite_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum InviteStatus {
    Any,      // 🔍 匹配所有值（仅用于查询过滤）
    Pending,  // 待处理
    Accepted, // 已接受
    Rejected, // 已拒绝
    Approved, // 已批准（用于申请）
    Expired,  // 已过期
}

impl InviteStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            InviteStatus::Any => "any",
            InviteStatus::Pending => "pending",
            InviteStatus::Accepted => "accepted",
            InviteStatus::Rejected => "rejected",
            InviteStatus::Approved => "approved",
            InviteStatus::Expired => "expired",
        }
    }
}

impl std::fmt::Display for InviteStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for InviteStatus {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "any" => Ok(InviteStatus::Any),
            "pending" => Ok(InviteStatus::Pending),
            "accepted" => Ok(InviteStatus::Accepted),
            "rejected" => Ok(InviteStatus::Rejected),
            "approved" => Ok(InviteStatus::Approved),
            "expired" => Ok(InviteStatus::Expired),
            _ => Err(format!("无效的邀请状态: {}", s)),
        }
    }
}

// ============================================================
// 8. 举报目标类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "report_target_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReportTargetType {
    User,    // 用户
    Moment,  // 生活瞬间
    Comment, // 评论
    Message, // 消息
    Group,   // 群组
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

impl std::fmt::Display for ReportTargetType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReportTargetType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "user" => Ok(ReportTargetType::User),
            "moment" => Ok(ReportTargetType::Moment),
            "comment" => Ok(ReportTargetType::Comment),
            "message" => Ok(ReportTargetType::Message),
            "group" => Ok(ReportTargetType::Group),
            _ => Err(format!("无效的举报目标类型: {}", s)),
        }
    }
}

// ============================================================
// 9. 平台枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "platform", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum Platform {
    Ios,     // iOS
    Android, // Android
    Web,     // Web
}

impl Platform {
    pub fn as_str(&self) -> &'static str {
        match self {
            Platform::Ios => "ios",
            Platform::Android => "android",
            Platform::Web => "web",
        }
    }
}

impl std::fmt::Display for Platform {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for Platform {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "ios" => Ok(Platform::Ios),
            "android" => Ok(Platform::Android),
            "web" => Ok(Platform::Web),
            _ => Err(format!("无效的平台: {}", s)),
        }
    }
}

// ============================================================
// 10. 隐私级别枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "privacy_level", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum PrivacyLevel {
    Public,  // 公开
    Friends, // 仅好友
    Private, // 私密
}

impl PrivacyLevel {
    pub fn as_str(&self) -> &'static str {
        match self {
            PrivacyLevel::Public => "public",
            PrivacyLevel::Friends => "friends",
            PrivacyLevel::Private => "private",
        }
    }
}

impl std::fmt::Display for PrivacyLevel {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for PrivacyLevel {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "public" => Ok(PrivacyLevel::Public),
            "friends" => Ok(PrivacyLevel::Friends),
            "private" => Ok(PrivacyLevel::Private),
            _ => Err(format!("无效的隐私级别: {}", s)),
        }
    }
}

// ============================================================
// 11. 用户类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "user_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum UserType {
    Temp,   // 临时用户
    Normal, // 普通用户
    Admin,  // 管理员用户
    System, // 系统用户
}

impl UserType {
    pub fn as_str(&self) -> &'static str {
        match self {
            UserType::Temp => "temp",
            UserType::Normal => "normal",
            UserType::Admin => "admin",
            UserType::System => "system",
        }
    }
}

impl std::fmt::Display for UserType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 13. 群组成员移除原因枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum GroupMemberRemovalReason {
    Left,   // 主动退出
    Kicked, // 被移除
    Banned, // 被封禁
}

impl GroupMemberRemovalReason {
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupMemberRemovalReason::Left => "left",
            GroupMemberRemovalReason::Kicked => "kicked",
            GroupMemberRemovalReason::Banned => "banned",
        }
    }
}

impl std::fmt::Display for GroupMemberRemovalReason {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 14. 地图资源类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MapResourceType {
    All,    // 所有
    User,   // 用户
    Group,  // 群组
    Moment, // 瞬间
}

impl MapResourceType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MapResourceType::All => "all",
            MapResourceType::User => "user",
            MapResourceType::Group => "group",
            MapResourceType::Moment => "moment",
        }
    }
}

impl std::fmt::Display for MapResourceType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for MapResourceType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "all" => Ok(MapResourceType::All),
            "user" => Ok(MapResourceType::User),
            "group" => Ok(MapResourceType::Group),
            "moment" => Ok(MapResourceType::Moment),
            _ => Err(format!("无效的地图资源类型: {}", s)),
        }
    }
}

// ============================================================
// 21. 信息流反馈操作枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FeedFeedbackAction {
    Hide,          // 隐藏
    NotInterested, // 不感兴趣
    Clicked,       // 点击
    Viewed,        // 查看
}

impl FeedFeedbackAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedFeedbackAction::Hide => "hide",
            FeedFeedbackAction::NotInterested => "not_interested",
            FeedFeedbackAction::Clicked => "clicked",
            FeedFeedbackAction::Viewed => "viewed",
        }
    }

    /// 判断是否应从 feed 中移除
    pub fn should_remove_from_feed(&self) -> bool {
        matches!(
            self,
            FeedFeedbackAction::Hide | FeedFeedbackAction::NotInterested
        )
    }
}

impl std::fmt::Display for FeedFeedbackAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for FeedFeedbackAction {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "hide" => Ok(FeedFeedbackAction::Hide),
            "not_interested" => Ok(FeedFeedbackAction::NotInterested),
            "clicked" => Ok(FeedFeedbackAction::Clicked),
            "viewed" => Ok(FeedFeedbackAction::Viewed),
            _ => Err(format!("无效的反馈操作: {}", s)),
        }
    }
}

// ============================================================
// 22. 私聊权限枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "chat_permission", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ChatPermission {
    Everyone, // 所有人
    Friends,  // 仅好友
    Nobody,   // 禁止
}

impl ChatPermission {
    pub fn as_str(&self) -> &'static str {
        match self {
            ChatPermission::Everyone => "everyone",
            ChatPermission::Friends => "friends",
            ChatPermission::Nobody => "nobody",
        }
    }
}

impl std::fmt::Display for ChatPermission {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ChatPermission {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "everyone" => Ok(ChatPermission::Everyone),
            "friends" => Ok(ChatPermission::Friends),
            "nobody" => Ok(ChatPermission::Nobody),
            _ => Err(format!("无效的私聊权限: {}", s)),
        }
    }
}

// ============================================================
// 23. 通知 API 类型枚举 (用于 MarkNotificationsReadRequest)
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum NotificationApiType {
    Like,
    Comment,
    Follow,
    System,
    Mention,
}

impl NotificationApiType {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationApiType::Like => "like",
            NotificationApiType::Comment => "comment",
            NotificationApiType::Follow => "follow",
            NotificationApiType::System => "system",
            NotificationApiType::Mention => "mention",
        }
    }
}

impl std::fmt::Display for NotificationApiType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for NotificationApiType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "like" => Ok(NotificationApiType::Like),
            "comment" => Ok(NotificationApiType::Comment),
            "follow" => Ok(NotificationApiType::Follow),
            "system" => Ok(NotificationApiType::System),
            "mention" => Ok(NotificationApiType::Mention),
            _ => Err(format!("无效的通知API类型: {}", s)),
        }
    }
}

// ============================================================
// 24. 反应类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "reaction_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReactionType {
    Like,
    Love,
    Laugh,
    Angry,
    Sad,
    Wow,
}

impl ReactionType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReactionType::Like => "like",
            ReactionType::Love => "love",
            ReactionType::Laugh => "laugh",
            ReactionType::Angry => "angry",
            ReactionType::Sad => "sad",
            ReactionType::Wow => "wow",
        }
    }
}

impl std::fmt::Display for ReactionType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReactionType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "like" => Ok(ReactionType::Like),
            "love" => Ok(ReactionType::Love),
            "laugh" => Ok(ReactionType::Laugh),
            "angry" => Ok(ReactionType::Angry),
            "sad" => Ok(ReactionType::Sad),
            "wow" => Ok(ReactionType::Wow),
            _ => Err(format!("无效的反应类型: {}", s)),
        }
    }
}

// ============================================================
// 25. 订阅失败原因枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SubscriptionFailureReason {
    InvalidItem,          // 订阅项无效
    NotFound,             // 订阅目标不存在
    PermissionDenied,     // 权限不足
    TooManySubscriptions, // 订阅数量超限
    RateLimited,          // 触发限流
    InternalError,        // 内部错误
}

impl SubscriptionFailureReason {
    pub fn as_str(&self) -> &'static str {
        match self {
            SubscriptionFailureReason::InvalidItem => "invalid_item",
            SubscriptionFailureReason::NotFound => "not_found",
            SubscriptionFailureReason::PermissionDenied => "permission_denied",
            SubscriptionFailureReason::TooManySubscriptions => "too_many_subscriptions",
            SubscriptionFailureReason::RateLimited => "rate_limited",
            SubscriptionFailureReason::InternalError => "internal_error",
        }
    }
}

impl std::fmt::Display for SubscriptionFailureReason {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for SubscriptionFailureReason {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "invalid_item" => Ok(SubscriptionFailureReason::InvalidItem),
            "not_found" => Ok(SubscriptionFailureReason::NotFound),
            "permission_denied" => Ok(SubscriptionFailureReason::PermissionDenied),
            "too_many_subscriptions" => Ok(SubscriptionFailureReason::TooManySubscriptions),
            "rate_limited" => Ok(SubscriptionFailureReason::RateLimited),
            "internal_error" => Ok(SubscriptionFailureReason::InternalError),
            _ => Err(format!("无效的订阅失败原因: {}", s)),
        }
    }
}

// ============================================================
// 26. 信息流来源枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FeedSource {
    Following, // 关注
    Nearby,    // 附近
    All,       // 综合
}

impl FeedSource {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedSource::Following => "following",
            FeedSource::Nearby => "nearby",
            FeedSource::All => "all",
        }
    }
}

impl std::fmt::Display for FeedSource {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for FeedSource {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "following" => Ok(FeedSource::Following),
            "nearby" => Ok(FeedSource::Nearby),
            "all" => Ok(FeedSource::All),
            _ => Err(format!("无效的信息流来源: {}", s)),
        }
    }
}

// ============================================================
// 27. 趋势枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Trend {
    Rising,  // 上升
    Stable,  // 稳定
    Falling, // 下降
}

impl Trend {
    pub fn as_str(&self) -> &'static str {
        match self {
            Trend::Rising => "rising",
            Trend::Stable => "stable",
            Trend::Falling => "falling",
        }
    }
}

impl std::fmt::Display for Trend {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for Trend {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "rising" => Ok(Trend::Rising),
            "stable" => Ok(Trend::Stable),
            "falling" => Ok(Trend::Falling),
            _ => Err(format!("无效的趋势: {}", s)),
        }
    }
}

// ============================================================
// DTO 特有枚举（不存储在数据库 ENUM 类型，但可能有 CHECK 约束）
// ============================================================

// 文件分类（对应 uploaded_files.category CHECK 约束）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "varchar")]
#[serde(rename_all = "snake_case")]
pub enum FileCategory {
    Avatar,
    GroupAvatar,
    MomentImage,
    ChatImage,
    ChatVideo,
    ChatAudio,
    ChatFile,
}

impl FileCategory {
    pub fn as_str(&self) -> &'static str {
        match self {
            FileCategory::Avatar => "avatar",
            FileCategory::GroupAvatar => "group_avatar",
            FileCategory::MomentImage => "moment_image",
            FileCategory::ChatImage => "chat_image",
            FileCategory::ChatVideo => "chat_video",
            FileCategory::ChatAudio => "chat_audio",
            FileCategory::ChatFile => "chat_file",
        }
    }
}

impl std::fmt::Display for FileCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for FileCategory {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "avatar" => Ok(FileCategory::Avatar),
            "group_avatar" => Ok(FileCategory::GroupAvatar),
            "moment_image" => Ok(FileCategory::MomentImage),
            "chat_image" => Ok(FileCategory::ChatImage),
            "chat_video" => Ok(FileCategory::ChatVideo),
            "chat_audio" => Ok(FileCategory::ChatAudio),
            "chat_file" => Ok(FileCategory::ChatFile),
            _ => Err(format!("无效的文件分类: {}", s)),
        }
    }
}

// 收藏操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum FavoriteAction {
    Add,
    Remove,
}

// 点赞操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LikeAction {
    Like,
    Unlike,
}

// 举报处理操作（DTO层的，与数据库的 report_status 不同）
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum ReportAction {
    Resolve,
    Dismiss,
    Escalate,
}

impl ReportAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReportAction::Resolve => "resolve",
            ReportAction::Dismiss => "dismiss",
            ReportAction::Escalate => "escalate",
        }
    }
}

impl std::fmt::Display for ReportAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReportAction {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "resolve" => Ok(ReportAction::Resolve),
            "dismiss" => Ok(ReportAction::Dismiss),
            "escalate" => Ok(ReportAction::Escalate),
            _ => Err(format!("无效的操作类型: {}", s)),
        }
    }
}

// 系统健康检查状态
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum HealthStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

impl HealthStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            HealthStatus::Healthy => "healthy",
            HealthStatus::Degraded => "degraded",
            HealthStatus::Unhealthy => "unhealthy",
        }
    }
}

impl std::fmt::Display for HealthStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// 配置值类型
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum ConfigValueType {
    String,
    Number,
    Boolean,
}

impl ConfigValueType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ConfigValueType::String => "string",
            ConfigValueType::Number => "number",
            ConfigValueType::Boolean => "boolean",
        }
    }
}

impl std::fmt::Display for ConfigValueType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// 通知级别
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum NotificationSeverity {
    Info,
    Warning,
    Error,
}

impl NotificationSeverity {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationSeverity::Info => "info",
            NotificationSeverity::Warning => "warning",
            NotificationSeverity::Error => "error",
        }
    }
}

impl std::fmt::Display for NotificationSeverity {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for NotificationSeverity {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "info" => Ok(NotificationSeverity::Info),
            "warning" => Ok(NotificationSeverity::Warning),
            "error" => Ok(NotificationSeverity::Error),
            _ => Err(format!("无效的通知级别: {}", s)),
        }
    }
}
