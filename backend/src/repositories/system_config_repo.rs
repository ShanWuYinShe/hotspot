use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 系统配置Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct SystemConfigRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub key: String,
    pub value: String,
    pub config_type: Option<String>,  // 使用 Option 以兼容 sqlx 宏
    pub category: Option<String>,     // 使用 Option 以兼容 sqlx 宏
    pub description: Option<String>,  // description 在数据库中是 TEXT，可能为 NULL
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

/// 配置变更历史Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct ConfigChangeLogRow {
    pub id: i64,
    pub cursor_id: Uuid,
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

    /// 获取配置项
    pub async fn find_by_key(&self, key: &str) -> Result<Option<SystemConfigRow>> {
        let row = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT id, cursor_id, key, value, type as config_type, category, description, created_at, updated_at
            FROM system_configs
            WHERE key = $1 AND is_deleted = FALSE
            "#,
            key
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 创建配置项
    pub async fn create(
        &self,
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
            RETURNING id, cursor_id, key, value, type as config_type, category, description, created_at, updated_at
            "#,
            key,
            value,
            description,
            config_type,
            category
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新配置项
    pub async fn update(&self, key: &str, value: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE system_configs
            SET value = $2, updated_at = NOW()
            WHERE key = $1
            "#,
            key,
            value
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除配置项
    pub async fn delete(&self, key: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE system_configs
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE key = $1
            "#,
            key
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 记录配置变更历史
    pub async fn log_change(
        &self,
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
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 将 UUID cursor 转换为自增主键（配置表）
    async fn resolve_config_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM system_configs WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 将 UUID cursor 转换为自增主键（历史表）
    async fn resolve_history_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM system_config_history WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 获取配置变更历史
    pub async fn get_change_history(&self, key: &str, limit: i32, cursor: Option<Uuid>) -> Result<Vec<ConfigChangeLogRow>> {
        let cursor_pk = self.resolve_history_cursor(cursor).await?;
        
        let rows = sqlx::query_as!(
            ConfigChangeLogRow,
            r#"
            SELECT id, cursor_id, config_key as key, old_value, new_value, changed_by, changed_at
            FROM system_config_history
            WHERE config_key = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            key,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }

    /// 获取所有配置变更
    pub async fn list_all_changes(
        &self, 
        start_time: Option<&str>,
        end_time: Option<&str>,
        changed_by: Option<i64>,
        limit: i32, 
        cursor: Option<Uuid>
    ) -> Result<Vec<ConfigChangeLogRow>> {
        let cursor_pk = self.resolve_history_cursor(cursor).await?;
        
        // 解析时间字符串为 DateTime
        let start_dt = start_time.and_then(|s| chrono::DateTime::parse_from_rfc3339(s).ok())
            .map(|dt| dt.with_timezone(&chrono::Utc));
        let end_dt = end_time.and_then(|s| chrono::DateTime::parse_from_rfc3339(s).ok())
            .map(|dt| dt.with_timezone(&chrono::Utc));
        
        let rows = sqlx::query_as!(
            ConfigChangeLogRow,
            r#"
            SELECT id, cursor_id, config_key as key, old_value, new_value, changed_by, changed_at
            FROM system_config_history
            WHERE id < $1
              AND ($2::timestamptz IS NULL OR changed_at >= $2)
              AND ($3::timestamptz IS NULL OR changed_at <= $3)
              AND ($4::bigint IS NULL OR changed_by = $4)
            ORDER BY id DESC
            LIMIT $5
            "#,
            cursor_pk,
            start_dt,
            end_dt,
            changed_by,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }

    /// 按分类列出配置
    pub async fn list(&self, category: Option<String>, limit: i32, cursor: Option<Uuid>) -> Result<Vec<SystemConfigRow>> {
        let cursor_pk = self.resolve_config_cursor(cursor).await?;
        
        let rows = sqlx::query_as!(
            SystemConfigRow,
            r#"
            SELECT id, cursor_id, key, value, type as config_type, category, description, created_at, updated_at
            FROM system_configs
            WHERE ($1::text IS NULL OR category = $1)
              AND id < $2 
              AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            category,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }
}
