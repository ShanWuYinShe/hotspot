//! 数据库枚举类型定义
//! 
//! 这些枚举类型与 PostgreSQL ENUM 类型一一对应
//! 部分枚举暂未使用，但为未来功能预留

#![allow(dead_code)]

use serde::{Deserialize, Serialize};
use sqlx::Type;

// ============================================================
// 1. 成员角色枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "member_role", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum MemberRole {
    Owner,      // 群主
    Admin,      // 管理员
    Member,     // 普通成员
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
            _ => Err(format!("无效的成员角色: {}", s))
        }
    }
}

impl std::fmt::Display for MemberRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 2. 房间类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "room_type", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum RoomType {
    Direct,         // 私聊（与数据库一致）
    PublicGroup,    // 公开群组
    PrivateGroup,   // 私密群组
}

impl RoomType {
    pub fn as_str(&self) -> &'static str {
        match self {
            RoomType::Direct => "direct",
            RoomType::PublicGroup => "public_group",
            RoomType::PrivateGroup => "private_group",
        }
    }

    pub fn is_group(&self) -> bool {
        matches!(self, RoomType::PublicGroup | RoomType::PrivateGroup)
    }
    
    pub fn is_direct(&self) -> bool {
        matches!(self, RoomType::Direct)
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "direct" => Ok(RoomType::Direct),
            "public_group" => Ok(RoomType::PublicGroup),
            "private_group" => Ok(RoomType::PrivateGroup),
            _ => Err(format!("无效的群组类型: {}", s))
        }
    }
}

impl std::fmt::Display for RoomType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 3. 关系类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "relationship_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum RelationshipType {
    Any,         // 🔍 匹配所有值（仅用于查询过滤）
    Friend,      // 好友
    Following,   // 关注
    Blocked,     // 拉黑
}

impl RelationshipType {
    pub fn as_str(&self) -> &'static str {
        match self {
            RelationshipType::Any => "any",
            RelationshipType::Friend => "friend",
            RelationshipType::Following => "following",
            RelationshipType::Blocked => "blocked",
        }
    }
}

impl std::fmt::Display for RelationshipType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 4. 关系状态枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "relationship_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum RelationshipStatus {
    Any,        // 🔍 匹配所有值（仅用于查询过滤）
    Pending,    // 待处理
    Accepted,   // 已接受
    Rejected,   // 已拒绝
}

impl RelationshipStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            RelationshipStatus::Any => "any",
            RelationshipStatus::Pending => "pending",
            RelationshipStatus::Accepted => "accepted",
            RelationshipStatus::Rejected => "rejected",
        }
    }
}

impl std::fmt::Display for RelationshipStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 5. 消息类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "message_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum MessageType {
    Any,        // 🔍 匹配所有值（仅用于查询过滤）
    Text,       // 文本
    Image,      // 图片
    Video,      // 视频
    Audio,      // 音频
    File,       // 文件
    Location,   // 位置
    System,     // 系统消息
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
    Any,        // 🔍 匹配所有值（仅用于查询过滤）
    Pending,    // 待处理
    Reviewing,  // 审核中
    Resolved,   // 已解决
    Dismissed,  // 已驳回
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

// ============================================================
// 7. 邀请状态枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "invite_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum InviteStatus {
    Any,        // 🔍 匹配所有值（仅用于查询过滤）
    Pending,    // 待处理
    Accepted,   // 已接受
    Rejected,   // 已拒绝
    Approved,   // 已批准（用于申请）
    Expired,    // 已过期
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
// 8. 收件箱分类枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "inbox_category", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum InboxCategory {
    Any,        // 🔍 匹配所有值（仅用于查询过滤）
    Social,     // 社交互动
    Content,    // 内容互动
    System,     // 系统通知
}

impl InboxCategory {
    pub fn as_str(&self) -> &'static str {
        match self {
            InboxCategory::Any => "any",
            InboxCategory::Social => "social",
            InboxCategory::Content => "content",
            InboxCategory::System => "system",
        }
    }
}

impl std::fmt::Display for InboxCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 9. 举报目标类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "report_target_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReportTargetType {
    User,       // 用户
    Moment,     // 生活瞬间
    Comment,    // 评论
    Message,    // 消息
    Group,      // 群组
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

// ============================================================
// 10. 通知类别枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "notification_category", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum NotificationCategory {
    Social,     // 社交互动
    System,     // 系统通知
    Content,    // 内容互动
}

impl NotificationCategory {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationCategory::Social => "social",
            NotificationCategory::System => "system",
            NotificationCategory::Content => "content",
        }
    }
}

impl std::fmt::Display for NotificationCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 11. 通知严重程度枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "notification_severity", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum NotificationSeverity {
    Info,       // 信息
    Warning,    // 警告
    Error,      // 错误
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

// ============================================================
// 12. 平台枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "platform", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum Platform {
    Ios,        // iOS
    Android,    // Android
    Web,        // Web
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
// 13. 群组加入方式枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "join_method", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum JoinMethod {
    Open,           // 开放加入
    Approval,       // 需要审批
    InviteOnly,     // 仅限邀请
}

impl JoinMethod {
    pub fn as_str(&self) -> &'static str {
        match self {
            JoinMethod::Open => "open",
            JoinMethod::Approval => "approval",
            JoinMethod::InviteOnly => "invite_only",
        }
    }
}

impl std::fmt::Display for JoinMethod {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 14. 隐私级别枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "privacy_level", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum PrivacyLevel {
    Public,     // 公开
    Friends,    // 仅好友
    Private,    // 私密
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

// ============================================================
// 15. 反应类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "reaction_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReactionType {
    Like,       // 喜欢
    Love,       // 爱心
    Laugh,      // 笑脸
    Angry,      // 生气
    Sad,        // 悲伤
    Wow,        // 惊讶
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

// ============================================================
// 16. 信息流频道枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FeedChannel {
    Following,      // 关注
    Nearby,         // 附近
    Recommended,    // 推荐
}

impl FeedChannel {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedChannel::Following => "following",
            FeedChannel::Nearby => "nearby",
            FeedChannel::Recommended => "recommended",
        }
    }
}

// ============================================================
// 17. 信息流项目类型枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FeedItemType {
    Moment,         // 生活瞬间
    GroupCard,      // 群组卡片
    System,         // 系统消息
}

impl FeedItemType {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedItemType::Moment => "moment",
            FeedItemType::GroupCard => "group_card",
            FeedItemType::System => "system",
        }
    }
}

// ============================================================
// 18. 排行榜时间范围枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum LeaderboardTimeframe {
    Daily,      // 每日
    Weekly,     // 每周
    Monthly,    // 每月
    AllTime,    // 全部时间
}

impl LeaderboardTimeframe {
    pub fn as_str(&self) -> &'static str {
        match self {
            LeaderboardTimeframe::Daily => "daily",
            LeaderboardTimeframe::Weekly => "weekly",
            LeaderboardTimeframe::Monthly => "monthly",
            LeaderboardTimeframe::AllTime => "all_time",
        }
    }
}

// ============================================================
// 19. 排行榜范围枚举
// ============================================================
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum LeaderboardScope {
    Nearby,     // 附近
    Friends,    // 好友
}

impl LeaderboardScope {
    pub fn as_str(&self) -> &'static str {
        match self {
            LeaderboardScope::Nearby => "nearby",
            LeaderboardScope::Friends => "friends",
        }
    }
}

// ============================================================
// 测试
// ============================================================
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_member_role() {
        assert_eq!(MemberRole::Owner.as_str(), "owner");
        assert_eq!(MemberRole::Admin.as_str(), "admin");
        assert_eq!(MemberRole::Member.as_str(), "member");
    }

    #[test]
    fn test_room_type() {
        assert_eq!(RoomType::Direct.as_str(), "direct");
        assert_eq!(RoomType::PublicGroup.as_str(), "public_group");
        assert_eq!(RoomType::PrivateGroup.as_str(), "private_group");
        
        assert!(!RoomType::Direct.is_group());
        assert!(RoomType::Direct.is_direct());
        assert!(RoomType::PublicGroup.is_group());
        assert!(RoomType::PrivateGroup.is_group());
    }

    #[test]
    fn test_message_type() {
        assert_eq!(MessageType::Text.as_str(), "text");
        assert_eq!(MessageType::Image.as_str(), "image");
        
        assert!(!MessageType::Text.is_media());
        assert!(MessageType::Image.is_media());
        assert!(MessageType::Video.is_media());
    }
}

