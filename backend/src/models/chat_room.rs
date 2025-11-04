use chrono::{DateTime, Utc};
use serde::Serialize;
use sqlx::FromRow;
use uuid::Uuid;

use crate::types::enums::{RoomType, MemberRole, InviteStatus};

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct ChatRoom {
    #[serde(skip_serializing)]
    pub id: i64,
    pub room_id: Uuid,
    pub room_type: RoomType, // 使用ENUM类型
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub password_hash: Option<String>,
    pub max_members: i32,
    pub is_deleted: bool,                    // 软删除标志
    pub deleted_at: Option<DateTime<Utc>>,  // 删除时间
    pub created_at: DateTime<Utc>,
    pub last_message_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct RoomMember {
    pub id: i64,
    pub room_id: Uuid,
    pub user_id: i64,
    pub role: MemberRole, // 使用ENUM类型
    pub nickname: Option<String>,
    pub status: String, // 'active', 'left', 'removed', 'banned' - 暂时保留String
    pub joined_at: DateTime<Utc>,
    pub left_at: Option<DateTime<Utc>>,
    pub muted_until: Option<DateTime<Utc>>,
    pub unread_count: i32,
    pub last_read_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct RoomInvite {
    #[serde(skip_serializing)]
    pub id: i64,
    pub invite_id: Uuid,
    pub room_id: Uuid,
    pub inviter_id: i64,
    pub invitee_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus, // 使用ENUM类型
    pub created_at: DateTime<Utc>,
    pub responded_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct RoomApplication {
    #[serde(skip_serializing)]
    pub id: i64,
    pub application_id: Uuid,
    pub room_id: Uuid,
    pub applicant_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus, // 使用ENUM类型（复用InviteStatus）
    pub created_at: DateTime<Utc>,
    pub reviewed_at: Option<DateTime<Utc>>,
    pub reviewer_id: Option<i64>,
    pub review_reason: Option<String>,
}

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct RoomSettings {
    pub id: i64,
    pub user_id: i64,
    pub room_id: Uuid,
    pub muted: bool,
    pub mention_only: bool,
    pub pinned: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}



