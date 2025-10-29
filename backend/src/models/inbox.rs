use chrono::NaiveDateTime;
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct InboxItem {
    pub id: i64,
    pub user_id: i64,
    pub category: String,      // system, social, message, group
    pub item_type: String,      // friend_request, comment, like, mention, etc.
    pub data: serde_json::Value, // 通知数据（JSON格式）
    pub is_read: bool,
    pub created_at: NaiveDateTime,
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct DeviceToken {
    pub id: i64,
    pub user_id: i64,
    pub token: String,
    pub platform: String, // ios, android, web
    pub created_at: NaiveDateTime,
}

