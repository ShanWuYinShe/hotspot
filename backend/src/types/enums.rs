//! 数据库枚举类型定义
//!
//! 这些枚举类型与 API 规范 (api_spec_common_types.mdc) 一一对应
//! 所有枚举命名和取值以 API 规范为准

use serde::{Deserialize, Serialize};
use sqlx::Type;

// ============================================================
// 用户相关枚举
// ============================================================

/// 用户类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "user_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum UserType {
    Temp,   // 临时用户
    Normal, // 普通用户
    Admin,  // 管理员用户
    System, // 系统用户
}

impl Default for UserType {
    fn default() -> Self {
        UserType::Normal
    }
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

/// 用户账号状态
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "user_account_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum UserAccountStatus {
    Normal, // 正常
    Banned, // 被封禁
}

impl UserAccountStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            UserAccountStatus::Normal => "normal",
            UserAccountStatus::Banned => "banned",
        }
    }
}

impl std::fmt::Display for UserAccountStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for UserAccountStatus {
    type Err = String;
    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "normal" => Ok(UserAccountStatus::Normal),
            "banned" => Ok(UserAccountStatus::Banned),
            _ => Err(format!("Unknown UserAccountStatus: {}", s)),
        }
    }
}

/// 性别
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "gender", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum Gender {
    Male,             // 男
    Female,           // 女
    ApacheHelicopter, // 武装直升机
}

impl Gender {
    pub fn as_str(&self) -> &'static str {
        match self {
            Gender::Male => "male",
            Gender::Female => "female",
            Gender::ApacheHelicopter => "apache_helicopter",
        }
    }
}

impl std::fmt::Display for Gender {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for Gender {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "male" => Ok(Gender::Male),
            "female" => Ok(Gender::Female),
            "apache_helicopter" => Ok(Gender::ApacheHelicopter),
            _ => Err(format!("无效的性别: {}", s)),
        }
    }
}

/// 关注来源
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FollowSource {
    Moment,         // 通过瞬间
    Search,         // 通过搜索
    Recommendation, // 通过推荐
    Profile,        // 通过个人主页
    Group,          // 通过群组
}

impl FollowSource {
    pub fn as_str(&self) -> &'static str {
        match self {
            FollowSource::Moment => "moment",
            FollowSource::Search => "search",
            FollowSource::Recommendation => "recommendation",
            FollowSource::Profile => "profile",
            FollowSource::Group => "group",
        }
    }
}

impl std::fmt::Display for FollowSource {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for FollowSource {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "moment" => Ok(FollowSource::Moment),
            "search" => Ok(FollowSource::Search),
            "recommendation" => Ok(FollowSource::Recommendation),
            "profile" => Ok(FollowSource::Profile),
            "group" => Ok(FollowSource::Group),
            _ => Err(format!("无效的关注来源: {}", s)),
        }
    }
}

// ============================================================
// 隐私与权限相关枚举
// ============================================================

/// 隐私级别
/// 注：'friends' 表示"好友可见"，好友 = 双方互相关注的用户
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

/// 私聊权限
/// 注：'friends' 表示"仅好友"，好友 = 双方互相关注的用户
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "chat_permission", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum PrivateChatPermission {
    Everyone, // 所有人
    Friends,  // 仅好友
    Nobody,   // 禁止
}

impl PrivateChatPermission {
    pub fn as_str(&self) -> &'static str {
        match self {
            PrivateChatPermission::Everyone => "everyone",
            PrivateChatPermission::Friends => "friends",
            PrivateChatPermission::Nobody => "nobody",
        }
    }
}

impl std::fmt::Display for PrivateChatPermission {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for PrivateChatPermission {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "everyone" => Ok(PrivateChatPermission::Everyone),
            "friends" => Ok(PrivateChatPermission::Friends),
            "nobody" => Ok(PrivateChatPermission::Nobody),
            _ => Err(format!("无效的私聊权限: {}", s)),
        }
    }
}

// ============================================================
// 群组相关枚举
// ============================================================

/// 群组角色
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "member_role", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum GroupRole {
    Owner,  // 群主
    Admin,  // 管理员
    Member, // 普通成员
}

impl GroupRole {
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupRole::Owner => "owner",
            GroupRole::Admin => "admin",
            GroupRole::Member => "member",
        }
    }

    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "owner" => Ok(GroupRole::Owner),
            "admin" => Ok(GroupRole::Admin),
            "member" => Ok(GroupRole::Member),
            _ => Err(format!("无效的群组角色: {}", s)),
        }
    }
}

impl std::fmt::Display for GroupRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 可设置的群组角色（不含 owner）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SetableGroupRole {
    Admin,  // 管理员
    Member, // 普通成员
}

impl SetableGroupRole {
    pub fn as_str(&self) -> &'static str {
        match self {
            SetableGroupRole::Admin => "admin",
            SetableGroupRole::Member => "member",
        }
    }

    pub fn to_group_role(&self) -> GroupRole {
        match self {
            SetableGroupRole::Admin => GroupRole::Admin,
            SetableGroupRole::Member => GroupRole::Member,
        }
    }
}

impl std::fmt::Display for SetableGroupRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for SetableGroupRole {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "admin" => Ok(SetableGroupRole::Admin),
            "member" => Ok(SetableGroupRole::Member),
            _ => Err(format!("无效的可设置群组角色: {}", s)),
        }
    }
}

/// 群组加入方式
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "join_method", rename_all = "snake_case")]
#[serde(rename_all = "snake_case")]
pub enum GroupJoinMethod {
    Open,       // 开放加入
    Approval,   // 需要审批
    InviteOnly, // 仅邀请
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

impl std::fmt::Display for GroupJoinMethod {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for GroupJoinMethod {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "open" => Ok(GroupJoinMethod::Open),
            "approval" => Ok(GroupJoinMethod::Approval),
            "invite_only" => Ok(GroupJoinMethod::InviteOnly),
            _ => Err(format!("无效的群组加入方式: {}", s)),
        }
    }
}

/// 成员离开原因
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MemberLeaveReason {
    Left,   // 主动退出
    Kicked, // 被移除
    Banned, // 被封禁
}

impl MemberLeaveReason {
    pub fn as_str(&self) -> &'static str {
        match self {
            MemberLeaveReason::Left => "left",
            MemberLeaveReason::Kicked => "kicked",
            MemberLeaveReason::Banned => "banned",
        }
    }
}

impl std::fmt::Display for MemberLeaveReason {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 消息相关枚举
// ============================================================

/// 消息类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "message_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum MessageType {
    Text,   // 文本
    Image,  // 图片
    Video,  // 视频
    Audio,  // 音频
    File,   // 文件
    System, // 系统消息
}

impl MessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MessageType::Text => "text",
            MessageType::Image => "image",
            MessageType::Video => "video",
            MessageType::Audio => "audio",
            MessageType::File => "file",
            MessageType::System => "system",
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

/// 可发送的消息类型（不含 System）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SendMessageType {
    Text,
    Image,
    Video,
    Audio,
    File,
}

impl SendMessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            SendMessageType::Text => "text",
            SendMessageType::Image => "image",
            SendMessageType::Video => "video",
            SendMessageType::Audio => "audio",
            SendMessageType::File => "file",
        }
    }

    pub fn to_message_type(&self) -> MessageType {
        match self {
            SendMessageType::Text => MessageType::Text,
            SendMessageType::Image => MessageType::Image,
            SendMessageType::Video => MessageType::Video,
            SendMessageType::Audio => MessageType::Audio,
            SendMessageType::File => MessageType::File,
        }
    }
}

impl std::fmt::Display for SendMessageType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for SendMessageType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "text" => Ok(SendMessageType::Text),
            "image" => Ok(SendMessageType::Image),
            "video" => Ok(SendMessageType::Video),
            "audio" => Ok(SendMessageType::Audio),
            "file" => Ok(SendMessageType::File),
            _ => Err(format!("无效的可发送消息类型: {}", s)),
        }
    }
}

/// 反应类型
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
// 举报相关枚举
// ============================================================

/// 举报状态
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "report_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ReportStatus {
    Pending,   // 待处理
    Reviewing, // 审核中
    Resolved,  // 已解决
    Dismissed, // 已驳回
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

impl std::fmt::Display for ReportStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ReportStatus {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "pending" => Ok(ReportStatus::Pending),
            "reviewing" => Ok(ReportStatus::Reviewing),
            "resolved" => Ok(ReportStatus::Resolved),
            "dismissed" => Ok(ReportStatus::Dismissed),
            _ => Err(format!("无效的举报状态: {}", s)),
        }
    }
}

impl std::convert::From<String> for ReportStatus {
    fn from(s: String) -> Self {
        s.parse().unwrap_or(ReportStatus::Pending)
    }
}

/// 举报目标类型
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

impl std::convert::From<String> for ReportTargetType {
    fn from(s: String) -> Self {
        s.parse().unwrap_or(ReportTargetType::Moment)
    }
}

/// 举报处理操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum ReportHandleAction {
    Resolve,  // 确认违规并处理
    Dismiss,  // 驳回举报
    Escalate, // 升级处理
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
            _ => Err(format!("无效的举报处理操作: {}", s)),
        }
    }
}

// ============================================================
// 申请/邀请状态枚举
// ============================================================

/// 申请状态（群组申请等）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "application_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ApplicationStatus {
    Pending,  // 待处理
    Accepted, // 已接受
    Rejected, // 已拒绝
}

impl ApplicationStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ApplicationStatus::Pending => "pending",
            ApplicationStatus::Accepted => "accepted",
            ApplicationStatus::Rejected => "rejected",
        }
    }
}

impl std::fmt::Display for ApplicationStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ApplicationStatus {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "pending" => Ok(ApplicationStatus::Pending),
            "accepted" => Ok(ApplicationStatus::Accepted),
            "rejected" => Ok(ApplicationStatus::Rejected),
            _ => Err(format!("无效的申请状态: {}", s)),
        }
    }
}

/// 响应状态（用于邀请/申请响应）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ResponseStatus {
    Accepted, // 已接受
    Rejected, // 已拒绝
}

impl ResponseStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ResponseStatus::Accepted => "accepted",
            ResponseStatus::Rejected => "rejected",
        }
    }
}

impl std::fmt::Display for ResponseStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 关注请求处理动作
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FollowRequestAction {
    Accept,
    Reject,
}

impl FollowRequestAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            FollowRequestAction::Accept => "accept",
            FollowRequestAction::Reject => "reject",
        }
    }
}

impl std::fmt::Display for FollowRequestAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 认证相关枚举
// ============================================================

/// 客户端类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ClientType {
    Web, // Web 客户端
    App, // App 客户端
}

impl ClientType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ClientType::Web => "web",
            ClientType::App => "app",
        }
    }
}

impl std::fmt::Display for ClientType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ClientType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "web" => Ok(ClientType::Web),
            "app" => Ok(ClientType::App),
            _ => Err(format!("无效的客户端类型: {}", s)),
        }
    }
}

/// 验证码用途类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum VerificationCodeType {
    Register,      // 注册
    ResetPassword, // 重置密码
    ChangePhone,   // 更换手机号
}

impl VerificationCodeType {
    pub fn as_str(&self) -> &'static str {
        match self {
            VerificationCodeType::Register => "register",
            VerificationCodeType::ResetPassword => "reset_password",
            VerificationCodeType::ChangePhone => "change_phone",
        }
    }
}

impl std::fmt::Display for VerificationCodeType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for VerificationCodeType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "register" => Ok(VerificationCodeType::Register),
            "reset_password" => Ok(VerificationCodeType::ResetPassword),
            "change_phone" => Ok(VerificationCodeType::ChangePhone),
            _ => Err(format!("无效的验证码类型: {}", s)),
        }
    }
}

// ============================================================
// 设备与平台枚举
// ============================================================

/// 设备平台
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "platform", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum DevicePlatform {
    Ios,     // iOS
    Android, // Android
    Harmony, // HarmonyOS
    Web,     // Web
}

impl DevicePlatform {
    pub fn as_str(&self) -> &'static str {
        match self {
            DevicePlatform::Ios => "ios",
            DevicePlatform::Android => "android",
            DevicePlatform::Harmony => "harmony",
            DevicePlatform::Web => "web",
        }
    }
}

impl std::fmt::Display for DevicePlatform {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for DevicePlatform {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "ios" => Ok(DevicePlatform::Ios),
            "android" => Ok(DevicePlatform::Android),
            "harmony" => Ok(DevicePlatform::Harmony),
            "web" => Ok(DevicePlatform::Web),
            _ => Err(format!("无效的设备平台: {}", s)),
        }
    }
}

// ============================================================
// 媒体上传枚举
// ============================================================

/// 媒体分类
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "varchar")]
#[serde(rename_all = "snake_case")]
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
    pub fn as_str(&self) -> &'static str {
        match self {
            MediaCategory::Avatar => "avatar",
            MediaCategory::GroupAvatar => "group_avatar",
            MediaCategory::MomentImage => "moment_image",
            MediaCategory::ChatImage => "chat_image",
            MediaCategory::ChatVideo => "chat_video",
            MediaCategory::ChatAudio => "chat_audio",
            MediaCategory::ChatFile => "chat_file",
        }
    }

    /// 返回该分类允许的 MIME 类型列表
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
            MediaCategory::ChatFile => &[], // 任意类型
        }
    }

    /// 返回该分类的最大文件大小（字节）
    pub fn max_file_size(&self) -> i64 {
        match self {
            MediaCategory::Avatar | MediaCategory::GroupAvatar => 5 * 1024 * 1024, // 5MB
            MediaCategory::MomentImage | MediaCategory::ChatImage => 10 * 1024 * 1024, // 10MB
            MediaCategory::ChatAudio => 20 * 1024 * 1024,                          // 20MB
            MediaCategory::ChatVideo | MediaCategory::ChatFile => 100 * 1024 * 1024, // 100MB
        }
    }

    /// 验证 MIME 类型是否符合分类要求
    pub fn validate_mime_type(&self, mime_type: &str) -> bool {
        let allowed = self.allowed_mime_types();
        if allowed.is_empty() {
            // chat_file 允许任意类型
            return true;
        }
        allowed.contains(&mime_type)
    }

    /// 验证文件大小是否符合分类要求
    pub fn validate_file_size(&self, size: i64) -> bool {
        size <= self.max_file_size()
    }
}

impl std::fmt::Display for MediaCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for MediaCategory {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "avatar" => Ok(MediaCategory::Avatar),
            "group_avatar" => Ok(MediaCategory::GroupAvatar),
            "moment_image" => Ok(MediaCategory::MomentImage),
            "chat_image" => Ok(MediaCategory::ChatImage),
            "chat_video" => Ok(MediaCategory::ChatVideo),
            "chat_audio" => Ok(MediaCategory::ChatAudio),
            "chat_file" => Ok(MediaCategory::ChatFile),
            _ => Err(format!("无效的媒体分类: {}", s)),
        }
    }
}

// ============================================================
// 信息流与推荐枚举
// ============================================================

/// 信息流来源
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
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

/// 信息流内容类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FeedContentType {
    Moment, // 瞬间
    Group,  // 群组
    User,   // 用户
}

impl FeedContentType {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedContentType::Moment => "moment",
            FeedContentType::Group => "group",
            FeedContentType::User => "user",
        }
    }
}

impl std::fmt::Display for FeedContentType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for FeedContentType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "moment" => Ok(FeedContentType::Moment),
            "group" => Ok(FeedContentType::Group),
            "user" => Ok(FeedContentType::User),
            _ => Err(format!("无效的信息流内容类型: {}", s)),
        }
    }
}

/// 信息流反馈操作
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FeedFeedbackAction {
    Hide,          // 隐藏
    NotInterested, // 不感兴趣
}

impl FeedFeedbackAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            FeedFeedbackAction::Hide => "hide",
            FeedFeedbackAction::NotInterested => "not_interested",
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
            _ => Err(format!("无效的反馈操作: {}", s)),
        }
    }
}

// ============================================================
// 标签枚举
// ============================================================

/// 标签趋势
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TagTrend {
    Rising,  // 上升
    Stable,  // 稳定
    Falling, // 下降
}

impl TagTrend {
    pub fn as_str(&self) -> &'static str {
        match self {
            TagTrend::Rising => "rising",
            TagTrend::Stable => "stable",
            TagTrend::Falling => "falling",
        }
    }
}

impl std::fmt::Display for TagTrend {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for TagTrend {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "rising" => Ok(TagTrend::Rising),
            "stable" => Ok(TagTrend::Stable),
            "falling" => Ok(TagTrend::Falling),
            _ => Err(format!("无效的标签趋势: {}", s)),
        }
    }
}

/// 标签时间范围
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TagTimeframe {
    Daily,   // 每天
    Weekly,  // 每周
    Monthly, // 每月
}

impl TagTimeframe {
    pub fn as_str(&self) -> &'static str {
        match self {
            TagTimeframe::Daily => "daily",
            TagTimeframe::Weekly => "weekly",
            TagTimeframe::Monthly => "monthly",
        }
    }
}

impl std::fmt::Display for TagTimeframe {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for TagTimeframe {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "daily" => Ok(TagTimeframe::Daily),
            "weekly" => Ok(TagTimeframe::Weekly),
            "monthly" => Ok(TagTimeframe::Monthly),
            _ => Err(format!("无效的标签时间范围: {}", s)),
        }
    }
}

// ============================================================
// 通知枚举
// ============================================================

/// 通知过滤类型（用于 API 筛选）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum NotificationFilterType {
    Like,
    Comment,
    Follow,
    System,
    Mention,
}

impl NotificationFilterType {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationFilterType::Like => "like",
            NotificationFilterType::Comment => "comment",
            NotificationFilterType::Follow => "follow",
            NotificationFilterType::System => "system",
            NotificationFilterType::Mention => "mention",
        }
    }
}

impl std::fmt::Display for NotificationFilterType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for NotificationFilterType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "like" => Ok(NotificationFilterType::Like),
            "comment" => Ok(NotificationFilterType::Comment),
            "follow" => Ok(NotificationFilterType::Follow),
            "system" => Ok(NotificationFilterType::System),
            "mention" => Ok(NotificationFilterType::Mention),
            _ => Err(format!("无效的通知过滤类型: {}", s)),
        }
    }
}

/// 通知类型（用于具体通知项）
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum NotificationType {
    MomentLike,
    MomentComment,
    CommentReply,
    NewFollower,
    MutualFollow,
    FollowRequest,
    FollowRequestHandled,
    GroupInvite,
    GroupInviteResponse,
    GroupApplication,
    GroupApplicationResponse,
    GroupAnnouncement,
    SystemAnnouncement,
    MessageMention,
}

impl NotificationType {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationType::MomentLike => "moment_like",
            NotificationType::MomentComment => "moment_comment",
            NotificationType::CommentReply => "comment_reply",
            NotificationType::NewFollower => "new_follower",
            NotificationType::MutualFollow => "mutual_follow",
            NotificationType::FollowRequest => "follow_request",
            NotificationType::FollowRequestHandled => "follow_request_handled",
            NotificationType::GroupInvite => "group_invite",
            NotificationType::GroupInviteResponse => "group_invite_response",
            NotificationType::GroupApplication => "group_application",
            NotificationType::GroupApplicationResponse => "group_application_response",
            NotificationType::GroupAnnouncement => "group_announcement",
            NotificationType::SystemAnnouncement => "system_announcement",
            NotificationType::MessageMention => "message_mention",
        }
    }
}

impl std::fmt::Display for NotificationType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for NotificationType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "moment_like" => Ok(NotificationType::MomentLike),
            "moment_comment" => Ok(NotificationType::MomentComment),
            "comment_reply" => Ok(NotificationType::CommentReply),
            "new_follower" => Ok(NotificationType::NewFollower),
            "mutual_follow" => Ok(NotificationType::MutualFollow),
            "follow_request" => Ok(NotificationType::FollowRequest),
            "follow_request_handled" => Ok(NotificationType::FollowRequestHandled),
            "group_invite" => Ok(NotificationType::GroupInvite),
            "group_invite_response" => Ok(NotificationType::GroupInviteResponse),
            "group_application" => Ok(NotificationType::GroupApplication),
            "group_application_response" => Ok(NotificationType::GroupApplicationResponse),
            "group_announcement" => Ok(NotificationType::GroupAnnouncement),
            "system_announcement" => Ok(NotificationType::SystemAnnouncement),
            "message_mention" => Ok(NotificationType::MessageMention),
            _ => Err(format!("无效的通知类型: {}", s)),
        }
    }
}

/// 通知目标类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum NotificationTargetType {
    Moment,
    Comment,
    User,
    Group,
    Web,
}

impl NotificationTargetType {
    pub fn as_str(&self) -> &'static str {
        match self {
            NotificationTargetType::Moment => "moment",
            NotificationTargetType::Comment => "comment",
            NotificationTargetType::User => "user",
            NotificationTargetType::Group => "group",
            NotificationTargetType::Web => "web",
        }
    }
}

impl std::fmt::Display for NotificationTargetType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for NotificationTargetType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "moment" => Ok(NotificationTargetType::Moment),
            "comment" => Ok(NotificationTargetType::Comment),
            "user" => Ok(NotificationTargetType::User),
            "group" => Ok(NotificationTargetType::Group),
            "web" => Ok(NotificationTargetType::Web),
            _ => Err(format!("无效的通知目标类型: {}", s)),
        }
    }
}

// ============================================================
// 地图枚举
// ============================================================

/// 地图数据类型过滤
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
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

impl std::fmt::Display for MapDataType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for MapDataType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "user" => Ok(MapDataType::User),
            "group" => Ok(MapDataType::Group),
            "moment" => Ok(MapDataType::Moment),
            "all" => Ok(MapDataType::All),
            _ => Err(format!("无效的地图数据类型: {}", s)),
        }
    }
}

/// 地图渲染项类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MapItemType {
    Cluster, // 聚合
    User,    // 用户
    Group,   // 群组
    Moment,  // 瞬间
}

impl MapItemType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MapItemType::Cluster => "cluster",
            MapItemType::User => "user",
            MapItemType::Group => "group",
            MapItemType::Moment => "moment",
        }
    }
}

impl std::fmt::Display for MapItemType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 系统与配置枚举
// ============================================================

/// 系统健康状态
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum SystemHealthStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

impl SystemHealthStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            SystemHealthStatus::Healthy => "healthy",
            SystemHealthStatus::Degraded => "degraded",
            SystemHealthStatus::Unhealthy => "unhealthy",
        }
    }
}

impl std::fmt::Display for SystemHealthStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 公告严重程度
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
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

impl std::fmt::Display for AnnouncementSeverity {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for AnnouncementSeverity {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "info" => Ok(AnnouncementSeverity::Info),
            "warning" => Ok(AnnouncementSeverity::Warning),
            "error" => Ok(AnnouncementSeverity::Error),
            _ => Err(format!("无效的公告严重程度: {}", s)),
        }
    }
}

// ============================================================
// WebSocket 订阅枚举
// ============================================================

/// 订阅操作类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SubscriptionOperation {
    Subscribe,   // 增量订阅
    Unsubscribe, // 取消订阅
    Set,         // 替换式订阅
}

impl SubscriptionOperation {
    pub fn as_str(&self) -> &'static str {
        match self {
            SubscriptionOperation::Subscribe => "subscribe",
            SubscriptionOperation::Unsubscribe => "unsubscribe",
            SubscriptionOperation::Set => "set",
        }
    }
}

impl std::fmt::Display for SubscriptionOperation {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 订阅失败原因
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

// ============================================================
// DTO 操作类型枚举（内部使用）
// ============================================================

/// 点赞操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LikeAction {
    Like,
    Unlike,
}

impl LikeAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            LikeAction::Like => "like",
            LikeAction::Unlike => "unlike",
        }
    }
}

impl std::fmt::Display for LikeAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 收藏操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum FavoriteAction {
    Add,
    Remove,
}

impl FavoriteAction {
    pub fn as_str(&self) -> &'static str {
        match self {
            FavoriteAction::Add => "add",
            FavoriteAction::Remove => "remove",
        }
    }
}

impl std::fmt::Display for FavoriteAction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 配置值类型
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Type)]
#[sqlx(type_name = "config_value_type", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ConfigValueType {
    String,
    Number,
    Boolean,
    Json,
}

impl ConfigValueType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ConfigValueType::String => "string",
            ConfigValueType::Number => "number",
            ConfigValueType::Boolean => "boolean",
            ConfigValueType::Json => "json",
        }
    }
}

impl std::fmt::Display for ConfigValueType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// 上传状态
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Type)]
#[sqlx(type_name = "upload_status", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum UploadStatus {
    Pending,
    Uploading,
    Completed,
    Failed,
    Cancelled,
}

impl UploadStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            UploadStatus::Pending => "pending",
            UploadStatus::Uploading => "uploading",
            UploadStatus::Completed => "completed",
            UploadStatus::Failed => "failed",
            UploadStatus::Cancelled => "cancelled",
        }
    }
}

impl std::fmt::Display for UploadStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

// ============================================================
// 查询过滤专用枚举（内部使用，仅用于 Repository 层过滤）
// ============================================================

/// 申请状态过滤（含 Any 变体，仅用于查询）
/// API 规范中使用 'pending' | 'accepted' | 'rejected' | 'all' 字符串
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ApplicationStatusFilter {
    Any,
    Specific(ApplicationStatus),
}

impl Default for ApplicationStatusFilter {
    fn default() -> Self {
        ApplicationStatusFilter::Any
    }
}

impl<'de> serde::Deserialize<'de> for ApplicationStatusFilter {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        match s.to_lowercase().as_str() {
            "all" => Ok(ApplicationStatusFilter::Any),
            "pending" => Ok(ApplicationStatusFilter::Specific(
                ApplicationStatus::Pending,
            )),
            "accepted" => Ok(ApplicationStatusFilter::Specific(
                ApplicationStatus::Accepted,
            )),
            "rejected" => Ok(ApplicationStatusFilter::Specific(
                ApplicationStatus::Rejected,
            )),
            _ => Err(serde::de::Error::custom(format!(
                "无效的申请状态过滤: {}",
                s
            ))),
        }
    }
}

impl serde::Serialize for ApplicationStatusFilter {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        match self {
            ApplicationStatusFilter::Any => serializer.serialize_str("all"),
            ApplicationStatusFilter::Specific(s) => serializer.serialize_str(s.as_str()),
        }
    }
}

// ============================================================
// 推荐系统枚举
// ============================================================

/// 推荐理由类型 - API 文档: RecommendationType
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RecommendationType {
    FriendLiked, // 好友点赞
    Nearby,      // 附近热门
    Trending,    // 热门趋势
    Following,   // 关注的人
    TagMatch,    // 标签匹配
}

impl RecommendationType {
    pub fn as_str(&self) -> &'static str {
        match self {
            RecommendationType::FriendLiked => "friend_liked",
            RecommendationType::Nearby => "nearby",
            RecommendationType::Trending => "trending",
            RecommendationType::Following => "following",
            RecommendationType::TagMatch => "tag_match",
        }
    }
}

// ============================================================
// 位置分享相关枚举
// ============================================================

/// 分享类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "VARCHAR", rename_all = "lowercase")]
#[serde(rename_all = "lowercase")]
pub enum ShareType {
    Location, // 位置分享
    Moment,   // 动态分享
    Group,    // 群组分享
    User,     // 用户分享
}

impl Default for ShareType {
    fn default() -> Self {
        ShareType::Location
    }
}

impl ShareType {
    pub fn as_str(&self) -> &'static str {
        match self {
            ShareType::Location => "location",
            ShareType::Moment => "moment",
            ShareType::Group => "group",
            ShareType::User => "user",
        }
    }
}

impl std::fmt::Display for ShareType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl std::str::FromStr for ShareType {
    type Err = String;
    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "location" => Ok(ShareType::Location),
            "moment" => Ok(ShareType::Moment),
            "group" => Ok(ShareType::Group),
            "user" => Ok(ShareType::User),
            _ => Err(format!("Unknown ShareType: {}", s)),
        }
    }
}

// ============================================================
