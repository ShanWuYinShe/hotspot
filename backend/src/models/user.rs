use chrono::{DateTime, Utc};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow)]
pub struct User {
    pub user_id: i64,
    pub login_id: Option<String>,
    pub nickname: String,
    pub password_hash: Option<String>,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: Option<DateTime<Utc>>,
}

impl Default for User {
    fn default() -> Self {
        Self {
            user_id: 0,
            login_id: None,
            nickname: String::new(),
            password_hash: None,
            avatar_url: None,
            bio: None,
            latitude: None,
            longitude: None,
            location_name: None,
            created_at: Utc::now(),
            last_active_at: None,
        }
    }
}

