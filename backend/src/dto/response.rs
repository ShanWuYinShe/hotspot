use chrono::{DateTime, Utc};
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

    pub fn success_data(data: T) -> Self {
        Self::success(data)
    }

    pub fn error(code: i32, message: String) -> Self {
        Self {
            code,
            message,
            data: None,
        }
    }
}

/// 分页响应
#[derive(Debug, Serialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

// ==================== 认证模块响应 ====================

#[derive(Debug, Serialize)]
pub struct AuthResponse {
    pub user_id: String,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct SessionResponse {
    pub id: String,
    pub device_type: Option<String>,
    pub device_name: Option<String>,
    pub ip_address: Option<String>,
    pub location: Option<String>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    pub is_current: bool,
}

// ==================== 用户模块响应 ====================

#[derive(Debug, Serialize, Clone)]
pub struct UserResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct UserProfileResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relationship: Option<UserRelationshipInfo>,
}

#[derive(Debug, Serialize)]
pub struct UserRelationshipInfo {
    pub status: String, // 'none' | 'friend' | 'pending_sent' | 'pending_received' | 'following' | 'follower' | 'blocked' | 'blocked_by'
    pub can_message: bool,
    pub can_view_moments: bool,
}

#[derive(Debug, Serialize)]
pub struct NearbyUserResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
    pub distance: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct FriendResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<DateTime<Utc>>,
    pub relation_id: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct BlockedUserResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub relation_id: String,
    pub blocked_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ==================== 群组模块响应 ====================

#[derive(Debug, Serialize)]
pub struct GroupResponse {
    pub id: String,
    pub group_type: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub owner_id: String,
    pub owner_name: String,
    pub location_name: String,
    pub member_count: i32,
    pub is_password_required: bool,
    pub created_at: DateTime<Utc>,
    pub is_member: bool,
}

#[derive(Debug, Serialize)]
pub struct GroupDetailResponse {
    pub id: String,
    pub group_type: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub owner_id: String,
    pub owner_name: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub member_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
    pub is_password_required: bool,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_role: Option<String>,
    pub is_member: bool,
}

#[derive(Debug, Serialize)]
pub struct NearbyGroupResponse {
    pub id: String,
    pub group_type: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub owner_name: String,
    pub location_name: String,
    pub member_count: i32,
    pub is_password_required: bool,
    pub distance: f64,
    pub is_member: bool,
}

#[derive(Debug, Serialize)]
pub struct MemberResponse {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub role: String,
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<DateTime<Utc>>,
    pub joined_at: DateTime<Utc>,
    pub is_muted: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub muted_until: Option<DateTime<Utc>>,
    pub is_banned: bool,
}

// ==================== 消息模块响应 ====================

#[derive(Debug, Serialize)]
pub struct MessageResponse {
    pub id: String,
    pub target_id: String,
    pub target_type: String,
    pub message_type: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender: Option<UserResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<ReplyInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mentions: Option<Vec<String>>,
    pub is_edited: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<DateTime<Utc>>,
    pub sent_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reactions: Option<Vec<ReactionSummary>>,
    // 位置消息字段
    #[serde(skip_serializing_if = "Option::is_none")]
    pub latitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub longitude: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location_name: Option<String>,
    // 文件/媒体消息字段
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<i32>,
    // 系统消息字段
    #[serde(skip_serializing_if = "Option::is_none")]
    pub system_event: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub affected_users: Option<Vec<String>>,
}

#[derive(Debug, Serialize)]
pub struct ReactionSummary {
    pub reaction_type: String,
    pub count: i32,
    pub reacted_by_me: bool,
}

#[derive(Debug, Serialize)]
pub struct ReplyInfo {
    pub id: String,
    pub content: String,
    pub sender_name: String,
}

// ==================== 私聊模块响应 ====================

#[derive(Debug, Serialize)]
pub struct ChatResponse {
    pub id: String,
    pub other_user: ChatOtherUserInfo,
    pub unread_count: i32,
    pub muted: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<ChatLastMessageInfo>,
    pub last_activity_at: DateTime<Utc>,
}

#[derive(Debug, Serialize)]
pub struct ChatOtherUserInfo {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub is_online: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_seen: Option<DateTime<Utc>>,
}

#[derive(Debug, Serialize)]
pub struct ChatLastMessageInfo {
    pub content: String,
    pub message_type: String, // 'text' | 'image' | 'video' | 'audio' | 'file' | 'location'
    pub sent_at: DateTime<Utc>,
}

// ==================== 其他模块响应（框架） ====================

#[derive(Debug, Serialize)]
pub struct MomentResponse {
    pub id: String,
    pub user: UserResponse,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<String>>,
    pub tags: Vec<String>,
    pub location: LocationInfo,
    pub created_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub updated_at: Option<DateTime<Utc>>,
    pub likes_count: i32,
    pub comments_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub liked_by_me: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub favorited_by_me: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
}

/// 瞬间摘要（用于列表场景，如收藏列表、用户发布历史）
/// 相比 MomentResponse，移除了：
/// - tags（详情页再展示）
/// - location 完整坐标（只保留名称）
/// - updated_at（非关键信息）
/// - liked_by_me/favorited_by_me（避免额外 JOIN 查询，提升性能）
/// - distance（仅位置搜索需要）
#[derive(Debug, Serialize)]
pub struct MomentSummaryResponse {
    pub id: String,
    pub user: UserResponse,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<String>>,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
    pub likes_count: i32,
    pub comments_count: i32,
}

#[derive(Debug, Serialize)]
pub struct LocationInfo {
    pub latitude: f64,
    pub longitude: f64,
    pub name: String,
}

#[derive(Debug, Serialize)]
pub struct CommentResponse {
    pub id: String,
    pub moment_id: String,
    pub user: UserResponse,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to_user: Option<UserResponse>,
    pub created_at: DateTime<Utc>,
    pub likes_count: i32,
    pub replies_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub liked_by_me: Option<bool>,
}

// ==================== 收件箱模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct InboxItemResponse {
    pub id: String,
    pub category: String,
    pub item_type: String,
    pub title: Option<String>,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub action_url: Option<String>,
    pub severity: Option<String>,
    pub created_at: String,
    pub is_read: bool,
}

#[derive(Debug, Serialize)]
pub struct UpdateDeviceTokenResponse {
    pub success: bool,
    pub message: String,
}

// ==================== 举报模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct ReportResponse {
    pub id: String,
    pub reporter_id: String,
    pub reporter_nickname: Option<String>,
    pub target_type: String,
    pub target_id: String,
    pub reason: String,
    pub description: Option<String>,
    pub status: String,
    pub created_at: String,
    pub handled_at: Option<String>,
    pub handler_id: Option<String>,
    pub handler_note: Option<String>,
}

// ==================== 订阅模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct SubscriptionResponse {
    pub friend_request_enabled: bool,
    pub friend_accepted_enabled: bool,
    pub group_invite_enabled: bool,
    pub group_invite_response_enabled: bool,
    pub message_mention_enabled: bool,
    pub moment_like_enabled: bool,
    pub moment_comment_enabled: bool,
    pub comment_reply_enabled: bool,
    pub system_announcement_enabled: bool,
}

#[derive(Debug, Serialize)]
pub struct UpdateSubscriptionsResponse {
    pub success: bool,
    pub message: String,
}

// ==================== 排行榜模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct LeaderboardEntry {
    pub rank: i32,
    pub user_id: String,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub score: f64,
    pub stats: Option<serde_json::Value>,
    pub distance: Option<f64>,
}

#[derive(Debug, Serialize)]
pub struct LeaderboardResponse {
    pub category: String,
    pub timeframe: String,
    pub scope: Option<String>,
    pub items: Vec<LeaderboardEntry>,
    pub has_more: bool,
    pub next_cursor: Option<String>,
}

// ==================== 信息流模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct FeedItemResponse {
    pub item_id: String,
    pub content_type: String,
    pub content_id: String,
    pub title: Option<String>,
    pub description: String,
    pub author_id: i64,
    pub author_name: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub score: f64,
}

// ==================== 媒体模块响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct UploadResponse {
    pub url: String,
    pub file_size: i64,
    pub mime_type: String,
}

// ==================== Auth 补充响应 ====================

#[derive(Debug, Serialize)]
pub struct RequestPasswordResetResponse {
    pub success: bool,
    pub message: String,
    pub reset_id: String,
    pub expires_in: i32,
}

#[derive(Debug, Serialize)]
pub struct VerifyResetCodeResponse {
    pub success: bool,
    pub message: String,
    pub reset_token: String,
}

#[derive(Debug, Serialize)]
pub struct ResetPasswordResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct CheckStatusResponse {
    pub user_id: String,
    pub nickname: String,
    pub is_valid: bool,
    pub expires_at: String,
}

#[derive(Debug, Serialize)]
pub struct SetPasswordResponse {
    pub success: bool,
    pub message: String,
    pub can_login: bool,
}

#[derive(Debug, Serialize)]
pub struct RevokeSessionResponse {
    pub success: bool,
    pub message: String,
}

// ==================== User 补充响应 ====================

#[derive(Debug, Serialize)]
pub struct GetMeResponse {
    pub profile: GetMeProfileResponse,
    pub stats: GetMeStatsResponse,
    pub unread: GetMeUnreadResponse,
    pub settings: GetSettingsResponse,
}

#[derive(Debug, Serialize)]
pub struct GetMeProfileResponse {
    pub user_id: String,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Serialize)]
pub struct GetMeStatsResponse {
    pub moments_count: i32,
    pub friends_count: i32,
    pub followers_count: i32,
    pub following_count: i32,
    pub groups_count: i32,
}

#[derive(Debug, Serialize)]
pub struct GetMeUnreadResponse {
    pub messages: i32,
    pub friend_requests: i32,
    pub group_invites: i32,
    pub notifications: i32,
}

#[derive(Debug, Serialize)]
pub struct GetSettingsResponse {
    pub privacy_level: String,
    pub location_sharing: bool,
    pub allow_follow: bool,
}

#[derive(Debug, Serialize)]
pub struct UpdateSettingsResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DeleteAccountResponse {
    pub success: bool,
    pub message: String,
}

// ==================== Group 补充响应 ====================

#[derive(Debug, Serialize)]
pub struct UpdateGroupProfileResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct UpdateGroupLocationResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct UpdatePrivacyResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct UpdateNotificationsResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct InviteMembersResponse {
    pub success: bool,
    pub message: String,
    pub results: Vec<InviteResult>,
}

#[derive(Debug, Serialize)]
pub struct InviteResult {
    pub user_id: String,
    pub status: String, // "sent" or "failed"
    pub reason: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct AcceptInviteResponse {
    pub success: bool,
    pub message: String,
    pub group: GroupInfo,
}

#[derive(Debug, Serialize)]
pub struct GroupInfo {
    pub id: String,
    pub group_type: String,
    pub name: String,
}

#[derive(Debug, Serialize)]
pub struct RejectInviteResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct GroupInviteResponse {
    pub id: String,
    pub group: GroupInfo,
    pub inviter: UserResponse,
    pub message: Option<String>,
    pub status: String,
    pub created_at: String,
    pub responded_at: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct ApplyToGroupResponse {
    pub success: bool,
    pub message: String,
    pub application_id: String,
}

#[derive(Debug, Serialize)]
pub struct ReviewApplicationResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct MyGroupResponse {
    pub id: String,
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub member_count: i32,
    pub my_role: String,
    pub unread_count: i32,
    pub notifications_enabled: bool,
    pub mention_only: bool,
    pub joined_at: String,
    pub last_active_at: String,
    pub last_message: Option<LastMessageInfo>,
}

#[derive(Debug, Serialize)]
pub struct LastMessageInfo {
    pub content: String,
    pub sender_name: String,
    pub sent_at: String,
}

// ==================== Leaderboard 响应 ====================

#[derive(Debug, Serialize)]
pub struct CreatorRankResponse {
    pub rank: i32,
    pub user: UserResponse,
    pub score: f64,
    pub stats: CreatorStats,
    pub distance: Option<f64>,
}

#[derive(Debug, Serialize)]
pub struct CreatorStats {
    pub moments_count: i32,
    pub total_likes: i32,
    pub total_comments: i32,
    pub followers_count: i32,
}

#[derive(Debug, Serialize)]
pub struct MomentRankResponse {
    pub rank: i32,
    pub moment: MomentResponse,
    pub score: f64,
}

#[derive(Debug, Serialize)]
pub struct GetMyRankResponse {
    pub category: String,
    pub timeframe: String,
    pub scope: Option<String>,
    pub my_rank: Option<i32>,
    pub my_score: f64,
    pub total_participants: i32,
    pub top_3: Vec<serde_json::Value>,
}

// ==================== 私聊相关响应（补全） ====================

#[derive(Debug, Serialize)]
pub struct StartChatResponse {
    pub success: bool,
    pub message: String,
    pub room_id: String,
}

#[derive(Debug, Serialize)]
pub struct GetChatSettingsResponse {
    pub muted: bool,
    pub pinned: bool,
}

#[derive(Debug, Serialize)]
pub struct UpdateChatSettingsResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct DeleteChatResponse {
    pub success: bool,
    pub message: String,
}

#[derive(Debug, Serialize)]
pub struct UpgradeToGroupResponse {
    pub success: bool,
    pub message: String,
    pub room_id: String,
    pub group_type: String,
    pub name: String,
}
