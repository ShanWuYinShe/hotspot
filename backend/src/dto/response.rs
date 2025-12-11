use serde::Serialize;

/// 统一API响应格式
#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: "Success".to_string(),
            data: Some(data),
        }
    }
}

impl ApiResponse<()> {
    pub fn success_empty() -> Self {
        Self {
            code: 0,
            message: "Success".to_string(),
            data: None,
        }
    }
}

// ==================== 通用类型（API 文档定义） ====================

/// 地理数据 - API 文档: GeoData
#[derive(Debug, Serialize, Clone)]
pub struct GeoData {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: i32,
    pub degrees: f64,
}

/// 消息限制 - API 文档: MessageLimit
#[derive(Debug, Serialize)]
pub struct MessageLimit {
    pub remaining: i32,
    pub total: i32,
    pub is_unlocked: bool,
}

// ==================== 通用类型定义 ====================

#[derive(Debug, Clone, Serialize)]
pub struct UserInfo {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
}

#[derive(Debug, Serialize)]
pub struct UserRelationship {
    pub is_mutual_follow: bool,
    pub is_following: bool,
    pub is_followed_by: bool,
    pub is_blocked: bool,
    pub is_blocking: bool,
}

#[derive(Debug, Serialize)]
pub struct GroupInfo {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
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
#[derive(Debug, Serialize)]
pub struct GroupSettings {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_method: Option<String>, // 'open' | 'approval' | 'invite_only'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_searchable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub password_hash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub longitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub latitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
}

// ==================== 3.1 认证 API ====================

#[derive(Debug, Serialize)]
pub struct AuthData {
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
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
pub struct RequestPasswordResetData {
    pub reset_id: String,
}

#[derive(Debug, Serialize)]
pub struct VerifyResetCodeData {
    pub reset_token: String,
}

#[derive(Debug, Serialize)]
pub struct SessionItem {
    pub id: String,
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
    pub location_name: Option<String>,
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

#[derive(Debug, Serialize)]
pub struct UserSettingsData {
    // 隐私设置
    pub privacy_level: String,       // 'public' | 'friends' | 'private'
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: String,     // 'public' | 'friends' | 'private'
    pub following_visible: String,   // 'public' | 'friends' | 'private'
    pub followers_visible: String,   // 'public' | 'friends' | 'private'
    pub moments_visible: String,     // 'public' | 'friends' | 'private'
    pub stats_visible: String,       // 'public' | 'friends' | 'private'
    
    // 社交设置
    pub allow_follow: bool,          // 是否允许被关注
    pub allow_private_chat: String,  // 'everyone' | 'friends' | 'nobody'
    
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
    pub followed_at: String,       // 我关注对方的时间
    pub followed_back_at: String,  // 对方关注我的时间
}

#[derive(Debug, Serialize)]
pub struct FollowingItem {
    pub user: UserInfo,
    pub followed_at: String,       // 我关注的时间
    pub is_mutual: bool,           // 是否互相关注（是否是好友）
}

#[derive(Debug, Serialize)]
pub struct FollowerItem {
    pub user: UserInfo,
    pub followed_at: String,       // 对方关注我的时间
    pub is_following_back: bool,   // 我是否关注了对方
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
pub struct CreateMomentData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct MomentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ==================== 3.4 评论与点赞 API ====================

#[derive(Debug, Serialize)]
pub struct CreateCommentData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct CommentListItem {
    pub id: String,
    pub user: UserInfo,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to_user: Option<UserInfo>,
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
pub struct CreatePrivateChatData {
    pub id: String,           // 私聊ID
}

#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file'
    pub sender_id: i64,
    pub sent_at: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateChatListItem {
    pub id: String,           // 私聊ID
    pub other_user: UserInfo,
    pub unread_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
}

/// 私聊详情数据 - API 文档: PrivateChatDetailData
/// 用户状态和关系信息通过单独的 API 获取（get_status、get_relationship）
#[derive(Debug, Serialize)]
pub struct PrivateChatDetailData {
    pub id: String,
    pub other_user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message_limit: Option<MessageLimit>,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_archived: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub created_at: String,
}

// ==================== 群组管理 API ====================

#[derive(Debug, Serialize)]
pub struct CreateGroupChatData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct GroupSearchItem {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub members_count: i32,
    pub has_password: bool,
    pub geo: GeoData,
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
}

#[derive(Debug, Serialize)]
pub struct InviteMemberData {
    pub id: String,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct ApplyToGroupData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct GroupApplicationItem {
    pub id: String,
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
    pub id: String,
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

#[derive(Debug, Serialize)]
pub struct SendPrivateMessageData {
    pub id: String,
    pub sent_at: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateMessageItem {
    pub id: String,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,  // 文本内容（type='text' 时）
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<String>,
    // 媒体消息字段
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
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
    pub id: String,
    pub sent_at: String,
}

#[derive(Debug, Serialize)]
pub struct GroupMessageItem {
    pub id: String,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,  // 文本内容（type='text'/'system' 时）
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<i64>>,  // @提及的用户ID列表
    // 媒体消息字段
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    // 时间字段
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<i64>,  // 撤回操作者用户ID
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
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct ReportItem {
    pub id: String,
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

#[derive(Debug, Serialize)]
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

// ==================== 3.14 系统健康检查 API ====================

#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: String, // 'healthy' | 'degraded' | 'unhealthy'
    pub version: String,
    pub timestamp: String,
}

// ==================== 3.16 文件与媒体上传 API ====================

#[derive(Debug, Serialize)]
pub struct UploadMediaData {
    pub file_id: String,           // 文件ID（用于后续业务API）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    pub file_size: i64,
    pub mime_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<i32>,     // 时长（秒，仅音频/视频）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,        // 宽度（像素，仅图片/视频）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,       // 高度（像素，仅图片/视频）
}

#[derive(Debug, Serialize)]
pub struct InitiateUploadData {
    pub upload_id: String,
    pub chunk_count: i32,
    pub upload_urls: Vec<String>,
}

#[derive(Debug, Serialize)]
pub struct UploadStatusData {
    pub upload_id: String,
    pub status: String, // 'in_progress' | 'completed' | 'failed'
    pub uploaded_parts: Vec<i32>,
    pub total_parts: i32,
    pub created_at: String,
    pub expires_at: String,
}

// ==================== 信息流 API ====================

#[derive(Debug, Serialize)]
pub struct FeedMomentItem {
    #[serde(flatten)]
    pub moment: MomentSearchItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,  // 推荐理由（如"好友动态"、"附近热门"）
}

#[derive(Debug, Serialize)]
pub struct FeedGroupItem {
    #[serde(flatten)]
    pub group: GroupSearchItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,  // 推荐理由（如"好友在这里"、"附近热门"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friends_in_group: Option<i32>,  // 群内好友数量
}

#[derive(Debug, Serialize)]
pub struct FeedUserItem {
    #[serde(flatten)]
    pub user: UserSearchItem,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,  // 推荐理由（如"共同好友"、"相同标签"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutual_friends_count: Option<i32>,  // 共同好友数
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_tags: Option<Vec<String>>,  // 共同标签
}

#[derive(Debug, Serialize)]
pub struct TrendingTagItem {
    pub name: String,
    pub usage_count: i32,
    pub trend: String,  // 'rising' | 'stable' | 'falling'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hot_moment_preview: Option<MomentSearchItem>,
}

#[derive(Debug, Serialize)]
pub struct FeedTrendingData {
    pub trending_tags: Vec<TrendingTagItem>,
    pub hot_moments: Vec<MomentSearchItem>,  // 最多5个热门瞬间预览
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

#[derive(Debug, Serialize)]
pub struct CreateConfigData {
    pub key: String,
}

#[derive(Debug, Serialize)]
pub struct ConfigItem {
    pub key: String,
    pub value: String,
    pub r#type: String, // 'string' | 'number' | 'boolean'
    pub category: String,
    pub description: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize)]
pub struct ConfigHistoryItem {
    pub id: String,
    pub key: String,
    pub old_value: String,
    pub new_value: String,
    pub changed_by: i64,
    pub changed_at: String,
}

// ==================== 补充响应类型 ====================

#[derive(Debug, Serialize)]
pub struct GroupSettingsData {
    pub id: String,
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
pub struct UploadChunkData {
    pub part_number: i32,
    pub etag: String,
}

#[derive(Debug, Serialize)]
pub struct MomentSearchItem {
    pub id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<String>>,
    pub tags: Vec<String>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub geo: GeoData,
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
pub struct UserSearchItem {
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub geo: GeoData,
}

#[derive(Debug, Serialize)]
pub struct UserStatusData {
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
}

