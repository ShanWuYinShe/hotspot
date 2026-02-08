use crate::types::enums::{UserGender, UserStatus, UserType};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct User {
    pub user_id: i64,
    pub username: String,
    pub email: Option<String>,
    pub phone: Option<String>,
    #[serde(skip)]
    pub password_hash: String,

    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub gender: Option<UserGender>,
    pub birthday: Option<chrono::NaiveDate>,

    pub user_type: UserType,
    pub status: UserStatus,

    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserStats {
    pub user_id: i64,
    pub moments_count: i32,
    pub following_count: i32,
    pub followers_count: i32,
    pub likes_received: i32,
    pub groups_count: i32,
    pub reactions_given: i32,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserSettings {
    pub user_id: i64,
    pub language: Option<String>,
    pub theme: Option<String>,
    pub notifications_enabled: Option<bool>,
    pub privacy_settings: Option<serde_json::Value>,
    pub preferences: Option<serde_json::Value>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserOnlineStatus {
    pub user_id: i64,
    pub is_online: bool,
    pub last_active_at: Option<DateTime<Utc>>,
    pub connection_count: i32,
    pub client_ip: Option<::ipnetwork::IpNetwork>,
    pub device_info: Option<serde_json::Value>,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Follow {
    pub follower_id: i64,
    pub following_id: i64,
    pub created_at: DateTime<Utc>,
}
