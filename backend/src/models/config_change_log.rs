use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct ConfigChangeLog {
    pub id: i64,
    pub config_key: String,
    pub old_value: Option<String>,
    pub new_value: String,
    pub changed_by: i64,
    pub changed_at: DateTime<Utc>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateConfigChangeLogRequest {
    pub config_key: String,
    pub old_value: Option<String>,
    pub new_value: String,
    pub changed_by: i64,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigChangeLogResponse {
    pub id: String,
    pub config_key: String,
    pub old_value: Option<String>,
    pub new_value: String,
    pub changed_by: String,
    pub changed_at: String,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

impl From<ConfigChangeLog> for ConfigChangeLogResponse {
    fn from(log: ConfigChangeLog) -> Self {
        Self {
            id: log.id.to_string(),
            config_key: log.config_key,
            old_value: log.old_value,
            new_value: log.new_value,
            changed_by: log.changed_by.to_string(),
            changed_at: log.changed_at.to_rfc3339(),
            ip_address: log.ip_address,
            user_agent: log.user_agent,
        }
    }
}











