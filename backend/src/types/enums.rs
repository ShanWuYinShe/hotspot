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

    /// 检查是否为管理员
    pub fn is_admin(&self) -> bool {
        matches!(self, UserType::Admin)
    }
}

impl std::fmt::Display for UserType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 12. 举报处理操作枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ReportHandleAction {
    Resolve,  // 解决
    Dismiss,  // 驳回
    Escalate, // 升级
}

impl ReportHandleAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            ReportHandleAction::Resolve => "resolve",
            ReportHandleAction::Dismiss => "dismiss",
            ReportHandleAction::Escalate => "escalate",
        }
    }
}

impl std::fmt::Display for ReportHandleAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReportHandleAction {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "resolve" => Ok(ReportHandleAction::Resolve),
            "dismiss" => Ok(ReportHandleAction::Dismiss),
            "escalate" => Ok(ReportHandleAction::Escalate),
            _ => Err(format!("无效的操作类型: {}", s)),
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
    Report,        // 举报 (用于权重调整)
}

impl FeedFeedbackAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedFeedbackAction::Hide => "hide",
            FeedFeedbackAction::NotInterested => "not_interested",
            FeedFeedbackAction::Clicked => "clicked",
            FeedFeedbackAction::Viewed => "viewed",
            FeedFeedbackAction::Report => "report",
        }
    }

    /// 判断是否应从 feed 中移除
    pub fn should_remove_from_feed(&self) -> bool {
        matches!(
            self,
            FeedFeedbackAction::Hide
                | FeedFeedbackAction::NotInterested
                | FeedFeedbackAction::Report
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
            "report" => Ok(FeedFeedbackAction::Report),
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
