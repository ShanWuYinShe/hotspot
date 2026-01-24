use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool, Postgres, Transaction};

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// 系统配置Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct SystemConfigRow {
    pub id: i64,
    pub key: String,
    pub value: String,
    pub config_type: Option<String>, // 使用 Option 以兼容 sqlx 宏
    pub category: Option<String>,    // 使用 Option 以兼容 sqlx 宏
    pub description: Option<String>, // description 在数据库中是 TEXT，可能为 NULL
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct SystemConfigRepository {
    pool: PgPool,
}

impl SystemConfigRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    /// 获取配置项
    pub async fn find_by_key(&self, key: &str) -> Result<Option<SystemConfigRow>> {
        let row = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT id, key, value, type as config_type, category, description, created_at, updated_at
            FROM system_configs
            WHERE key = $1 AND is_deleted = FALSE
            "#,
            key
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 创建配置项（事务版）
    pub async fn create_tx(
        &self,
        tx: &mut Transaction<'_, Postgres>,
        key: String,
        value: String,
        description: Option<String>,
        config_type: String,
        category: String,
    ) -> Result<SystemConfigRow> {
        let row = sqlx::query_as!(
            SystemConfigRow,
            r#"
            INSERT INTO system_configs (key, value, description, type, category)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id, key, value, type as config_type, category, description, created_at, updated_at
            "#,
            key,
            value,
            description,
            config_type,
            category
        )
        .fetch_one(&mut **tx)
        .await?;

        Ok(row)
    }

    /// 更新配置项（事务版）
    pub async fn update_tx(
        &self,
        tx: &mut Transaction<'_, Postgres>,
        key: &str,
        value: String,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE system_configs
            SET value = $2, updated_at = NOW()
            WHERE key = $1
            "#,
            key,
            value
        )
        .execute(&mut **tx)
        .await?;

        Ok(())
    }

    /// 记录配置变更历史（事务版）
    pub async fn insert_history_tx(
        &self,
        tx: &mut Transaction<'_, Postgres>,
        config_key: &str,
        old_value: Option<String>,
        new_value: &str,
        changed_by: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO system_config_history (config_key, old_value, new_value, changed_by)
            VALUES ($1, $2, $3, $4)
            "#,
            config_key,
            old_value,
            new_value,
            changed_by
        )
        .execute(&mut **tx)
        .await?;

        Ok(())
    }

    /// 按分类列出配置
    pub async fn list(
        &self,
        category: Option<String>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<SystemConfigRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT id, key, value, type as config_type, category, description, created_at, updated_at
            FROM system_configs
            WHERE ($1::text IS NULL OR category = $1)
              AND id < $2 
              AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            category,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((rows, next_cursor))
    }
}
