use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

use crate::types::MemberRole;

/// 群组成员模型（映射到 room_members 表）
#[derive(Debug, Clone, FromRow)]
pub struct GroupMember {
    pub id: i64,
    
    // 数据库使用 room_id，但在应用层使用 group_id
    #[sqlx(rename = "room_id")]
    pub group_id: Uuid,
    
    pub user_id: i64,
    pub role: MemberRole,  // ✨ 使用枚举类型
    pub nickname: Option<String>,
    pub is_banned: bool,
    pub is_muted: bool,
    pub muted_until: Option<DateTime<Utc>>,
    pub joined_at: DateTime<Utc>,
    pub unread_count: i32,
    pub last_read_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

