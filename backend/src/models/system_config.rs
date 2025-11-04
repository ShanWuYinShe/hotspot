use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SystemConfig {
    pub id: i64,
    pub config_key: String,
    pub config_value: String,
    pub value_type: String,
    pub description: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSystemConfigRequest {
    pub config_key: String,
    pub config_value: String,
    pub value_type: String,
    pub description: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSystemConfigRequest {
    pub config_value: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemConfigResponse {
    pub config_key: String,
    pub config_value: String,
    pub value_type: String,
    pub description: Option<String>,
    pub updated_at: String,
}

impl From<SystemConfig> for SystemConfigResponse {
    fn from(config: SystemConfig) -> Self {
        Self {
            config_key: config.config_key,
            config_value: config.config_value,
            value_type: config.value_type,
            description: config.description,
            updated_at: config.updated_at.to_rfc3339(),
        }
    }
}











