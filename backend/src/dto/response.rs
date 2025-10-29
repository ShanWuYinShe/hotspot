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
}

#[derive(Debug, Serialize)]
pub struct ReplyInfo {
    pub id: String,
    pub content: String,
    pub sender_name: String,
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

