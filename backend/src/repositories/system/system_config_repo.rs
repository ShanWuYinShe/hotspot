use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct SystemConfigRow {
    pub config_key: String,
    pub config_value: serde_json::Value,
    pub description: Option<String>,
    pub is_public: bool,
    pub updated_at: DateTime<Utc>,
    pub updated_by: Option<i64>,
}

#[derive(Debug, Clone)]
pub struct SystemConfigRepository {
    pool: PgPool,
}

impl SystemConfigRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn find_by_key(&self, key: &str) -> Result<Option<SystemConfigRow>> {
        let row = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT config_key, config_value, description,
                   COALESCE(is_public, false) as "is_public!",
                   updated_at, updated_by
            FROM system_configs
            WHERE config_key = $1
            "#,
            key
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn upsert(
        &self,
        key: String,
        value: serde_json::Value,
        description: Option<String>,
        is_public: bool,
        updated_by: Option<i64>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO system_configs (config_key, config_value, description, is_public, updated_by)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (config_key) DO UPDATE
            SET config_value = EXCLUDED.config_value,
                description = EXCLUDED.description,
                is_public = EXCLUDED.is_public,
                updated_by = EXCLUDED.updated_by,
                updated_at = NOW()
            "#,
            key,
            value,
            description,
            is_public,
            updated_by
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn list_all(&self) -> Result<Vec<SystemConfigRow>> {
        let rows = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT config_key, config_value, description,
                   COALESCE(is_public, false) as "is_public!",
                   updated_at, updated_by
            FROM system_configs
            "#
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }
}
