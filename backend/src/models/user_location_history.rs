use chrono::{DateTime, Utc};
use sqlx::FromRow;

/// 用户位置历史模型
#[derive(Debug, Clone, FromRow)]
pub struct UserLocationHistory {
    pub id: i64,
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    // 跳过 location 字段（geography 类型，sqlx 不支持）
    // 位置查询将在 SQL 层面使用 ST_Distance 处理
    pub created_at: DateTime<Utc>,
}









