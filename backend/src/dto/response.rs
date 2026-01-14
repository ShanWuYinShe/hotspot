use serde::{Deserialize, Serialize};

/// 统一API响应格式
#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: None,
            data: Some(data),
        }
    }
}

impl ApiResponse<()> {
    pub fn success_empty() -> Self {
        Self {
            code: 0,
            message: None,
            data: None,
        }
    }
}

// ==================== 通用类型（API 文档定义） ====================

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

/// 位置信息 - API 文档: Location
#[derive(Debug, Serialize, Clone)]
pub struct Location {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
}

// ==================== 通用类型定义 ====================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserRelation {
    pub is_mutual_follow: bool,
    pub is_following: bool,
    pub is_followed_by: bool,
    pub is_blocked: bool,
    pub is_blocking: bool,
    pub has_pending_follow_request: bool,
}

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

#[derive(Debug, Serialize)]
pub struct PaginationMeta {
    pub has_more: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct PaginatedData<T> {
    pub items: Vec<T>,
    pub pagination: PaginationMeta,
}

/// 群组设置 - API 文档: GroupSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GroupSettings {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_method: Option<String>, // 'open' | 'approval' | 'invite_only'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_searchable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub longitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub latitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
}

// ==================== 3.1 认证 API ====================

#[derive(Debug, Serialize, Clone)]
pub struct AuthData {
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    /// access_token 过期时间
    pub access_token_expires_at: String,
    /// refresh_token 过期时间
    pub refresh_token_expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct CheckStatusData {
    pub user_id: i64,
    pub nickname: String,
    pub is_valid: bool,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct SendVerificationCodeData {
    pub code_id: String,
    pub expires_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub retry_after: Option<i32>,
}

#[derive(Debug, Serialize)]
pub struct VerifyCodeData {
    pub verified: bool,
    pub verify_token: String,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct RequestPasswordResetData {
    pub reset_id: String,
}

#[derive(Debug, Serialize)]
pub struct VerifyResetCodeData {
    pub reset_token: String,
}

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

// ==================== 3.2 用户 API ====================

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
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<Location>,
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct UserStatsData {
    pub moments_count: i32,
    pub friends_count: i32,
    pub followers_count: i32,
    pub following_count: i32,
    pub groups_count: i32,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UserSettingsData {
    // 隐私设置
    pub privacy_level: String, // 'public' | 'friends' | 'private'
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: String,   // 'public' | 'friends' | 'private'
    pub following_visible: String, // 'public' | 'friends' | 'private'
    pub followers_visible: String, // 'public' | 'friends' | 'private'
    pub moments_visible: String,   // 'public' | 'friends' | 'private'
    pub stats_visible: String,     // 'public' | 'friends' | 'private'

    // 社交设置
    pub allow_follow: bool,         // 是否允许被关注
    pub allow_private_chat: String, // 'everyone' | 'friends' | 'nobody'

    // 搜索设置
    pub searchable: bool,
    pub phone_searchable: bool,
    pub nickname_searchable: bool,
    pub location_searchable: bool,
}

#[derive(Debug, Serialize)]
pub struct FriendListItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
    pub followed_at: String,      // 我关注对方的时间
    pub followed_back_at: String, // 对方关注我的时间
}

#[derive(Debug, Serialize)]
pub struct FollowingItem {
    pub user: UserInfo,
    pub followed_at: String, // 我关注的时间
    pub is_mutual: bool,     // 是否互相关注（是否是好友）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct FollowerItem {
    pub user: UserInfo,
    pub followed_at: String,     // 对方关注我的时间
    pub is_following_back: bool, // 我是否关注了对方
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct BlockedUserItem {
    pub user: UserInfo,
    pub blocked_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ==================== 3.3 生活瞬间 API ====================

#[derive(Debug, Serialize)]
pub struct MomentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ==================== 3.4 评论与点赞 API ====================

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

#[derive(Debug, Serialize)]
pub struct CommentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ==================== 3.5 私聊管理 API ====================

#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub message_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
    pub sender_id: i64,
    pub sent_at: String,
}

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

/// 私聊详情数据 - API 文档: PrivateChatDetailData
/// 用户关系信息通过单独的 API 获取（get_relationship）
#[derive(Debug, Serialize)]
pub struct PrivateChatDetailData {
    pub other_user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message_limit: Option<MessageLimit>,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub created_at: String,
    pub can_message: bool,
}

#[derive(Debug, Serialize)]
pub struct UserDetailData {
    #[serde(flatten)]
    pub user: UserItem,
    pub stats: UserStatsData,
}

#[derive(Debug, Serialize)]
pub struct GroupDetailData {
    #[serde(flatten)]
    pub group: GroupItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_role: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_joined: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unread_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_muted: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_pinned: Option<bool>,
}

// ==================== 群组管理 API ====================

#[derive(Debug, Serialize)]
pub struct CreateGroupChatData {
    pub group_chat_id: String,
}

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
    // API文档: location 应该是可选的（群组通常有公开位置，但不是必须的）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
}

#[derive(Debug, Serialize)]
pub struct JoinedGroupItem {
    #[serde(flatten)]
    pub group_info: GroupInfo,
    pub my_role: String,
    pub unread_count: i32,
    pub last_activity_at: String,
}

// ==================== 3.7 群组成员管理 API ====================

#[derive(Debug, Serialize)]
pub struct GroupMemberItem {
    pub user: UserInfo,
    pub role: String, // 'owner' | 'admin' | 'member'
    pub joined_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub muted_until: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct InviteMemberData {
    pub invite_id: String,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct ApplyToGroupData {
    pub application_id: String,
}

#[derive(Debug, Serialize)]
pub struct GroupApplicationItem {
    pub application_id: String,
    pub applicant: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: String, // 'pending' | 'accepted' | 'rejected'
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct GroupInviteItem {
    pub invite_id: String,
    pub group: GroupInfo,
    pub inviter: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: String, // 'pending' | 'accepted' | 'rejected'
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

// ==================== 3.8 私聊消息 API ====================

#[derive(Debug, Serialize, Clone)]
pub struct MessageReplyTo {
    pub message_id: String,
    pub user_id: i64,
    pub nickname: String,
    pub content: String,
}

#[derive(Debug, Serialize)]
pub struct SendPrivateMessageData {
    pub message_id: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateMessageItem {
    pub message_id: String,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>, // 文本内容（type='text' 时）
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to_message_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    // 时间字段
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
}

// ==================== 3.9 群聊消息 API ====================

#[derive(Debug, Serialize)]
pub struct SendGroupMessageData {
    pub message_id: String,
}

#[derive(Debug, Serialize)]
pub struct GroupMessageItem {
    pub message_id: String,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>, // 文本内容（type='text'/'system' 时）
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to_message_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>, // @提及的用户ID列表
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    // 时间字段
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<i64>, // 撤回操作者用户ID
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_count: Option<i32>,
}

#[derive(Debug, Serialize)]
pub struct MessageReadItem {
    pub user: UserInfo,
    pub read_at: String,
}

#[derive(Debug, Serialize)]
pub struct MessageReactionItem {
    pub user: UserInfo,
    pub r#type: String,
    pub created_at: String,
}

// ==================== 3.11 举报审核 API ====================

#[derive(Debug, Serialize)]
pub struct CreateReportData {
    pub report_id: String,
}

#[derive(Debug, Serialize)]
pub struct ReportItem {
    pub report_id: String,
    pub target_type: String,
    pub target_id: String,
    pub reason: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub status: String, // 'pending' | 'reviewing' | 'resolved' | 'dismissed'
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
}

// ==================== 3.12 通知订阅设置 API ====================

#[derive(Debug, Serialize, Deserialize)]
pub struct SubscriptionsData {
    pub new_follower_enabled: bool,
    pub mutual_follow_enabled: bool,
    pub group_invite_enabled: bool,
    pub group_invite_response_enabled: bool,
    pub group_application_enabled: bool,
    pub group_application_response_enabled: bool,
    pub message_mention_enabled: bool,
    pub moment_like_enabled: bool,
    pub moment_comment_enabled: bool,
    pub comment_reply_enabled: bool,
    pub system_announcement_enabled: bool,
}

// ==================== 通知 API ====================

#[derive(Debug, Serialize)]
pub struct NotificationItem {
    pub notification_id: String,
    pub r#type: String,
    pub title: String,
    pub content: String,
    pub created_at: String,
    pub read: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reference_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_preview: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct UnreadNotificationCountData {
    pub total: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub breakdown: Option<UnreadNotificationBreakdown>,
}

#[derive(Debug, Serialize)]
pub struct UnreadNotificationBreakdown {
    pub like: i64,
    pub comment: i64,
    pub follow: i64,
    pub system: i64,
    pub mention: i64,
}

// ==================== 3.14 系统健康检查 API ====================

#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: String, // 'healthy' | 'degraded' | 'unhealthy'
    pub version: String,
    pub timestamp: String,
}

// ==================== 3.16 文件与媒体上传 API ====================

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

#[derive(Debug, Serialize, Clone, serde::Deserialize)]
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

#[derive(Debug, Serialize)]
pub struct PresignUploadData {
    pub upload_id: String,
    pub upload_url: String,
    pub headers: std::collections::HashMap<String, String>,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct CheckChunkedUploadData {
    pub upload_id: String,
    pub chunk_size: i64,
    pub uploaded_chunks: Vec<i32>,
}

#[derive(Debug, Serialize)]
pub struct InitiateUploadData {
    pub upload_id: String,
    /// 上传目标：backend=走后端分块上传；object_storage=对象存储直传（预留）
    pub upload_target: String,
    /// 分块大小（字节）
    pub chunk_size: i64,
    pub chunk_count: i32,
    /// 分块上传 URL 列表（当分块很多时可能为空，客户端用 upload_part_url 按需获取）
    pub upload_urls: Vec<String>,
    /// 上传会话过期时间
    pub expires_at: String,
}

// ==================== 信息流 API ====================

#[derive(Debug, Serialize)]
pub struct FeedGroupItem {
    #[serde(flatten)]
    pub group: GroupItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>, // 推荐理由（如"好友在这里"、"附近热门"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friends_in_group: Option<i32>, // 群内好友数量
    // Zero-Lookup
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_role: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_joined: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct FeedUserItem {
    #[serde(flatten)]
    pub user: UserItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>, // 推荐理由（如"共同好友"、"相同标签"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutual_friends_count: Option<i32>, // 共同好友数
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_tags: Option<Vec<String>>, // 共同标签
}

#[derive(Debug, Serialize)]
pub struct TrendingTagItem {
    pub name: String,
    pub usage_count: i32,
    pub trend: String, // 'rising' | 'stable' | 'falling'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hot_moment_preview: Option<MomentItem>,
}

#[derive(Debug, Serialize)]
pub struct FeedTrendingData {
    pub trending_tags: Vec<TrendingTagItem>,
    pub hot_moments: Vec<MomentItem>, // 最多5个热门瞬间预览
}

// ==================== 标签管理 API ====================

#[derive(Debug, Serialize)]
pub struct TagData {
    pub name: String,
    pub usage_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trend: Option<String>, // 'rising' | 'stable' | 'falling'
}

#[derive(Debug, Serialize)]
pub struct TrendingTagsData {
    pub items: Vec<TagData>,
}

#[derive(Debug, Serialize)]
pub struct TagSuggestData {
    pub items: Vec<TagSuggestItem>,
}

#[derive(Debug, Serialize)]
pub struct TagSuggestItem {
    pub name: String,
    pub usage_count: i32,
}

// ==================== 3.20 系统配置管理 API ====================

#[derive(Debug, Serialize, Deserialize)]
pub struct UploadConfig {
    pub image_max_size: i64,
    pub video_max_size: i64,
    pub audio_max_size: i64,
    pub file_max_size: i64,
    pub allowed_image_types: Vec<String>,
    pub allowed_video_types: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ContentDefaults {
    pub default_location_name: String,
    pub default_group_avatar: String,
    pub default_user_avatar: String,
    pub default_group_settings: GroupSettings,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct FeatureSwitches {
    pub enable_public_registration: bool,
    pub enable_guest_login: bool,
    pub enable_moments: bool,
    pub enable_nearby_people: bool,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ContactInfo {
    pub support_email: String,
    pub privacy_policy_url: String,
    pub terms_of_service_url: String,
    pub official_website: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct AppLimits {
    pub max_group_members: i32,
    pub max_moment_tags: i32,
    pub max_moment_images: i32,
    pub moment_description_max_length: i32,
    pub comment_max_length: i32,
    pub search_radius_max: i32,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct GlobalSettings {
    pub upload: UploadConfig,
    pub defaults: ContentDefaults,
    pub features: FeatureSwitches,
    pub contact: ContactInfo,
    pub limits: AppLimits,
    pub version: String,
}

#[derive(Debug, Serialize)]
pub struct FetchSettingsData {
    pub action: String, // 'update' | 'no_change' | 'force_update'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub config: Option<GlobalSettings>,
    pub latest_version: String,
}

#[derive(Debug, Serialize)]
pub struct AdminGlobalSettingsData {
    #[serde(flatten)]
    pub settings: GlobalSettings,
    pub updated_at: String,
    pub updated_by_user_id: i64,
}

// ==================== 管理员 RBAC（admin/rbac） ====================

#[derive(Debug, Serialize)]
pub struct AdminPermissionItem {
    pub key: String,
    pub description: String,
}

#[derive(Debug, Serialize)]
pub struct AdminPermissionsData {
    pub items: Vec<AdminPermissionItem>,
}

#[derive(Debug, Serialize)]
pub struct AdminRoleItem {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub permissions: Vec<String>,
}

#[derive(Debug, Serialize)]
pub struct AdminRolesData {
    pub items: Vec<AdminRoleItem>,
}

#[derive(Debug, Serialize)]
pub struct AdminRoleBasic {
    pub id: String,
    pub name: String,
}

#[derive(Debug, Serialize)]
pub struct AdminAdminItem {
    pub user: UserInfo,
    /// '*' 表示拥有全部权限（超级管理员）
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

#[derive(Debug, Serialize)]
pub struct AdminAdminsData {
    pub items: Vec<AdminAdminItem>,
}

#[derive(Debug, Serialize)]
pub struct AdminMeData {
    /// '*' 表示拥有全部权限（超级管理员）
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

// ==================== 补充响应类型 ====================

#[derive(Debug, Serialize)]
pub struct GroupSettingsData {
    pub group_chat_id: String,
    pub settings: GroupSettings,
    pub has_password: bool,
    pub created_at: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct BannedMemberItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    pub banned_at: String,
}

#[derive(Debug, Serialize)]
pub struct DeviceTokenItem {
    pub id: String,
    pub platform: String, // 'ios' | 'android' | 'harmony'
    pub device_token: String,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct MomentItem {
    pub moment_id: String, // Changed from id to moment_id per spec
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,
    pub privacy_level: String, // 'public' | 'friends' | 'private'
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    // 位置信息必填：与API规范和数据库schema一致
    pub location: RelativeLocation,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_liked: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_favorited: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct CheckLikedData {
    pub is_liked: bool,
}

#[derive(Debug, Serialize)]
pub struct CheckFavoritedData {
    pub is_favorited: bool,
}

#[derive(Debug, Serialize, Clone)]
pub struct UserItem {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    pub user_type: crate::types::enums::UserType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    /// 在线状态（受 online_visible 隐私设置控制）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    /// 最后上线时间（受 last_online_visible 隐私设置控制，仅离线时返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    /// 位置信息（受 location_visible 隐私设置控制，为 false 时返回 null）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    /// 关系状态（可选）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relation: Option<UserRelation>,
}

// ==================== Map 聚合 / 视口查询（public） ====================

#[derive(Debug, Serialize, Clone)]
pub struct MapPoint {
    pub latitude: f64,
    pub longitude: f64,
}

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

#[derive(Debug, Serialize)]
pub struct MapRenderItem {
    pub id: String,
    pub r#type: String, // 'cluster' | 'user' | 'group' | 'moment'
    pub location: MapPoint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cluster: Option<MapClusterData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity: Option<MapEntityData>,
}

#[derive(Debug, Serialize)]
pub struct MapRenderData {
    pub items: Vec<MapRenderItem>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub toast_message: Option<String>,
}

// ==================== 管理员用户管理 (Admin) ====================

#[derive(Debug, Serialize)]
pub struct AdminUserItem {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
    pub status: String, // 'normal' | 'banned'
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    // 管理员视图补充字段
    pub login_id: String,
    pub is_online: bool,
}

// ==================== 管理员群组管理 (Admin) ====================

#[derive(Debug, Serialize)]
pub struct AdminGroupItem {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    pub owner: UserInfo,
    pub created_at: String,
}

// ==================== 管理员系统公告管理 (Admin) ====================

#[derive(Debug, Serialize)]
pub struct CreateAnnouncementData {
    pub announcement_id: String,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct SystemAnnouncementItem {
    pub announcement_id: String,
    pub title: String,
    pub content: String,
    pub severity: String, // 'info' | 'warning' | 'error'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_by: Option<UserInfo>,
}
