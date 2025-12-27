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

/// 配置变更历史Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ConfigChangeLogRow {
    pub id: i64,
    pub key: String,
    pub old_value: Option<String>,
    pub new_value: String,
    pub changed_by: i64,
    pub changed_at: DateTime<Utc>,
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

    /// 删除配置项（事务版）
    pub async fn delete_tx(&self, tx: &mut Transaction<'_, Postgres>, key: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE system_configs
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE key = $1
            "#,
            key
        )
        .execute(&mut **tx)
        .await?;

        Ok(())
    }

    /// 记录配置变更历史（事务版）
    pub async fn log_change_tx(
        &self,
        tx: &mut Transaction<'_, Postgres>,
        key: &str,
        old_value: Option<String>,
        new_value: String,
        changed_by: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO system_config_history (config_key, old_value, new_value, changed_by)
            VALUES ($1, $2, $3, $4)
            "#,
            key,
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

    /// 获取配置变更历史
    pub async fn get_change_history(
        &self,
        key: String,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<ConfigChangeLogRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            ConfigChangeLogRow,
            r#"
            SELECT id, config_key as key, old_value, new_value, changed_by, changed_at
            FROM system_config_history
            WHERE config_key = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            key,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((rows, next_cursor))
    }

    /// 获取所有配置变更历史
    pub async fn list_all_changes(
        &self,
        start_time: Option<DateTime<Utc>>,
        end_time: Option<DateTime<Utc>>,
        changed_by: Option<i64>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<ConfigChangeLogRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            ConfigChangeLogRow,
            r#"
            SELECT id, config_key as key, old_value, new_value, changed_by, changed_at
            FROM system_config_history
            WHERE ($1::timestamptz IS NULL OR changed_at >= $1)
              AND ($2::timestamptz IS NULL OR changed_at <= $2)
              AND ($3::bigint IS NULL OR changed_by = $3)
              AND id < $4
            ORDER BY id DESC
            LIMIT $5
            "#,
            start_time,
            end_time,
            changed_by,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.id)).transpose()?;
        Ok((rows, next_cursor))
    }
}
