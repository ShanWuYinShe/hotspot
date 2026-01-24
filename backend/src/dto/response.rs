//! Response DTO 定义
//!
//! 所有响应类型严格遵循 API 规范 (spec/*.mdc)
//! 使用强类型枚举，无向后兼容代码

use serde::{Deserialize, Serialize};
use serde_json::Value;

// 导入所有需要的枚举类型
use crate::types::enums::{
    AnnouncementSeverity, ApplicationStatus, DevicePlatform, Gender, GroupJoinMethod,
    GroupRole, MapItemType, MessageType, NotificationTargetType, NotificationType, PrivacyLevel,
    PrivateChatPermission, ReactionType, ReportStatus, ReportTargetType, SystemHealthStatus,
    TagTrend, UserAccountStatus, UserType,
};

// ============================================================
// 统一 API 响应格式
// ============================================================

/// 统一 API 响应格式 - API 文档: ApiResponse<T>
#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
    pub request_id: String,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: None,
            data: Some(data),
            request_id: String::new(),
        }
    }

    pub fn error(code: i32, message: String) -> Self {
        Self {
            code,
            message: Some(message),
            data: None,
            request_id: String::new(),
        }
    }
}

impl ApiResponse<()> {
    pub fn success_empty() -> Self {
        Self {
            code: 0,
            message: None,
            data: None,
            request_id: String::new(),
        }
    }
}

// ============================================================
// 通用基础类型 (API 文档定义)
// ============================================================

/// 位置信息 - API 文档: Location
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Location {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
}

/// 相对位置 - API 文档: RelativeLocation
#[derive(Debug, Serialize, Clone)]
pub struct RelativeLocation {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: f64,
    pub bearing_degrees: f64,
}

/// 消息限制 - API 文档: MessageLimit
#[derive(Debug, Serialize)]
pub struct MessageLimit {
    pub remaining: i32,
    pub total: i32,
    pub is_unlocked: bool,
}

/// 分页元数据 - API 文档: PaginationMeta
#[derive(Debug, Serialize)]
pub struct PaginationMeta {
    pub has_more: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

/// 分页数据容器 - API 文档: PaginatedData<T>
#[derive(Debug, Serialize)]
pub struct PaginatedData<T> {
    pub items: Vec<T>,
    pub pagination: PaginationMeta,
}

// ============================================================
// 用户相关类型
// ============================================================

/// 用户基础信息 - API 文档: UserInfo
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
}

/// 用户关系信息 - API 文档: Relationship
/// 注意：不返回"对方是否拉黑了我"（is_blocking），这是对方的隐私
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserRelation {
    pub is_mutual_follow: bool,
    pub is_following: bool,       // 我是否关注了对方
    pub is_followed_by: bool,     // 对方是否关注了我
    pub is_blocked: bool,         // 我是否拉黑了对方
    pub has_pending_follow_request: bool,
}

/// 关注请求项 - API 文档: FollowRequestItem
#[derive(Debug, Serialize, Clone)]
pub struct FollowRequestItem {
    pub request_id: String,
    pub requester: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub requested_at: String,
}

/// 用户搜索/列表项 - API 文档: UserSearchItem
#[derive(Debug, Serialize, Clone, Default)]
pub struct UserItem {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    // 隐私受控字段（隐私限制时不返回，不暴露原因）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relation: Option<UserRelation>,
}

/// 用户详情 - API 文档: UserCard
#[derive(Debug, Serialize)]
pub struct UserCard {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
    pub stats: UserStatsData,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relation: Option<UserRelation>,
}

/// 用户统计数据 - API 文档: UserStats
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserStatsData {
    pub moments_count: i32,
    pub friends_count: i32,
    pub followers_count: i32,
    pub following_count: i32,
    pub groups_count: i32,
}

/// 当前用户资料 - API 文档: MyProfileData
#[derive(Debug, Serialize)]
pub struct MyProfileData {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gender: Option<Gender>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    pub location: Location,
    pub is_online: bool,
    pub last_online: String,
    pub created_at: String,
    pub stats: UserStatsData,
}

/// 用户设置数据 - API 文档: UserSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct UserSettingsData {
    // 隐私设置
    pub privacy_level: PrivacyLevel,
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: PrivacyLevel,
    pub following_visible: PrivacyLevel,
    pub followers_visible: PrivacyLevel,
    pub groups_visible: PrivacyLevel,
    pub moments_visible: PrivacyLevel,
    pub liked_moments_visible: PrivacyLevel,
    pub stats_visible: PrivacyLevel,
    // 社交设置
    pub allow_follow: bool,
    pub allow_private_chat: PrivateChatPermission,
    // 搜索设置
    pub searchable: bool,
    pub phone_searchable: bool,
    pub nickname_searchable: bool,
    pub location_searchable: bool,
}

/// 好友列表项 - API 文档: FriendItem
#[derive(Debug, Serialize)]
pub struct FriendItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
    pub followed_at: String,
    pub followed_back_at: String,
}

/// 关注列表项 - API 文档: FollowingItem
#[derive(Debug, Serialize)]
pub struct FollowingItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_mutual: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
}

/// 粉丝列表项 - API 文档: FollowerItem
#[derive(Debug, Serialize)]
pub struct FollowerItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_following_back: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
}

/// 拉黑用户列表项 - API 文档: BlockedUserItem
#[derive(Debug, Serialize)]
pub struct BlockedUserItem {
    pub user: UserInfo,
    pub blocked_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ============================================================
// 认证相关类型
// ============================================================

/// 认证令牌数据 - API 文档: AuthTokenPairWeb/AuthTokenPairApp
#[derive(Debug, Serialize, Clone)]
pub struct AuthData {
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    pub access_token_expires_at: String,
    pub refresh_token_expires_at: String,
}

/// 认证状态检查数据 - API 文档: CheckStatusData
#[derive(Debug, Serialize)]
pub struct CheckStatusData {
    pub user_id: i64,
    pub nickname: String,
    pub is_valid: bool,
    pub expires_at: String,
}

/// 发送验证码数据 - API 文档: SendVerificationCodeData
#[derive(Debug, Serialize)]
pub struct SendVerificationCodeData {
    pub code_id: String,
    pub expires_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub retry_after: Option<i32>,
}

/// 验证码验证数据 - API 文档: VerifyCodeData
#[derive(Debug, Serialize)]
pub struct VerifyCodeData {
    pub verified: bool,
    pub verify_token: String,
    pub expires_at: String,
}

/// 密码重置请求数据 - API 文档: RequestPasswordResetData
#[derive(Debug, Serialize)]
pub struct RequestPasswordResetData {
    pub reset_id: String,
}

/// 重置码验证数据 - API 文档: VerifyResetCodeData
#[derive(Debug, Serialize)]
pub struct VerifyResetCodeData {
    pub reset_token: String,
}

/// 会话信息项 - API 文档: SessionItem
#[derive(Debug, Serialize)]
pub struct SessionItem {
    pub session_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    pub created_at: String,
    pub last_active_at: String,
    pub is_current: bool,
}

// ============================================================
// 瞬间相关类型
// ============================================================

/// 瞬间图片 - API 文档: MomentImage
#[derive(Debug, Serialize, Clone)]
pub struct MomentImage {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content_type: Option<String>,
}

/// 瞬间列表/搜索项 - API 文档: MomentSearchItem
/// 注意：不返回 privacy_level，隐私过滤在服务端完成
#[derive(Debug, Serialize)]
pub struct MomentItem {
    pub moment_id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub location: RelativeLocation,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_liked: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_favorited: Option<bool>,
}

/// 瞬间详情 - API 文档: MomentCard
#[derive(Debug, Serialize)]
pub struct MomentCard {
    pub moment_id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub location: Location,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<MyMomentState>,
}

/// 我的瞬间互动状态 - API 文档: MyMomentState
#[derive(Debug, Serialize, Clone)]
pub struct MyMomentState {
    pub is_liked: bool,
    pub is_favorited: bool,
}

/// 用户瞬间列表项 - API 文档: UserMomentListItem
#[derive(Debug, Serialize)]
pub struct UserMomentListItem {
    pub moment_id: String,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub location: Location,
}

/// 瞬间点赞项 - API 文档: MomentLikeItem
#[derive(Debug, Serialize)]
pub struct MomentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

/// 点赞状态检查数据
#[derive(Debug, Serialize)]
pub struct CheckLikedData {
    pub is_liked: bool,
}

/// 收藏状态检查数据
#[derive(Debug, Serialize)]
pub struct CheckFavoritedData {
    pub is_favorited: bool,
}

/// 瞬间互动状态数据 - API 文档: MomentInteractionStateData
#[derive(Debug, Serialize)]
pub struct MomentInteractionStateData {
    pub is_liked: bool,
    pub is_favorited: bool,
}

// ============================================================
// 评论相关类型
// ============================================================

/// 评论列表项 - API 文档: CommentListItem
#[derive(Debug, Serialize)]
pub struct CommentListItem {
    pub comment_id: String,
    pub user: UserInfo,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub replies_count: i32,
}

/// 评论点赞项 - API 文档: CommentLikeItem
#[derive(Debug, Serialize)]
pub struct CommentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ============================================================
// 群组相关类型
// ============================================================

/// 群组基础信息 - API 文档: GroupInfo
#[derive(Debug, Serialize)]
pub struct GroupInfo {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
}

/// 群组搜索/列表项 - API 文档: GroupSearchItem
#[derive(Debug, Serialize)]
pub struct GroupItem {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub members_count: i32,
    pub has_password: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
}

/// 群组详情（含我的状态）- API 文档: GroupCard
#[derive(Debug, Serialize)]
pub struct GroupDetailData {
    #[serde(flatten)]
    pub group: GroupItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_role: Option<GroupRole>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_joined: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unread_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_muted: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_pinned: Option<bool>,
}

/// 我在群内的状态 - API 文档: Membership
#[derive(Debug, Serialize)]
pub struct GroupMyStateData {
    pub role: GroupRole,
    pub is_joined: bool,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub joined_at: Option<String>,
}

/// 群组设置 - API 文档: GroupSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GroupSettings {
    pub join_method: GroupJoinMethod,
    pub is_searchable: bool,
    pub max_members: i32,
    pub location: Location,
}

/// 群组设置数据（含 ID）
#[derive(Debug, Serialize)]
pub struct GroupSettingsData {
    pub group_chat_id: String,
    pub settings: GroupSettings,
    pub has_password: bool,
    pub created_at: String,
}

/// 已加入的群组项 - API 文档: JoinedGroupItem
#[derive(Debug, Serialize)]
pub struct JoinedGroupItem {
    #[serde(flatten)]
    pub group_info: GroupInfo,
    pub my_role: GroupRole,
    pub unread_count: i32,
    pub last_activity_at: String,
}

/// 群公告项 - API 文档: GroupAnnouncementItem
#[derive(Debug, Serialize)]
pub struct GroupAnnouncementItem {
    pub announcement_id: String,
    pub group_chat_id: String,
    pub title: String,
    pub content: String,
    pub created_by: UserInfo,
    pub created_at: String,
    pub is_pinned: bool,
}

// ============================================================
// 群组成员相关类型
// ============================================================

/// 群组成员项 - API 文档: GroupMemberItem
#[derive(Debug, Serialize)]
pub struct GroupMemberItem {
    pub user: UserInfo,
    pub role: GroupRole,
    pub joined_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub muted_until: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：成员隐私设置时不返回
}

/// 被封禁成员项 - API 文档: BannedMemberItem
#[derive(Debug, Serialize, Clone)]
pub struct BannedMemberItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    pub banned_at: String,
}

/// 邀请成员返回数据
#[derive(Debug, Serialize)]
pub struct InviteMemberData {
    pub invite_id: String,
    pub expires_at: String,
}

/// 申请加入返回数据
#[derive(Debug, Serialize)]
pub struct ApplyToGroupData {
    pub application_id: String,
}

/// 群组申请项 - API 文档: GroupApplicationItem
#[derive(Debug, Serialize)]
pub struct GroupApplicationItem {
    pub application_id: String,
    pub applicant: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_by: Option<UserInfo>,
}

/// 群组邀请项 - API 文档: GroupInviteItem
#[derive(Debug, Serialize)]
pub struct GroupInviteItem {
    pub invite_id: String,
    pub group: GroupInfo,
    pub inviter: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

// ============================================================
// 私聊相关类型
// ============================================================

/// 私聊最后一条消息 - API 文档: PrivateChatLastMessage
#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub message_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    pub sender_id: i64,
    pub sent_at: String,
}

/// 私聊列表项 - API 文档: PrivateChatListItem
#[derive(Debug, Serialize)]
pub struct PrivateChatListItem {
    pub other_user: UserInfo,
    pub unread_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_online: bool,
}

/// 最近消息 - API 文档: RecentMessages
#[derive(Debug, Serialize)]
pub struct RecentMessages {
    pub items: Vec<PrivateMessageItem>,
    pub has_more: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

/// 私聊详情数据 - API 文档: PrivateChatDetailData
#[derive(Debug, Serialize)]
pub struct PrivateChatDetailData {
    pub other_user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message_limit: Option<MessageLimit>,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub last_activity_at: String,
    pub created_at: String,
    pub can_message: bool,
    pub recent_messages: RecentMessages,
}

// ============================================================
// 消息相关类型
// ============================================================

/// 消息回复引用 - API 文档: MessageReplyTo
#[derive(Debug, Serialize, Clone)]
pub struct MessageReplyTo {
    pub message_id: String,
    pub user_id: i64,
    pub nickname: String,
    pub content: String,
}

/// 消息媒体信息 - API 文档: MessageMedia
#[derive(Debug, Serialize, Clone, Deserialize)]
pub struct MessageMedia {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
}

/// 位置消息信息 - API 文档: MessageLocation
#[derive(Debug, Serialize, Clone, Deserialize)]
pub struct MessageLocation {
    pub location: Location,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub address: Option<String>,
}

/// 发送私聊消息返回数据
#[derive(Debug, Serialize)]
pub struct SendPrivateMessageData {
    pub message_id: String,
}

/// 私聊消息项 - API 文档: PrivateMessageItem
#[derive(Debug, Serialize)]
pub struct PrivateMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<MessageLocation>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
}

/// 发送群聊消息返回数据
#[derive(Debug, Serialize)]
pub struct SendGroupMessageData {
    pub message_id: String,
}

/// 群聊消息项 - API 文档: GroupMessageItem
#[derive(Debug, Serialize)]
pub struct GroupMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<MessageLocation>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_count: Option<i32>,
}

/// 消息已读项 - API 文档: MessageReadItem
#[derive(Debug, Serialize)]
pub struct MessageReadItem {
    pub user: UserInfo,
    pub read_at: String,
}

/// 消息反应项 - API 文档: MessageReactionItem
#[derive(Debug, Serialize)]
pub struct MessageReactionItem {
    pub user: UserInfo,
    #[serde(rename = "type")]
    pub reaction_type: ReactionType,
    pub created_at: String,
}

// ============================================================
// 举报相关类型
// ============================================================

/// 创建举报返回数据
#[derive(Debug, Serialize)]
pub struct CreateReportData {
    pub report_id: String,
}

/// 举报内容快照 - API 文档: ReportTargetSnapshot
#[derive(Debug, Serialize)]
pub struct ReportTargetSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<String>,
}

/// 举报项 - API 文档: ReportItem
#[derive(Debug, Serialize)]
pub struct ReportItem {
    pub report_id: String,
    pub target_type: ReportTargetType,
    pub target_id: String,
    pub reason: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub status: ReportStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub updated_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resolution_note: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reporter: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_snapshot: Option<ReportTargetSnapshot>,
}

// ============================================================
// 通知相关类型
// ============================================================

/// 通知订阅设置 - API 文档: SubscriptionsData
#[derive(Debug, Serialize, Deserialize)]
pub struct SubscriptionsData {
    pub new_follower_enabled: bool,
    pub mutual_follow_enabled: bool,
    pub follow_request_enabled: bool,         // 收到关注请求（需审批时）
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

/// 通知项 - API 文档: NotificationItem
#[derive(Debug, Serialize)]
pub struct NotificationItem {
    pub notification_id: String,
    #[serde(rename = "type")]
    pub notification_type: NotificationType,
    pub is_read: bool,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_type: Option<NotificationTargetType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reference_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_text_preview: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_media_url: Option<String>,
}

/// 未读通知计数数据 - API 文档: UnreadNotificationCountData
#[derive(Debug, Serialize)]
pub struct UnreadNotificationCountData {
    pub total: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub breakdown: Option<UnreadNotificationBreakdown>,
}

/// 未读通知分类计数
#[derive(Debug, Serialize)]
pub struct UnreadNotificationBreakdown {
    pub like: i64,
    pub comment: i64,
    pub follow: i64,
    pub system: i64,
    pub mention: i64,
}

// ============================================================
// 设备令牌相关类型
// ============================================================

/// 设备令牌项 - API 文档: DeviceTokenItem
#[derive(Debug, Serialize)]
pub struct DeviceTokenItem {
    pub device_token_id: String,
    pub platform: DevicePlatform,
    pub device_token: String,
    pub created_at: String,
}

// ============================================================
// 系统健康检查
// ============================================================

/// 健康检查数据 - API 文档: HealthCheckData
#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: SystemHealthStatus,
    pub version: String,
    pub timestamp: String,
}

// ============================================================
// 文件上传相关类型
// ============================================================

/// 上传媒体数据
#[derive(Debug, Serialize)]
pub struct UploadMediaData {
    pub file_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    pub file_size: i64,
    pub mime_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
}

/// 预签名上传数据 - API 文档: PresignUploadData
#[derive(Debug, Serialize)]
pub struct PresignUploadData {
    pub upload_id: String,
    pub upload_url: String,
    pub headers: std::collections::HashMap<String, String>,
    pub expires_at: String,
}

// ============================================================
// 信息流相关类型
// ============================================================

/// 信息流群组项 - API 文档: FeedGroupItem
#[derive(Debug, Serialize)]
pub struct FeedGroupItem {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub has_password: bool,
    pub location: RelativeLocation,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friends_in_group: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<GroupMyStateData>,
}

/// 信息流用户项 - API 文档: FeedUserItem
#[derive(Debug, Serialize)]
pub struct FeedUserItem {
    #[serde(flatten)]
    pub user: UserItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutual_friends_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_tags: Option<Vec<String>>,
}

/// 信息流瞬间项 - API 文档: FeedMomentItem
#[derive(Debug, Serialize)]
pub struct FeedMomentItem {
    pub moment_id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub location: RelativeLocation,
    pub is_liked: bool,
    pub is_favorited: bool,
}

/// 热门标签项 - API 文档: TrendingTagItem
#[derive(Debug, Serialize)]
pub struct TrendingTagItem {
    pub name: String,
    pub usage_count: i32,
    pub trend: TagTrend,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hot_moment_preview: Option<FeedMomentItem>,
}

/// 信息流热门数据 - API 文档: FeedTrendingData
#[derive(Debug, Serialize)]
pub struct FeedTrendingData {
    pub trending_tags: Vec<TrendingTagItem>,
    pub hot_moments: Vec<FeedMomentItem>,
}

// ============================================================
// 标签相关类型
// ============================================================

/// 标签项 - API 文档: TagItem
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct TagItem {
    pub name: String,
    pub usage_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trend: Option<TagTrend>,
}

/// 热门标签数据 - API 文档: TrendingTagsData
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct TrendingTagsData {
    pub items: Vec<TagItem>,
}

/// 标签建议数据 - API 文档: TagSuggestData
#[derive(Debug, Serialize)]
pub struct TagSuggestData {
    pub items: Vec<TagSuggestItem>,
}

/// 标签建议项 - API 文档: TagSuggestItem
#[derive(Debug, Serialize)]
pub struct TagSuggestItem {
    pub name: String,
    pub usage_count: i32,
}

// ============================================================
// 系统配置相关类型
// ============================================================

/// 上传配置 - API 文档: UploadConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct UploadConfig {
    pub image_max_size: i64,
    pub video_max_size: i64,
    pub audio_max_size: i64,
    pub file_max_size: i64,
    pub allowed_image_types: Vec<String>,
    pub allowed_video_types: Vec<String>,
}

/// 内容默认值 - API 文档: ContentDefaults
#[derive(Debug, Serialize, Deserialize)]
pub struct ContentDefaults {
    pub default_location_name: String,
    pub default_group_avatar: String,
    pub default_user_avatar: String,
    pub default_group_settings: GroupSettings,
}

/// 功能开关 - API 文档: FeatureSwitches
#[derive(Debug, Serialize, Deserialize)]
pub struct FeatureSwitches {
    pub enable_public_registration: bool,
    pub enable_guest_login: bool,
    pub enable_moments: bool,
    pub enable_nearby_people: bool,
}

/// 联系信息 - API 文档: ContactInfo
#[derive(Debug, Serialize, Deserialize)]
pub struct ContactInfo {
    pub support_email: String,
    pub privacy_policy_url: String,
    pub terms_of_service_url: String,
    pub official_website: String,
}

/// 应用限制 - API 文档: AppLimits
#[derive(Debug, Serialize, Deserialize)]
pub struct AppLimits {
    pub max_group_members: i32,
    pub max_moment_tags: i32,
    pub max_moment_images: i32,
    pub moment_description_max_length: i32,
    pub comment_max_length: i32,
    pub search_radius_max: i32,
}

/// 全局设置 - API 文档: GlobalSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GlobalSettings {
    pub upload: UploadConfig,
    pub defaults: ContentDefaults,
    pub features: FeatureSwitches,
    pub contact: ContactInfo,
    pub limits: AppLimits,
}

/// 管理员全局设置数据 - API 文档: AdminGlobalSettingsData
#[derive(Debug, Serialize)]
pub struct AdminGlobalSettingsData {
    #[serde(flatten)]
    pub settings: GlobalSettings,
    pub updated_at: String,
    pub updated_by_user_id: i64,
}

/// 更新上传策略响应
#[derive(Debug, Serialize)]
pub struct UpdateUploadConfigData {
    pub upload: UploadConfig,
    pub updated_at: String,
}

/// 更新内容默认值响应
#[derive(Debug, Serialize)]
pub struct UpdateDefaultsConfigData {
    pub defaults: ContentDefaults,
    pub updated_at: String,
}

/// 更新功能开关响应
#[derive(Debug, Serialize)]
pub struct UpdateFeaturesConfigData {
    pub features: FeatureSwitches,
    pub updated_at: String,
}

/// 更新联系信息响应
#[derive(Debug, Serialize)]
pub struct UpdateContactConfigData {
    pub contact: ContactInfo,
    pub updated_at: String,
}

/// 更新系统限制响应
#[derive(Debug, Serialize)]
pub struct UpdateLimitsConfigData {
    pub limits: AppLimits,
    pub updated_at: String,
}

// ============================================================
// 管理员 RBAC 相关类型
// ============================================================

/// 管理员权限项 - API 文档: AdminPermissionItem
#[derive(Debug, Serialize)]
pub struct AdminPermissionItem {
    pub key: String,
    pub description: String,
}

/// 管理员权限数据
#[derive(Debug, Serialize)]
pub struct AdminPermissionsData {
    pub items: Vec<AdminPermissionItem>,
}

/// 管理员角色项 - API 文档: AdminRoleItem
#[derive(Debug, Serialize)]
pub struct AdminRoleItem {
    pub role_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub permissions: Vec<String>,
}

/// 管理员角色数据
#[derive(Debug, Serialize)]
pub struct AdminRolesData {
    pub items: Vec<AdminRoleItem>,
}

/// 管理员角色基础信息 - API 文档: AdminRoleBasic
#[derive(Debug, Serialize)]
pub struct AdminRoleBasic {
    pub role_id: String,
    pub name: String,
}

/// 管理员项 - API 文档: AdminAdminItem
#[derive(Debug, Serialize)]
pub struct AdminAdminItem {
    pub user: UserInfo,
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

/// 管理员列表数据
#[derive(Debug, Serialize)]
pub struct AdminAdminsData {
    pub items: Vec<AdminAdminItem>,
}

/// 当前管理员数据 - API 文档: AdminMeData
#[derive(Debug, Serialize)]
pub struct AdminMeData {
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

// ============================================================
// 管理员用户管理相关类型
// ============================================================

/// 管理员用户项 - API 文档: AdminUserItem
#[derive(Debug, Serialize)]
pub struct AdminUserItem {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    pub is_online: bool,
}

/// 管理员用户详情 - API 文档: AdminUserDetail
#[derive(Debug, Serialize)]
pub struct AdminUserDetail {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
    pub is_online: bool,
    pub stats: UserStatsData,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ban_history: Option<Vec<AdminBanRecord>>,
}

/// 管理员封禁记录 - API 文档: AdminBanRecord
#[derive(Debug, Serialize, Clone)]
pub struct AdminBanRecord {
    pub banned_at: String,
    pub banned_by: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration_days: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_by: Option<UserInfo>,
}

// ============================================================
// 管理员群组管理相关类型
// ============================================================

/// 管理员群组项 - API 文档: AdminGroupItem
#[derive(Debug, Serialize)]
pub struct AdminGroupItem {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
    pub owner: UserInfo,
    pub created_at: String,
}

/// 管理员群组详情 - API 文档: AdminGroupDetail
#[derive(Debug, Serialize)]
pub struct AdminGroupDetail {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub max_members: i32,
    pub owner: UserInfo,
    pub created_at: String,
    pub settings: AdminGroupSettingsSummary,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub admins: Option<Vec<UserInfo>>,
    pub stats: AdminGroupStats,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dismissal_history: Option<Vec<AdminDismissalRecord>>,
}

#[derive(Debug, Serialize)]
pub struct AdminGroupSettingsSummary {
    pub join_method: GroupJoinMethod,
    pub is_searchable: bool,
    pub has_password: bool,
}

#[derive(Debug, Serialize)]
pub struct AdminGroupStats {
    pub messages_count: i32,
    pub active_members_today: i32,
}

#[derive(Debug, Serialize, Clone)]
pub struct AdminDismissalRecord {
    pub dismissed_at: String,
    pub dismissed_by: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ============================================================
// 管理员审计日志相关类型
// ============================================================

/// 审计日志项 - API 文档: AuditLogItem
#[derive(Debug, Serialize, Clone)]
pub struct AuditLogItem {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
}

/// 审计日志详情 - API 文档: AuditLogDetail
#[derive(Debug, Serialize)]
pub struct AuditLogDetail {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub before_state: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub after_state: Option<Value>,
}

/// 导出结果 - API 文档: AuditLogExportResult
#[derive(Debug, Serialize)]
pub struct AuditLogExportResult {
    pub export_id: String,
    pub status: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub download_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

// ============================================================
// 管理员系统公告相关类型
// ============================================================

/// 创建公告返回数据
#[derive(Debug, Serialize)]
pub struct CreateAnnouncementData {
    pub announcement_id: String,
    pub created_at: String,
}

/// 系统公告项 - API 文档: SystemAnnouncementItem
#[derive(Debug, Serialize)]
pub struct SystemAnnouncementItem {
    pub announcement_id: String,
    pub title: String,
    pub content: String,
    pub severity: AnnouncementSeverity,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    pub created_at: String,
    pub created_by: UserInfo,
}

// ============================================================
// 地图渲染相关类型
// ============================================================

/// 地图点 - API 文档: GeoPoint
#[derive(Debug, Serialize, Clone)]
pub struct MapPoint {
    pub latitude: f64,
    pub longitude: f64,
}

/// 地图聚合数据 - API 文档: MapClusterData
#[derive(Debug, Serialize)]
pub struct MapClusterData {
    pub count: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub label: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub boundary: Option<Vec<MapPoint>>,
}

/// 地图实体数据 - API 文档: MapEntityData
#[derive(Debug, Serialize)]
pub struct MapEntityData {
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subtitle: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub payload_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_highlighted: Option<bool>,
}

/// 地图渲染项 - API 文档: MapRenderItem
#[derive(Debug, Serialize)]
pub struct MapRenderItem {
    pub id: String,
    #[serde(rename = "type")]
    pub item_type: MapItemType,
    pub location: MapPoint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cluster: Option<MapClusterData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity: Option<MapEntityData>,
}

/// 地图渲染数据 - API 文档: MapRenderData
#[derive(Debug, Serialize)]
pub struct MapRenderData {
    pub items: Vec<MapRenderItem>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub toast_message: Option<String>,
}
