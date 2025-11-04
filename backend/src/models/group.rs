use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

use crate::types::RoomType;

/// 群组模型（映射到 chat_rooms 表，只包含 public_group 和 private_group）
#[derive(Debug, Clone, FromRow)]
pub struct Group {
    #[sqlx(default)]
    pub id: Option<i64>,
    
    // 数据库使用 room_id，但在应用层使用 group_id
    #[sqlx(rename = "room_id")]
    pub group_id: Uuid,
    
    // 数据库使用 room_type，但在应用层使用 group_type
    #[sqlx(rename = "room_type")]
    pub group_type: RoomType,  // ✨ 使用枚举类型
    
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub password_hash: Option<String>,
    pub max_members: Option<i32>,
    
    // 位置信息（群组必需）
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    
    pub created_at: DateTime<Utc>,
    
    // 数据库使用 last_message_at，但在应用层使用 last_active_at
    #[sqlx(rename = "last_message_at")]
    pub last_active_at: Option<DateTime<Utc>>,
    
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

