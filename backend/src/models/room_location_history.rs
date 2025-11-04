use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;

/// 聊天室位置历史模型（包括公开群组和私密群组）
#[derive(Debug, Clone, FromRow)]
pub struct RoomLocationHistory {
    pub id: i64,
    pub room_id: Uuid,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    // 跳过 location 字段（geography 类型，sqlx 不支持）
    // 位置查询将在 SQL 层面使用 ST_Distance 处理
    pub created_at: DateTime<Utc>,
}









