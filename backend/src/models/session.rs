use chrono::{DateTime, Utc};
use sqlx::FromRow;
use sqlx::types::ipnetwork::IpNetwork;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct Session {
    pub id: i64,                        // 自增主键
    pub session_id: Uuid,               // 会话UUID
    pub user_id: i64,
    pub device_type: Option<String>,
    pub device_name: Option<String>,
    pub ip_address: Option<IpNetwork>,  // 使用IpNetwork类型
    pub location: Option<String>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

