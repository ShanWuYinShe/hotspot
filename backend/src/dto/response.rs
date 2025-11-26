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

// ==================== 通用类型定义 (2.2) ====================

#[derive(Debug, Clone, Serialize)]
pub struct UserInfo {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct UserRelationship {
    pub is_friend: bool,
    pub is_following: bool,
    pub is_followed_by: bool,
    pub is_blocked: bool,
    pub is_blocking: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friend_request_status: Option<String>, // 'sent' | 'received'
}

#[derive(Debug, Serialize)]
pub struct GroupInfo {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub member_count: i32,
}

#[derive(Debug, Serialize)]
pub struct MomentInfo {
    pub id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub images: Vec<String>,
    pub location_name: String,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
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

#[derive(Debug, Serialize)]
pub struct LocationInfo {
    pub latitude: f64,
    pub longitude: f64,
    pub name: String,
}

#[derive(Debug, Serialize)]
pub struct GroupSettings {
    pub join_method: String, // 'open' | 'approval' | 'invite_only'
    pub is_searchable: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub password_hash: Option<String>,
}

// ==================== 3.1 认证 API ====================

#[derive(Debug, Serialize)]
pub struct AuthData {
    pub user_id: String,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct CheckStatusData {
    pub user_id: String,
    pub nickname: String,
    pub is_valid: bool,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct SetPasswordData {
    pub can_login: bool,
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
pub struct DiscoveredUserItem {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub location_name: String,
    pub distance: i32, // 米 (Int)
}

#[derive(Debug, Serialize)]
pub struct MyProfileData {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<String>,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct UserDetailData {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<String>,
    pub created_at: String,
    pub relationship: UserRelationship,
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
    pub privacy_level: String, // 'public' | 'friends' | 'private'
    pub is_location_sharing: bool,
    pub can_be_followed: bool,
    pub allow_private_chat: bool,
}

#[derive(Debug, Serialize)]
pub struct SendFriendRequestData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct FriendListItem {
    pub user: UserInfo,
    pub relation_id: String,
    pub status: String, // 'pending' | 'accepted'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct FriendRequestItem {
    pub id: String,
    pub from_user: UserInfo,
    pub to_user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: String, // 'pending'
    pub created_at: String,
}

// ==================== 3.3 生活瞬间 API ====================

#[derive(Debug, Serialize)]
pub struct CreateMomentData {
    pub id: String,
}

#[derive(Debug, Serialize)]
pub struct NearbyMomentItem {
    #[serde(flatten)]
    pub moment_info: MomentInfo,
    pub distance: i32, // 米
}

#[derive(Debug, Serialize)]
pub struct MomentDetailData {
    pub id: String,
    pub user: UserInfo,
    pub description: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub images: Vec<String>,
    pub tags: Vec<String>,
    pub location: LocationInfo,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub updated_at: Option<String>,
    pub likes_count: i32,
    pub comments_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub liked_by_me: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub favorited_by_me: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<i32>,
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
    pub private_chat_id: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub id: String,
    pub content: String,
    pub r#type: String, // 'text' | 'image' | 'video' | 'audio' | 'file'
    pub sender_id: String,
    pub sent_at: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateChatListItem {
    pub private_chat_id: String,
    pub other_user: UserInfo,
    pub unread_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
}

#[derive(Debug, Serialize)]
pub struct PrivateChatDetailData {
    pub private_chat_id: String,
    pub other_user: UserInfo,
    pub other_user_status: OtherUserStatus,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_archived: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct OtherUserStatus {
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<String>,
}

// ==================== 3.6 群组管理 API ====================

#[derive(Debug, Serialize)]
pub struct CreateGroupChatData {
    pub group_chat_id: String,
}

#[derive(Debug, Serialize)]
pub struct DiscoveredGroupItem {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub member_count: i32,
    pub is_member: bool,
    pub location_name: String,
    pub distance: i32, // 米
}

#[derive(Debug, Serialize)]
pub struct JoinedGroupItem {
    #[serde(flatten)]
    pub group_info: GroupInfo,
    pub my_role: String, // 'owner' | 'admin' | 'member'
    pub unread_count: i32,
    pub last_activity_at: String,
}

#[derive(Debug, Serialize)]
pub struct GroupDetailData {
    pub group_chat_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub member_count: i32,
    pub my_role: String, // 'owner' | 'admin' | 'member'
    pub settings: GroupSettings,
    pub created_at: String,
    pub is_member: bool,
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
    pub content: String,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<String>,
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
    pub content: String,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<String>>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<String>,
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
    pub friend_request_enabled: bool,
    pub friend_accepted_enabled: bool,
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

// ==================== 3.13 通知中心 API ====================

#[derive(Debug, Serialize)]
pub struct InboxItemData {
    pub id: String,
    pub category: String, // 'social' | 'system' | 'content'
    pub r#type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub severity: Option<String>, // 'info' | 'warning' | 'error'
    pub created_at: String,
    pub is_read: bool,
}

#[derive(Debug, Serialize)]
pub struct UnreadInboxSummaryData {
    pub total: i32,
    pub by_category: UnreadCategoryStats,
}

#[derive(Debug, Serialize)]
pub struct UnreadCategoryStats {
    pub social: i32,
    pub system: i32,
    pub content: i32,
}

#[derive(Debug, Serialize)]
pub struct MarkInboxReadData {
    pub marked_count: i32,
}

// ==================== 3.14 系统健康检查 API ====================

#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: String, // 'healthy' | 'degraded' | 'unhealthy'
    pub version: String,
    pub timestamp: String,
}

// ==================== 3.15 排行榜 API ====================

#[derive(Debug, Serialize)]
pub struct GetCreatorLeaderboardData {
    pub items: Vec<CreatorRankItem>,
}

#[derive(Debug, Serialize)]
pub struct CreatorRankItem {
    pub rank: i32,
    pub user: UserInfo,
    pub score: i32, // 注意文档中写的是 number，但在排行榜中分数通常是整数或浮点。文档2.8节提到 distance 等是 number，这里用 i32 符合通常排行榜积分
}

#[derive(Debug, Serialize)]
pub struct GetMomentLeaderboardData {
    pub items: Vec<MomentRankItem>,
}

#[derive(Debug, Serialize)]
pub struct MomentRankItem {
    pub rank: i32,
    pub moment_id: String,
    pub score: i32,
}

#[derive(Debug, Serialize)]
pub struct MyCreatorRankData {
    pub timeframe: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scope: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_rank: Option<i32>,
    pub my_score: i32,
    pub total_participants: i32,
    pub top_3: Vec<CreatorRankItem>,
}

#[derive(Debug, Serialize)]
pub struct MyMomentRankData {
    pub timeframe: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scope: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_rank: Option<i32>,
    pub my_score: i32,
    pub total_participants: i32,
    pub top_3: Vec<MomentRankItem>,
}

// ==================== 3.16 文件与媒体上传 API ====================

#[derive(Debug, Serialize)]
pub struct UploadMediaData {
    pub url: String,
    pub file_size: i64,
    pub mime_type: String,
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

// ==================== 3.17 信息流（Feed）API ====================

#[derive(Debug, Serialize)]
pub struct FeedItem {
    pub item_id: String,
    pub r#type: String, // 'moment' | 'group_card' | 'system'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub moment: Option<MomentInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group: Option<GroupInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub system: Option<FeedSystemInfo>,
    pub reason: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub score: Option<f64>,
}

#[derive(Debug, Serialize)]
pub struct FeedSystemInfo {
    pub title: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub severity: Option<String>, // 'info' | 'warning' | 'error'
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    pub created_at: String,
}

// ==================== 3.18 批量操作 API ====================

#[derive(Debug, Serialize)]
pub struct BatchResultData {
    pub total: i32,
    pub succeeded: i32,
    pub failed: i32,
    pub results: Vec<BatchItemResult>,
}

#[derive(Debug, Serialize)]
pub struct BatchItemResult {
    pub index: i32,
    pub code: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
}

// ==================== 3.19 标签管理 API ====================

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
    pub items: Vec<String>,
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
    pub changed_by: String,
    pub changed_at: String,
}
